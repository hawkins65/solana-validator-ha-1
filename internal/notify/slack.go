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

// SlackOptions contains options for creating a Slack notifier
type SlackOptions struct {
	WebhookURL string
	Channel    string
	Username   string
	IconEmoji  string
	Logger     *log.Logger
}

// SlackNotifier sends notifications to Slack via webhooks
type SlackNotifier struct {
	webhookURL string
	channel    string
	username   string
	iconEmoji  string
	httpClient *http.Client
	logger     *log.Logger
	enabled    bool
}

// Slack webhook payload structures
type slackPayload struct {
	Channel     string        `json:"channel,omitempty"`
	Username    string        `json:"username,omitempty"`
	IconEmoji   string        `json:"icon_emoji,omitempty"`
	Attachments []slackAttachment `json:"attachments"`
}

type slackAttachment struct {
	Color      string       `json:"color"`
	Title      string       `json:"title"`
	Text       string       `json:"text"`
	Fields     []slackField `json:"fields,omitempty"`
	Footer     string       `json:"footer"`
	Timestamp  int64        `json:"ts"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(opts SlackOptions) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: opts.WebhookURL,
		channel:    opts.Channel,
		username:   opts.Username,
		iconEmoji:  opts.IconEmoji,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     opts.Logger,
		enabled:    opts.WebhookURL != "",
	}
}

// Name returns the notifier name
func (s *SlackNotifier) Name() string {
	return "slack"
}

// IsEnabled returns whether the notifier is enabled
func (s *SlackNotifier) IsEnabled() bool {
	return s.enabled
}

// Send sends a notification to Slack
func (s *SlackNotifier) Send(ctx context.Context, event Event) error {
	if !s.enabled {
		return nil
	}

	attachment := slackAttachment{
		Color:     s.getColor(event.Severity),
		Title:     s.getTitle(event),
		Text:      s.getDescription(event),
		Fields:    s.getFields(event),
		Footer:    "Solana Validator HA",
		Timestamp: event.Timestamp.Unix(),
	}

	payload := slackPayload{
		Channel:     s.channel,
		Username:    s.username,
		IconEmoji:   s.iconEmoji,
		Attachments: []slackAttachment{attachment},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *SlackNotifier) getTitle(event Event) string {
	var emoji string
	switch event.Severity {
	case SeverityCritical:
		emoji = ":rotating_light:"
	case SeverityError:
		emoji = ":warning:"
	case SeverityWarning:
		emoji = ":large_yellow_circle:"
	default:
		emoji = ":information_source:"
	}

	var title string
	switch event.Type {
	case EventStartup:
		title = "Validator HA Started"
	case EventShutdown:
		title = "Validator HA Stopped"
	case EventBecomingActive:
		title = "FAILOVER: Becoming Active"
	case EventBecameActive:
		title = "Became Active"
	case EventBecomingPassive:
		title = "Becoming Passive"
	case EventBecamePassive:
		title = "Became Passive"
	case EventHealthUnhealthy:
		title = "Health Alert: Unhealthy"
	case EventHealthRecovered:
		title = "Health Recovered"
	case EventDelinquent:
		title = "CRITICAL: Validator Delinquent"
	case EventGossipLost:
		title = "Lost from Gossip"
	case EventGossipRecovered:
		title = "Gossip Recovered"
	case EventPeerDiscovered:
		title = "Peer Discovered"
	case EventPeerLost:
		title = "Peer Lost"
	default:
		title = string(event.Type)
	}

	return fmt.Sprintf("%s %s", emoji, title)
}

func (s *SlackNotifier) getDescription(event Event) string {
	if event.Message != "" {
		return event.Message
	}

	switch event.Type {
	case EventStartup:
		return fmt.Sprintf("Validator *%s* HA manager has started", event.ValidatorName)
	case EventShutdown:
		return fmt.Sprintf("Validator *%s* HA manager is shutting down", event.ValidatorName)
	case EventBecomingActive:
		return fmt.Sprintf("Validator *%s* is transitioning to ACTIVE role", event.ValidatorName)
	case EventBecameActive:
		return fmt.Sprintf("Validator *%s* is now ACTIVE", event.ValidatorName)
	case EventBecomingPassive:
		return fmt.Sprintf("Validator *%s* is transitioning to passive role", event.ValidatorName)
	case EventBecamePassive:
		return fmt.Sprintf("Validator *%s* is now passive", event.ValidatorName)
	case EventHealthUnhealthy:
		return fmt.Sprintf("Validator *%s* is reporting unhealthy status", event.ValidatorName)
	case EventHealthRecovered:
		return fmt.Sprintf("Validator *%s* health has recovered", event.ValidatorName)
	case EventDelinquent:
		return fmt.Sprintf("Validator *%s* is DELINQUENT - not voting!", event.ValidatorName)
	case EventGossipLost:
		return fmt.Sprintf("Validator *%s* is no longer visible in gossip", event.ValidatorName)
	case EventGossipRecovered:
		return fmt.Sprintf("Validator *%s* is now visible in gossip", event.ValidatorName)
	case EventPeerDiscovered:
		return fmt.Sprintf("New peer discovered by *%s*", event.ValidatorName)
	case EventPeerLost:
		return fmt.Sprintf("Peer lost by *%s*", event.ValidatorName)
	default:
		return fmt.Sprintf("Event on validator *%s*", event.ValidatorName)
	}
}

func (s *SlackNotifier) getColor(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "#FF0000" // Red
	case SeverityError:
		return "#FF8C00" // Orange
	case SeverityWarning:
		return "#FFFF00" // Yellow
	default:
		return "#00FF00" // Green
	}
}

func (s *SlackNotifier) getFields(event Event) []slackField {
	fields := []slackField{
		{Title: "Validator", Value: event.ValidatorName, Short: true},
		{Title: "Cluster", Value: event.Cluster, Short: true},
	}

	if event.PublicIP != "" {
		fields = append(fields, slackField{Title: "IP", Value: event.PublicIP, Short: true})
	}

	if event.ActivePubkey != "" {
		fields = append(fields, slackField{Title: "Active Pubkey", Value: truncatePubkey(event.ActivePubkey), Short: true})
	}

	if event.PassivePubkey != "" {
		fields = append(fields, slackField{Title: "Passive Pubkey", Value: truncatePubkey(event.PassivePubkey), Short: true})
	}

	// Add any additional details
	for k, v := range event.Details {
		fields = append(fields, slackField{Title: k, Value: v, Short: true})
	}

	return fields
}
