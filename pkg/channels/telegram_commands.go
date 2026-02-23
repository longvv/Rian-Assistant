package channels

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/sipeed/picoclaw/pkg/config"
)

type TelegramCommander interface {
	Help(ctx context.Context, message telego.Message) error
	Start(ctx context.Context, message telego.Message) error
	Show(ctx context.Context, message telego.Message) error
	List(ctx context.Context, message telego.Message) error
	News(ctx context.Context, message telego.Message) error
	Status(ctx context.Context, message telego.Message) error
}

type cmd struct {
	bot    *telego.Bot
	config *config.Config
}

func NewTelegramCommands(bot *telego.Bot, cfg *config.Config) TelegramCommander {
	return &cmd{
		bot:    bot,
		config: cfg,
	}
}

func commandArgs(text string) string {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func findGoExecutable() string {
	if path, err := exec.LookPath("go"); err == nil {
		return path
	}
	commonPaths := []string{
		"/opt/homebrew/bin/go",
		"/usr/local/go/bin/go",
		"/usr/local/bin/go",
		"/usr/bin/go",
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "go" // fallback
}
func (c *cmd) Help(ctx context.Context, message telego.Message) error {
	msg := `/start - Start the bot
/help - Show this help message
/show [model|channel] - Show current configuration
/list [models|channels] - List available options
/news - Fetch latest news
/status - Show current bot status
	`
	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   msg,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Start(ctx context.Context, message telego.Message) error {
	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   "Hello! I am PicoClaw 🦞",
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Show(ctx context.Context, message telego.Message) error {
	args := commandArgs(message.Text)
	if args == "" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /show [model|channel]",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	var response string
	switch args {
	case "model":
		response = fmt.Sprintf("Current Model: %s (Provider: %s)",
			c.config.Agents.Defaults.Model,
			c.config.Agents.Defaults.Provider)
	case "channel":
		response = "Current Channel: telegram"
	default:
		response = fmt.Sprintf("Unknown parameter: %s. Try 'model' or 'channel'.", args)
	}

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   response,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}
func (c *cmd) List(ctx context.Context, message telego.Message) error {
	args := commandArgs(message.Text)
	if args == "" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /list [models|channels]",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	var response string
	switch args {
	case "models":
		provider := c.config.Agents.Defaults.Provider
		if provider == "" {
			provider = "configured default"
		}
		response = fmt.Sprintf("Configured Model: %s\nProvider: %s\n\nTo change models, update config.yaml",
			c.config.Agents.Defaults.Model, provider)

	case "channels":
		var enabled []string
		if c.config.Channels.Telegram.Enabled {
			enabled = append(enabled, "telegram")
		}
		if c.config.Channels.WhatsApp.Enabled {
			enabled = append(enabled, "whatsapp")
		}
		if c.config.Channels.Feishu.Enabled {
			enabled = append(enabled, "feishu")
		}
		if c.config.Channels.Discord.Enabled {
			enabled = append(enabled, "discord")
		}
		if c.config.Channels.Slack.Enabled {
			enabled = append(enabled, "slack")
		}
		response = fmt.Sprintf("Enabled Channels:\n- %s", strings.Join(enabled, "\n- "))

	default:
		response = fmt.Sprintf("Unknown parameter: %s. Try 'models' or 'channels'.", args)
	}

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   response,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) News(ctx context.Context, message telego.Message) error {
	pMsg, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   "Đang tổng hợp tin tức... 🗞️",
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	if err != nil {
		return err
	}

	go func() {
		workspaceDir := os.Getenv("WORKSPACE")
		if workspaceDir == "" {
			workspaceDir = "workspace"
		}

		var cmd *exec.Cmd
		binaryPath := "/usr/local/bin/news_aggregator"
		if _, err := os.Stat(binaryPath); err == nil {
			cmd = exec.Command(binaryPath)
		} else {
			scriptPath := "workspace/scripts/news_aggregator.go"
			goExe := findGoExecutable()
			cmd = exec.Command(goExe, "run", scriptPath)
		}

		cmd.Env = append(os.Environ(), "WORKSPACE="+workspaceDir)

		output, err := cmd.CombinedOutput()
		if err != nil {
			msg := fmt.Sprintf("❌ Lỗi khi tải tin tức: %v\n\n%s", err, string(output))
			_, _ = c.bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:    telego.ChatID{ID: message.Chat.ID},
				MessageID: pMsg.MessageID,
				Text:      msg,
			})
			return
		}

		htmlContent := markdownToTelegramHTML(string(output))

		if len(htmlContent) <= telegramHTMLLimit {
			_, _ = c.bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:    telego.ChatID{ID: message.Chat.ID},
				MessageID: pMsg.MessageID,
				Text:      htmlContent,
				ParseMode: telego.ModeHTML,
			})
		} else {
			// Delete placeholder and send document
			_ = c.bot.DeleteMessage(context.Background(), &telego.DeleteMessageParams{
				ChatID:    telego.ChatID{ID: message.Chat.ID},
				MessageID: pMsg.MessageID,
			})

			caption := buildCaption(string(output), telegramCaptionLimit)
			htmlCaption := markdownToTelegramHTML(caption)

			docParams := tu.Document(tu.ID(message.Chat.ID), tu.FileFromBytes(output, "news_report.md"))
			docParams.Caption = htmlCaption
			docParams.ParseMode = telego.ModeHTML

			if _, err := c.bot.SendDocument(context.Background(), docParams); err != nil {
				docParams.Caption = caption
				docParams.ParseMode = ""
				_, _ = c.bot.SendDocument(context.Background(), docParams)
			}
		}
	}()

	return nil
}

func (c *cmd) Status(ctx context.Context, message telego.Message) error {
	var sb strings.Builder

	sb.WriteString("🤖 *PicoClaw Status*\n\n")
	sb.WriteString(fmt.Sprintf("• **Model:** `%s`\n", c.config.Agents.Defaults.Model))
	sb.WriteString(fmt.Sprintf("• **Provider:** `%s`\n\n", c.config.Agents.Defaults.Provider))

	sb.WriteString("📡 *Enabled Channels*\n")

	enabledChannels := []string{}
	if c.config.Channels.Telegram.Enabled {
		enabledChannels = append(enabledChannels, "Telegram")
	}
	if c.config.Channels.WhatsApp.Enabled {
		enabledChannels = append(enabledChannels, "WhatsApp")
	}
	if c.config.Channels.Feishu.Enabled {
		enabledChannels = append(enabledChannels, "Feishu")
	}
	if c.config.Channels.Discord.Enabled {
		enabledChannels = append(enabledChannels, "Discord")
	}
	if c.config.Channels.Slack.Enabled {
		enabledChannels = append(enabledChannels, "Slack")
	}
	if c.config.Channels.MaixCam.Enabled {
		enabledChannels = append(enabledChannels, "MaixCam")
	}
	if c.config.Channels.QQ.Enabled {
		enabledChannels = append(enabledChannels, "QQ")
	}
	if c.config.Channels.DingTalk.Enabled {
		enabledChannels = append(enabledChannels, "DingTalk")
	}
	if c.config.Channels.LINE.Enabled {
		enabledChannels = append(enabledChannels, "LINE")
	}
	if c.config.Channels.OneBot.Enabled {
		enabledChannels = append(enabledChannels, "OneBot")
	}

	if len(enabledChannels) == 0 {
		sb.WriteString("None")
	} else {
		for _, ch := range enabledChannels {
			sb.WriteString(fmt.Sprintf("• %s\n", ch))
		}
	}

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      sb.String(),
		ParseMode: telego.ModeMarkdown,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})

	return err
}
