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

// Discord color codes (decimal)
const (
	colorCritical = 0xFF0000 // Red
	colorError    = 0xFF8C00 // Orange
	colorWarning  = 0xFFFF00 // Yellow
	colorInfo     = 0x00FF00 // Green
)

// DiscordOptions contains options for creating a Discord notifier
type DiscordOptions struct {
	WebhookURL string
	Username   string
	AvatarURL  string
	Logger     *log.Logger
}

// DiscordNotifier sends notifications to Discord via webhooks
type DiscordNotifier struct {
	webhookURL string
	username   string
	avatarURL  string
	httpClient *http.Client
	logger     *log.Logger
	enabled    bool
}

// Discord webhook payload structures
type discordPayload struct {
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Embeds    []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Color       int            `json:"color"`
	Timestamp   string         `json:"timestamp,omitempty"`
	Fields      []discordField `json:"fields,omitempty"`
	Footer      *discordFooter `json:"footer,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordFooter struct {
	Text string `json:"text"`
}

// NewDiscordNotifier creates a new Discord notifier
func NewDiscordNotifier(opts DiscordOptions) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: opts.WebhookURL,
		username:   opts.Username,
		avatarURL:  opts.AvatarURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     opts.Logger,
		enabled:    opts.WebhookURL != "",
	}
}

// Name returns the notifier name
func (d *DiscordNotifier) Name() string {
	return "discord"
}

// IsEnabled returns whether the notifier is enabled
func (d *DiscordNotifier) IsEnabled() bool {
	return d.enabled
}

// Send sends a notification to Discord
func (d *DiscordNotifier) Send(ctx context.Context, event Event) error {
	if !d.enabled {
		return nil
	}

	// Build embed
	embed := discordEmbed{
		Title:       d.getTitle(event),
		Description: d.getDescription(event),
		Color:       d.getColor(event.Severity),
		Timestamp:   event.Timestamp.Format(time.RFC3339),
		Fields:      d.getFields(event),
		Footer: &discordFooter{
			Text: "Solana Validator HA",
		},
	}

	payload := discordPayload{
		Username:  d.username,
		AvatarURL: d.avatarURL,
		Embeds:    []discordEmbed{embed},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send discord notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (d *DiscordNotifier) getTitle(event Event) string {
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

func (d *DiscordNotifier) getDescription(event Event) string {
	if event.Message != "" {
		return event.Message
	}

	switch event.Type {
	case EventStartup:
		return fmt.Sprintf("Validator **%s** HA manager has started", event.ValidatorName)
	case EventShutdown:
		return fmt.Sprintf("Validator **%s** HA manager is shutting down", event.ValidatorName)
	case EventBecomingActive:
		return fmt.Sprintf("Validator **%s** is transitioning to ACTIVE role", event.ValidatorName)
	case EventBecameActive:
		return fmt.Sprintf("Validator **%s** is now ACTIVE", event.ValidatorName)
	case EventBecomingPassive:
		return fmt.Sprintf("Validator **%s** is transitioning to passive role", event.ValidatorName)
	case EventBecamePassive:
		return fmt.Sprintf("Validator **%s** is now passive", event.ValidatorName)
	case EventHealthUnhealthy:
		return fmt.Sprintf("Validator **%s** is reporting unhealthy status", event.ValidatorName)
	case EventHealthRecovered:
		return fmt.Sprintf("Validator **%s** health has recovered", event.ValidatorName)
	case EventDelinquent:
		return fmt.Sprintf("Validator **%s** is DELINQUENT - not voting!", event.ValidatorName)
	case EventGossipLost:
		return fmt.Sprintf("Validator **%s** is no longer visible in gossip", event.ValidatorName)
	case EventGossipRecovered:
		return fmt.Sprintf("Validator **%s** is now visible in gossip", event.ValidatorName)
	case EventPeerDiscovered:
		return fmt.Sprintf("New peer discovered by **%s**", event.ValidatorName)
	case EventPeerLost:
		return fmt.Sprintf("Peer lost by **%s**", event.ValidatorName)
	default:
		return fmt.Sprintf("Event on validator **%s**", event.ValidatorName)
	}
}

func (d *DiscordNotifier) getColor(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return colorCritical
	case SeverityError:
		return colorError
	case SeverityWarning:
		return colorWarning
	default:
		return colorInfo
	}
}

func (d *DiscordNotifier) getFields(event Event) []discordField {
	fields := []discordField{
		{Name: "Validator", Value: event.ValidatorName, Inline: true},
		{Name: "Cluster", Value: event.Cluster, Inline: true},
	}

	if event.PublicIP != "" {
		fields = append(fields, discordField{Name: "IP", Value: event.PublicIP, Inline: true})
	}

	if event.ActivePubkey != "" {
		fields = append(fields, discordField{Name: "Active Pubkey", Value: truncatePubkey(event.ActivePubkey), Inline: true})
	}

	if event.PassivePubkey != "" {
		fields = append(fields, discordField{Name: "Passive Pubkey", Value: truncatePubkey(event.PassivePubkey), Inline: true})
	}

	// Add any additional details
	for k, v := range event.Details {
		fields = append(fields, discordField{Name: k, Value: v, Inline: true})
	}

	return fields
}

// truncatePubkey truncates a pubkey for display
func truncatePubkey(pubkey string) string {
	if len(pubkey) <= 12 {
		return pubkey
	}
	return pubkey[:6] + "..." + pubkey[len(pubkey)-4:]
}
