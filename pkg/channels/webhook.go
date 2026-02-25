package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type WebhookChannel struct {
	*BaseChannel
	config *config.WebhookConfig
	server *http.Server
}

func NewWebhookChannel(cfg config.WebhookConfig, bus *bus.MessageBus) (*WebhookChannel, error) {
	if cfg.Port == 0 {
		cfg.Port = 8081 // default port
	}
	if cfg.Path == "" {
		cfg.Path = "/webhook"
	}

	c := &WebhookChannel{
		BaseChannel: NewBaseChannel("webhook", &cfg, bus, cfg.AllowFrom),
		config:      &cfg,
	}

	return c, nil
}

func (c *WebhookChannel) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc(c.config.Path, c.handleWebhook)

	addr := fmt.Sprintf(":%d", c.config.Port)
	c.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	c.setRunning(true)
	logger.InfoCF("webhook", "Webhook server starting", map[string]interface{}{
		"port": c.config.Port,
		"path": c.config.Path,
	})

	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("webhook", "Webhook server failed", map[string]interface{}{
				"error": err.Error(),
			})
			c.setRunning(false)
		}
	}()

	go func() {
		<-ctx.Done()
		c.Stop(context.Background())
	}()

	return nil
}

func (c *WebhookChannel) Stop(ctx context.Context) error {
	if !c.IsRunning() {
		return nil
	}

	c.setRunning(false)
	if c.server != nil {
		logger.InfoC("webhook", "Stopping webhook server...")
		return c.server.Shutdown(ctx)
	}
	return nil
}

func (c *WebhookChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	// Webhooks are typically one-way triggers. We don't send responses back to the original webhook caller
	// directly through the message bus Outbound route (unless implementing synchronous HTTP reply).
	// For autonomous logging, Gateway will typically route outbound messages back to Telegram.
	logger.DebugCF("webhook", "Ignoring outbound message as webhook is a read-only trigger channel", map[string]interface{}{
		"chat_id": msg.ChatID,
	})
	return nil
}

func (c *WebhookChannel) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Basic secret validation if configured
	if c.config.Secret != "" {
		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, "Bearer ") || strings.TrimPrefix(token, "Bearer ") != c.config.Secret {
			logger.WarnC("webhook", "Unauthorized webhook access attempt")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Optionally parse JSON to pretty-print or extract specific fields for routing
	var payload map[string]interface{}
	content := string(body)
	if err := json.Unmarshal(body, &payload); err == nil {
		// Pretty format JSON for the AI
		pretty, _ := json.MarshalIndent(payload, "", "  ")
		content = "Webhook Event Received:\n```json\n" + string(pretty) + "\n```"
	}

	metadata := map[string]string{
		"event_type": r.Header.Get("X-Event-Type"),
		"source_ip":  r.RemoteAddr,
	}

	// Assuming the "sender" is the system or webhook client
	senderID := "webhook_client"
	chatID := "webhook_default"

	if !c.IsAllowed(senderID) {
		logger.WarnCF("webhook", "Sender rejected by allowlist", map[string]interface{}{
			"sender": senderID,
		})
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Dispatch to the AI Orchestrator/Gateway
	c.HandleMessage(senderID, chatID, content, nil, metadata)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook accepted"))
}
