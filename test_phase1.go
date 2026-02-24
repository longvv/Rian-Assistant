// Test script to verify PicoClaw Phase 1 Architecture (Gateway -> Orchestrator -> Worker)
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func main() {
	cfg, err := config.LoadConfig(os.ExpandEnv("$HOME/.picoclaw/config.json"))
	if err != nil {
		panic(err)
	}

	provider, _, err := providers.CreateProvider(cfg)
	if err != nil {
		panic(err)
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agentLoop.Run(ctx)

	// Start a goroutine to print outbound messages
	go func() {
		for {
			msg, ok := msgBus.SubscribeOutbound(ctx)
			if ok {
				fmt.Printf("\n[Telegram User Receives]:\n%s\n\n", msg.Content)
				// Exit the test after receiving the final message
				os.Exit(0)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	fmt.Println("Simulating a complex user request (requires orchestration)...")

	msgBus.PublishInbound(bus.InboundMessage{
		Channel:  "telegram", // Should route to gateway
		SenderID: "test_user",
		ChatID:   "12345",
		Content:  "Calculate 5 * 10, then search the web for the population of Vietnam, then write a short combined sentence.",
	})

	// Wait for the async multi-agent flow to complete
	time.Sleep(2 * time.Minute)
	fmt.Println("Test timed out.")
	os.Exit(1)
}
