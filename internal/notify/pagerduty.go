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

const pagerDutyEventsAPI = "https://events.pagerduty.com/v2/enqueue"

// PagerDutyOptions contains options for creating a PagerDuty notifier
type PagerDutyOptions struct {
	RoutingKey string
	Logger     *log.Logger
}

// PagerDutyNotifier sends notifications to PagerDuty via Events API v2
type PagerDutyNotifier struct {
	routingKey string
	httpClient *http.Client
	logger     *log.Logger
	enabled    bool
}

// PagerDuty Events API v2 payload structures
type pagerDutyPayload struct {
	RoutingKey  string         `json:"routing_key"`
	EventAction string         `json:"event_action"`
	DedupKey    string         `json:"dedup_key,omitempty"`
	Payload     pagerDutyEvent `json:"payload"`
}

type pagerDutyEvent struct {
	Summary       string            `json:"summary"`
	Severity      string            `json:"severity"`
	Source        string            `json:"source"`
	Timestamp     string            `json:"timestamp,omitempty"`
	Component     string            `json:"component,omitempty"`
	Group         string            `json:"group,omitempty"`
	Class         string            `json:"class,omitempty"`
	CustomDetails map[string]string `json:"custom_details,omitempty"`
}

// NewPagerDutyNotifier creates a new PagerDuty notifier
func NewPagerDutyNotifier(opts PagerDutyOptions) *PagerDutyNotifier {
	return &PagerDutyNotifier{
		routingKey: opts.RoutingKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     opts.Logger,
		enabled:    opts.RoutingKey != "",
	}
}

// Name returns the notifier name
func (p *PagerDutyNotifier) Name() string {
	return "pagerduty"
}

// IsEnabled returns whether the notifier is enabled
func (p *PagerDutyNotifier) IsEnabled() bool {
	return p.enabled
}

// Send sends a notification to PagerDuty
func (p *PagerDutyNotifier) Send(ctx context.Context, event Event) error {
	if !p.enabled {
		return nil
	}

	// Determine event action based on event type
	eventAction := "trigger"
	if event.Type == EventHealthRecovered || event.Type == EventGossipRecovered || event.Type == EventBecamePassive {
		eventAction = "resolve"
	}

	// Build custom details
	customDetails := map[string]string{
		"validator_name": event.ValidatorName,
		"cluster":        event.Cluster,
		"public_ip":      event.PublicIP,
		"event_type":     string(event.Type),
	}

	if event.ActivePubkey != "" {
		customDetails["active_pubkey"] = event.ActivePubkey
	}
	if event.PassivePubkey != "" {
		customDetails["passive_pubkey"] = event.PassivePubkey
	}

	// Add any additional details
	for k, v := range event.Details {
		customDetails[k] = v
	}

	payload := pagerDutyPayload{
		RoutingKey:  p.routingKey,
		EventAction: eventAction,
		DedupKey:    p.getDedupKey(event),
		Payload: pagerDutyEvent{
			Summary:       p.getSummary(event),
			Severity:      p.getSeverity(event.Severity),
			Source:        event.ValidatorName,
			Timestamp:     event.Timestamp.Format(time.RFC3339),
			Component:     "solana-validator-ha",
			Group:         event.Cluster,
			Class:         string(event.Type),
			CustomDetails: customDetails,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal pagerduty payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pagerDutyEventsAPI, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create pagerduty request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send pagerduty notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pagerduty API returned status %d", resp.StatusCode)
	}

	return nil
}

func (p *PagerDutyNotifier) getSummary(event Event) string {
	if event.Message != "" {
		return event.Message
	}

	switch event.Type {
	case EventStartup:
		return fmt.Sprintf("[%s] Validator HA manager started", event.ValidatorName)
	case EventShutdown:
		return fmt.Sprintf("[%s] Validator HA manager stopped", event.ValidatorName)
	case EventBecomingActive:
		return fmt.Sprintf("[%s] FAILOVER: Validator becoming active", event.ValidatorName)
	case EventBecameActive:
		return fmt.Sprintf("[%s] Validator is now active", event.ValidatorName)
	case EventBecomingPassive:
		return fmt.Sprintf("[%s] Validator becoming passive", event.ValidatorName)
	case EventBecamePassive:
		return fmt.Sprintf("[%s] Validator is now passive", event.ValidatorName)
	case EventHealthUnhealthy:
		return fmt.Sprintf("[%s] Validator health check failed", event.ValidatorName)
	case EventHealthRecovered:
		return fmt.Sprintf("[%s] Validator health recovered", event.ValidatorName)
	case EventDelinquent:
		return fmt.Sprintf("[%s] CRITICAL: Validator is delinquent (not voting)", event.ValidatorName)
	case EventGossipLost:
		return fmt.Sprintf("[%s] Validator lost from gossip network", event.ValidatorName)
	case EventGossipRecovered:
		return fmt.Sprintf("[%s] Validator visible in gossip network", event.ValidatorName)
	case EventPeerDiscovered:
		peerName := event.Details["peer_name"]
		return fmt.Sprintf("[%s] Peer discovered: %s", event.ValidatorName, peerName)
	case EventPeerLost:
		peerName := event.Details["peer_name"]
		return fmt.Sprintf("[%s] Peer lost: %s", event.ValidatorName, peerName)
	default:
		return fmt.Sprintf("[%s] Event: %s", event.ValidatorName, event.Type)
	}
}

func (p *PagerDutyNotifier) getSeverity(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "critical"
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	default:
		return "info"
	}
}

// getDedupKey returns a deduplication key for the event
// Events with the same dedup key will be grouped together
func (p *PagerDutyNotifier) getDedupKey(event Event) string {
	// Group related events together
	switch event.Type {
	case EventHealthUnhealthy, EventHealthRecovered:
		return fmt.Sprintf("%s-health", event.ValidatorName)
	case EventGossipLost, EventGossipRecovered:
		return fmt.Sprintf("%s-gossip", event.ValidatorName)
	case EventBecomingActive, EventBecameActive:
		return fmt.Sprintf("%s-active-%d", event.ValidatorName, event.Timestamp.Unix())
	case EventBecomingPassive, EventBecamePassive:
		return fmt.Sprintf("%s-passive-%d", event.ValidatorName, event.Timestamp.Unix())
	case EventPeerLost, EventPeerDiscovered:
		peerName := event.Details["peer_name"]
		return fmt.Sprintf("%s-peer-%s", event.ValidatorName, peerName)
	default:
		return fmt.Sprintf("%s-%s-%d", event.ValidatorName, event.Type, event.Timestamp.Unix())
	}
}
