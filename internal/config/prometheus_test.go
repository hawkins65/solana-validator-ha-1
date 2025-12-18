package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrometheus_SetDefaults(t *testing.T) {
	prometheus := &Prometheus{}
	prometheus.SetDefaults()

	assert.Equal(t, 9090, prometheus.Port)
	assert.Equal(t, 9091, prometheus.HealthCheckPort)
}

func TestPrometheus_Validate(t *testing.T) {
	// Test with valid ports
	prometheus := &Prometheus{
		Port:            9090,
		HealthCheckPort: 9091,
		StaticLabels: map[string]string{
			"environment": "test",
		},
	}

	err := prometheus.Validate()
	assert.NoError(t, err)

	// Test with zero port
	prometheus.Port = 0
	err = prometheus.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prometheus.port must be positive and non-zero")

	// Test with negative port
	prometheus.Port = -1
	err = prometheus.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prometheus.port must be positive and non-zero")

	// Test with valid port but zero health check port
	prometheus.Port = 8080
	prometheus.HealthCheckPort = 0
	err = prometheus.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prometheus.health_check_port must be positive and non-zero")

	// Test with valid ports again
	prometheus.HealthCheckPort = 9091
	err = prometheus.Validate()
	assert.NoError(t, err)
}
