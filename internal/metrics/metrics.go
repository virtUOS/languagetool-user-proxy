package metrics

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all the Prometheus metrics for the LanguageTool proxy
type Metrics struct {
	queries            Queries
	UsersTotal         prometheus.Gauge
	ValidSessionsTotal prometheus.Gauge
	RequestsTotal      *prometheus.CounterVec
}

// Queries provides database query methods for metrics
type Queries interface {
	CountUsers(ctx context.Context) (int64, error)
	CountValidSessions(ctx context.Context) (int64, error)
}

// NewMetrics creates a new Metrics instance with all metric types
func NewMetrics(queries Queries) *Metrics {
	m := &Metrics{
		queries: queries,
		UsersTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "languagetool_users_total",
			Help: "Total number of users in the system",
		}),
		ValidSessionsTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "languagetool_valid_sessions_total",
			Help: "Total number of valid, non-expired sessions",
		}),
		RequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "languagetool_requests_total",
			Help: "Total number of proxied requests",
		}, []string{"path", "status"}),
	}

	prometheus.MustRegister(m.UsersTotal)
	prometheus.MustRegister(m.ValidSessionsTotal)
	prometheus.MustRegister(m.RequestsTotal)

	return m
}

// UpdateGauges fetches current values from the database and updates the gauge metrics
func (m *Metrics) UpdateGauges(ctx context.Context) error {
	usersCount, err := m.queries.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}
	m.UsersTotal.Set(float64(usersCount))

	sessionsCount, err := m.queries.CountValidSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to count valid sessions: %w", err)
	}
	m.ValidSessionsTotal.Set(float64(sessionsCount))

	return nil
}

// IncrementRequest increments the request counter for a given path and status
func (m *Metrics) IncrementRequest(path, status string) {
	m.RequestsTotal.WithLabelValues(path, status).Inc()
}
