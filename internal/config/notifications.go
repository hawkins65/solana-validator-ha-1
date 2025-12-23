package config

import (
	"fmt"
	"os"
)

// NotificationConfig represents the notifications configuration
type NotificationConfig struct {
	Enabled   bool                   `koanf:"enabled"`
	Discord   DiscordConfig          `koanf:"discord"`
	Telegram  TelegramConfig         `koanf:"telegram"`
	Slack     SlackConfig            `koanf:"slack"`
	PagerDuty PagerDutyConfig        `koanf:"pagerduty"`
	Events    NotificationEvents     `koanf:"events"`
}

// NotificationEvents controls which events trigger notifications
type NotificationEvents struct {
	Startup         bool `koanf:"startup"`
	Shutdown        bool `koanf:"shutdown"`
	BecomingActive  bool `koanf:"becoming_active"`
	BecameActive    bool `koanf:"became_active"`
	BecomingPassive bool `koanf:"becoming_passive"`
	BecamePassive   bool `koanf:"became_passive"`
	HealthUnhealthy bool `koanf:"health_unhealthy"`
	HealthRecovered bool `koanf:"health_recovered"`
	Delinquent      bool `koanf:"delinquent"`
	GossipLost      bool `koanf:"gossip_lost"`
	GossipRecovered bool `koanf:"gossip_recovered"`
	PeerDiscovered  bool `koanf:"peer_discovered"`
	PeerLost        bool `koanf:"peer_lost"`
}

// DiscordConfig for Discord webhooks
type DiscordConfig struct {
	Enabled       bool   `koanf:"enabled"`
	WebhookURL    string `koanf:"webhook_url"`
	WebhookURLEnv string `koanf:"webhook_url_env"`
	Username      string `koanf:"username"`
	AvatarURL     string `koanf:"avatar_url"`
}

// TelegramConfig for Telegram Bot API
type TelegramConfig struct {
	Enabled     bool   `koanf:"enabled"`
	BotToken    string `koanf:"bot_token"`
	BotTokenEnv string `koanf:"bot_token_env"`
	ChatID      string `koanf:"chat_id"`
	ParseMode   string `koanf:"parse_mode"`
}

// SlackConfig for Slack webhooks
type SlackConfig struct {
	Enabled       bool   `koanf:"enabled"`
	WebhookURL    string `koanf:"webhook_url"`
	WebhookURLEnv string `koanf:"webhook_url_env"`
	Channel       string `koanf:"channel"`
	Username      string `koanf:"username"`
	IconEmoji     string `koanf:"icon_emoji"`
}

// PagerDutyConfig for PagerDuty Events API v2
type PagerDutyConfig struct {
	Enabled       bool   `koanf:"enabled"`
	RoutingKey    string `koanf:"routing_key"`
	RoutingKeyEnv string `koanf:"routing_key_env"`
}

// SetDefaults sets default values for notification configuration
func (n *NotificationConfig) SetDefaults() {
	// Events defaults - all enabled by default when notifications are enabled
	n.Events.Startup = true
	n.Events.Shutdown = true
	n.Events.BecomingActive = true
	n.Events.BecameActive = true
	n.Events.BecomingPassive = true
	n.Events.BecamePassive = true
	n.Events.HealthUnhealthy = true
	n.Events.HealthRecovered = true
	n.Events.Delinquent = true
	n.Events.GossipLost = true
	n.Events.GossipRecovered = true
	n.Events.PeerDiscovered = true
	n.Events.PeerLost = true

	// Telegram defaults
	if n.Telegram.ParseMode == "" {
		n.Telegram.ParseMode = "HTML"
	}

	// Discord defaults
	if n.Discord.Username == "" {
		n.Discord.Username = "Solana HA Bot"
	}

	// Slack defaults
	if n.Slack.Username == "" {
		n.Slack.Username = "Solana HA Bot"
	}
	if n.Slack.IconEmoji == "" {
		n.Slack.IconEmoji = ":robot_face:"
	}
}

// Validate validates the notification configuration
func (n *NotificationConfig) Validate() error {
	if !n.Enabled {
		return nil
	}

	// Validate Discord config
	if n.Discord.Enabled {
		if n.Discord.WebhookURL == "" && n.Discord.WebhookURLEnv == "" {
			return fmt.Errorf("notifications.discord: webhook_url or webhook_url_env is required when enabled")
		}
	}

	// Validate Telegram config
	if n.Telegram.Enabled {
		if n.Telegram.BotToken == "" && n.Telegram.BotTokenEnv == "" {
			return fmt.Errorf("notifications.telegram: bot_token or bot_token_env is required when enabled")
		}
		if n.Telegram.ChatID == "" {
			return fmt.Errorf("notifications.telegram: chat_id is required when enabled")
		}
		if n.Telegram.ParseMode != "HTML" && n.Telegram.ParseMode != "Markdown" && n.Telegram.ParseMode != "MarkdownV2" {
			return fmt.Errorf("notifications.telegram: parse_mode must be HTML, Markdown, or MarkdownV2")
		}
	}

	// Validate Slack config
	if n.Slack.Enabled {
		if n.Slack.WebhookURL == "" && n.Slack.WebhookURLEnv == "" {
			return fmt.Errorf("notifications.slack: webhook_url or webhook_url_env is required when enabled")
		}
	}

	// Validate PagerDuty config
	if n.PagerDuty.Enabled {
		if n.PagerDuty.RoutingKey == "" && n.PagerDuty.RoutingKeyEnv == "" {
			return fmt.Errorf("notifications.pagerduty: routing_key or routing_key_env is required when enabled")
		}
	}

	return nil
}

// ResolveSecrets resolves environment variable references for secrets
func (n *NotificationConfig) ResolveSecrets() error {
	if !n.Enabled {
		return nil
	}

	// Resolve Discord webhook URL
	if n.Discord.Enabled && n.Discord.WebhookURL == "" && n.Discord.WebhookURLEnv != "" {
		value := os.Getenv(n.Discord.WebhookURLEnv)
		if value == "" {
			return fmt.Errorf("notifications.discord: environment variable %s is not set", n.Discord.WebhookURLEnv)
		}
		n.Discord.WebhookURL = value
	}

	// Resolve Telegram bot token
	if n.Telegram.Enabled && n.Telegram.BotToken == "" && n.Telegram.BotTokenEnv != "" {
		value := os.Getenv(n.Telegram.BotTokenEnv)
		if value == "" {
			return fmt.Errorf("notifications.telegram: environment variable %s is not set", n.Telegram.BotTokenEnv)
		}
		n.Telegram.BotToken = value
	}

	// Resolve Slack webhook URL
	if n.Slack.Enabled && n.Slack.WebhookURL == "" && n.Slack.WebhookURLEnv != "" {
		value := os.Getenv(n.Slack.WebhookURLEnv)
		if value == "" {
			return fmt.Errorf("notifications.slack: environment variable %s is not set", n.Slack.WebhookURLEnv)
		}
		n.Slack.WebhookURL = value
	}

	// Resolve PagerDuty routing key
	if n.PagerDuty.Enabled && n.PagerDuty.RoutingKey == "" && n.PagerDuty.RoutingKeyEnv != "" {
		value := os.Getenv(n.PagerDuty.RoutingKeyEnv)
		if value == "" {
			return fmt.Errorf("notifications.pagerduty: environment variable %s is not set", n.PagerDuty.RoutingKeyEnv)
		}
		n.PagerDuty.RoutingKey = value
	}

	return nil
}

// HasAnyEnabled returns true if any notification service is enabled
func (n *NotificationConfig) HasAnyEnabled() bool {
	return n.Enabled && (n.Discord.Enabled || n.Telegram.Enabled || n.Slack.Enabled || n.PagerDuty.Enabled)
}
