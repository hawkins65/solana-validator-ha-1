package notify

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/sol-strategies/solana-validator-ha/internal/config"
)

// EventType represents the type of notification event
type EventType string

const (
	EventStartup         EventType = "startup"
	EventShutdown        EventType = "shutdown"
	EventBecomingActive  EventType = "becoming_active"
	EventBecameActive    EventType = "became_active"
	EventBecomingPassive EventType = "becoming_passive"
	EventBecamePassive   EventType = "became_passive"
	EventHealthUnhealthy EventType = "health_unhealthy"
	EventHealthRecovered EventType = "health_recovered"
	EventDelinquent      EventType = "delinquent"
	EventGossipLost      EventType = "gossip_lost"
	EventGossipRecovered EventType = "gossip_recovered"
	EventPeerDiscovered  EventType = "peer_discovered"
	EventPeerLost        EventType = "peer_lost"
)

// Severity levels for notifications
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityError    Severity = "error"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Event represents a notification event
type Event struct {
	Type          EventType
	Severity      Severity
	Timestamp     time.Time
	ValidatorName string
	PublicIP      string
	Cluster       string
	ActivePubkey  string
	PassivePubkey string
	Message       string
	Details       map[string]string
}

// Notifier interface for all notification services
type Notifier interface {
	// Name returns the service name (discord, slack, etc.)
	Name() string
	// Send sends a notification
	Send(ctx context.Context, event Event) error
	// IsEnabled returns whether this notifier is enabled
	IsEnabled() bool
}

// Manager coordinates all notification services
type Manager struct {
	notifiers   []Notifier
	logger      *log.Logger
	enabled     bool
	eventFilter config.NotificationEvents
}

// ManagerOptions contains options for creating a new Manager
type ManagerOptions struct {
	Config        *config.NotificationConfig
	ValidatorName string
	PublicIP      string
	Cluster       string
}

// NewManager creates a notification manager from config
func NewManager(opts ManagerOptions) *Manager {
	logger := log.WithPrefix(fmt.Sprintf("[%s notify]", opts.ValidatorName))

	if !opts.Config.Enabled {
		logger.Debug("notifications disabled")
		return &Manager{
			enabled: false,
			logger:  logger,
		}
	}

	notifiers := make([]Notifier, 0)

	// Create Discord notifier if enabled
	if opts.Config.Discord.Enabled {
		notifiers = append(notifiers, NewDiscordNotifier(DiscordOptions{
			WebhookURL: opts.Config.Discord.WebhookURL,
			Username:   opts.Config.Discord.Username,
			AvatarURL:  opts.Config.Discord.AvatarURL,
			Logger:     logger,
		}))
		logger.Debug("discord notifications enabled")
	}

	// Create Telegram notifier if enabled
	if opts.Config.Telegram.Enabled {
		notifiers = append(notifiers, NewTelegramNotifier(TelegramOptions{
			BotToken:  opts.Config.Telegram.BotToken,
			ChatID:    opts.Config.Telegram.ChatID,
			ParseMode: opts.Config.Telegram.ParseMode,
			Logger:    logger,
		}))
		logger.Debug("telegram notifications enabled")
	}

	// Create Slack notifier if enabled
	if opts.Config.Slack.Enabled {
		notifiers = append(notifiers, NewSlackNotifier(SlackOptions{
			WebhookURL: opts.Config.Slack.WebhookURL,
			Channel:    opts.Config.Slack.Channel,
			Username:   opts.Config.Slack.Username,
			IconEmoji:  opts.Config.Slack.IconEmoji,
			Logger:     logger,
		}))
		logger.Debug("slack notifications enabled")
	}

	// Create PagerDuty notifier if enabled
	if opts.Config.PagerDuty.Enabled {
		notifiers = append(notifiers, NewPagerDutyNotifier(PagerDutyOptions{
			RoutingKey: opts.Config.PagerDuty.RoutingKey,
			Logger:     logger,
		}))
		logger.Debug("pagerduty notifications enabled")
	}

	logger.Info("notification manager initialized", "services", len(notifiers))

	return &Manager{
		notifiers:   notifiers,
		logger:      logger,
		enabled:     true,
		eventFilter: opts.Config.Events,
	}
}

// IsEnabled returns whether the notification manager is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled && len(m.notifiers) > 0
}

// isEventEnabled checks if a specific event type is enabled
func (m *Manager) isEventEnabled(eventType EventType) bool {
	switch eventType {
	case EventStartup:
		return m.eventFilter.Startup
	case EventShutdown:
		return m.eventFilter.Shutdown
	case EventBecomingActive:
		return m.eventFilter.BecomingActive
	case EventBecameActive:
		return m.eventFilter.BecameActive
	case EventBecomingPassive:
		return m.eventFilter.BecomingPassive
	case EventBecamePassive:
		return m.eventFilter.BecamePassive
	case EventHealthUnhealthy:
		return m.eventFilter.HealthUnhealthy
	case EventHealthRecovered:
		return m.eventFilter.HealthRecovered
	case EventDelinquent:
		return m.eventFilter.Delinquent
	case EventGossipLost:
		return m.eventFilter.GossipLost
	case EventGossipRecovered:
		return m.eventFilter.GossipRecovered
	case EventPeerDiscovered:
		return m.eventFilter.PeerDiscovered
	case EventPeerLost:
		return m.eventFilter.PeerLost
	default:
		return true
	}
}

// Notify sends an event to all enabled notifiers synchronously
func (m *Manager) Notify(event Event) {
	if !m.enabled {
		return
	}

	if !m.isEventEnabled(event.Type) {
		m.logger.Debug("event type disabled, skipping notification", "event", event.Type)
		return
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, notifier := range m.notifiers {
		if !notifier.IsEnabled() {
			continue
		}

		if err := notifier.Send(ctx, event); err != nil {
			m.logger.Error("notification failed",
				"service", notifier.Name(),
				"event", event.Type,
				"error", err,
			)
		} else {
			m.logger.Debug("notification sent",
				"service", notifier.Name(),
				"event", event.Type,
			)
		}
	}
}

// NotifyAsync sends notification in background goroutine (non-blocking)
func (m *Manager) NotifyAsync(event Event) {
	if !m.enabled {
		return
	}

	if !m.isEventEnabled(event.Type) {
		m.logger.Debug("event type disabled, skipping notification", "event", event.Type)
		return
	}

	go m.Notify(event)
}

// Helper function to get default severity for an event type
func GetDefaultSeverity(eventType EventType) Severity {
	switch eventType {
	case EventBecomingActive, EventDelinquent:
		return SeverityCritical
	case EventHealthUnhealthy, EventGossipLost, EventPeerLost:
		return SeverityError
	case EventBecomingPassive, EventShutdown:
		return SeverityWarning
	default:
		return SeverityInfo
	}
}
