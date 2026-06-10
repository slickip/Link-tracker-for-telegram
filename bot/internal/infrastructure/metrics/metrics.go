package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	CommandRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "command_requests_total",
			Help: "Total number of processed bot commands.",
		},
		[]string{"command"},
	)

	CommandDurationMs = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "command_duration_ms_total",
			Help: "Bot command processing duration in milliseconds.",
			Buckets: []float64{
				5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000,
			},
		},
		[]string{"scope", "scope_type"},
	)

	SentNotificationTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sent_notification_total",
			Help: "Total number of sent notifications.",
		},
	)
)

func Register() {
	prometheus.MustRegister(
		CommandRequestsTotal,
		CommandDurationMs,
		SentNotificationTotal,
	)
}
