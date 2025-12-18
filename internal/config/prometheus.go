package config

import "fmt"

// Prometheus represents Prometheus metrics configuration
type Prometheus struct {
	Port            int               `koanf:"port"`
	HealthCheckPort int               `koanf:"health_check_port"`
	StaticLabels    map[string]string `koanf:"static_labels"`
}

// Validate validates the Prometheus configuration
func (p *Prometheus) Validate() error {
	// prometheus.port must be positive and non-zero
	if p.Port <= 0 {
		return fmt.Errorf("prometheus.port must be positive and non-zero")
	}

	// prometheus.health_check_port must be positive and non-zero
	if p.HealthCheckPort <= 0 {
		return fmt.Errorf("prometheus.health_check_port must be positive and non-zero")
	}

	return nil
}

// SetDefaults sets default values for the Prometheus configuration
func (p *Prometheus) SetDefaults() {
	// if prometheus.port is 0, set it to the default port
	if p.Port == 0 {
		p.Port = 9090
	}

	// if prometheus.health_check_port is 0, set it to the default port
	if p.HealthCheckPort == 0 {
		p.HealthCheckPort = 9091
	}
}
