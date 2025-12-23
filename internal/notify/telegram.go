package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
)

const telegramAPIBase = "https://api.telegram.org"

// TelegramOptions contains options for creating a Telegram notifier
type TelegramOptions struct {
	BotToken  string
	ChatID    string
	ParseMode string
	Logger    *log.Logger
}

// TelegramNotifier sends notifications to Telegram via Bot API
type TelegramNotifier struct {
	botToken   string
	chatID     string
	parseMode  string
	httpClient *http.Client
	logger     *log.Logger
	enabled    bool
}

// Telegram sendMessage payload
type telegramPayload struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// NewTelegramNotifier creates a new Telegram notifier
func NewTelegramNotifier(opts TelegramOptions) *TelegramNotifier {
	return &TelegramNotifier{
		botToken:   opts.BotToken,
		chatID:     opts.ChatID,
		parseMode:  opts.ParseMode,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     opts.Logger,
		enabled:    opts.BotToken != "" && opts.ChatID != "",
	}
}

// Name returns the notifier name
func (t *TelegramNotifier) Name() string {
	return "telegram"
}

// IsEnabled returns whether the notifier is enabled
func (t *TelegramNotifier) IsEnabled() bool {
	return t.enabled
}

// Send sends a notification to Telegram
func (t *TelegramNotifier) Send(ctx context.Context, event Event) error {
	if !t.enabled {
		return nil
	}

	message := t.formatMessage(event)

	payload := telegramPayload{
		ChatID:    t.chatID,
		Text:      message,
		ParseMode: t.parseMode,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", telegramAPIBase, t.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send telegram notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

func (t *TelegramNotifier) formatMessage(event Event) string {
	var emoji string
	switch event.Severity {
	case SeverityCritical:
		emoji = "\U0001F6A8" // Rotating light
	case SeverityError:
		emoji = "\u26A0\uFE0F" // Warning sign
	case SeverityWarning:
		emoji = "\U0001F7E1" // Yellow circle
	default:
		emoji = "\u2139\uFE0F" // Info
	}

	title := t.getTitle(event)
	description := t.getDescription(event)

	if t.parseMode == "HTML" {
		return fmt.Sprintf("%s <b>%s</b>\n\n%s\n\n<b>Validator:</b> %s\n<b>Cluster:</b> %s\n<b>IP:</b> %s\n<b>Time:</b> %s",
			emoji,
			title,
			description,
			event.ValidatorName,
			event.Cluster,
			event.PublicIP,
			event.Timestamp.Format(time.RFC3339),
		)
	}

	// Markdown format
	return fmt.Sprintf("%s *%s*\n\n%s\n\n*Validator:* %s\n*Cluster:* %s\n*IP:* %s\n*Time:* %s",
		emoji,
		title,
		description,
		event.ValidatorName,
		event.Cluster,
		event.PublicIP,
		event.Timestamp.Format(time.RFC3339),
	)
}

func (t *TelegramNotifier) getTitle(event Event) string {
	switch event.Type {
	case EventStartup:
		return "Validator HA Started"
	case EventShutdown:
		return "Validator HA Stopped"
	case EventBecomingActive:
		return "FAILOVER: Becoming Active"
	case EventBecameActive:
		return "Became Active"
	case EventBecomingPassive:
		return "Becoming Passive"
	case EventBecamePassive:
		return "Became Passive"
	case EventHealthUnhealthy:
		return "Health Alert: Unhealthy"
	case EventHealthRecovered:
		return "Health Recovered"
	case EventDelinquent:
		return "CRITICAL: Validator Delinquent"
	case EventGossipLost:
		return "Lost from Gossip"
	case EventGossipRecovered:
		return "Gossip Recovered"
	case EventPeerDiscovered:
		return "Peer Discovered"
	case EventPeerLost:
		return "Peer Lost"
	default:
		return string(event.Type)
	}
}

func (t *TelegramNotifier) getDescription(event Event) string {
	if event.Message != "" {
		return event.Message
	}

	switch event.Type {
	case EventStartup:
		return fmt.Sprintf("Validator %s HA manager has started", event.ValidatorName)
	case EventShutdown:
		return fmt.Sprintf("Validator %s HA manager is shutting down", event.ValidatorName)
	case EventBecomingActive:
		return fmt.Sprintf("Validator %s is transitioning to ACTIVE role", event.ValidatorName)
	case EventBecameActive:
		return fmt.Sprintf("Validator %s is now ACTIVE", event.ValidatorName)
	case EventBecomingPassive:
		return fmt.Sprintf("Validator %s is transitioning to passive role", event.ValidatorName)
	case EventBecamePassive:
		return fmt.Sprintf("Validator %s is now passive", event.ValidatorName)
	case EventHealthUnhealthy:
		return fmt.Sprintf("Validator %s is reporting unhealthy status", event.ValidatorName)
	case EventHealthRecovered:
		return fmt.Sprintf("Validator %s health has recovered", event.ValidatorName)
	case EventDelinquent:
		return fmt.Sprintf("Validator %s is DELINQUENT - not voting!", event.ValidatorName)
	case EventGossipLost:
		return fmt.Sprintf("Validator %s is no longer visible in gossip", event.ValidatorName)
	case EventGossipRecovered:
		return fmt.Sprintf("Validator %s is now visible in gossip", event.ValidatorName)
	case EventPeerDiscovered:
		return fmt.Sprintf("New peer discovered by %s", event.ValidatorName)
	case EventPeerLost:
		return fmt.Sprintf("Peer lost by %s", event.ValidatorName)
	default:
		return fmt.Sprintf("Event on validator %s", event.ValidatorName)
	}
}
