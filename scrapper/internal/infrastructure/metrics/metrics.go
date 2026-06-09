package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	LinksOnTrackTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "links_on_track_total",
			Help: "Current number of tracked links grouped by source.",
		},
		[]string{"tracked_source"},
	)

	RequestDurationMs = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "request_duration_ms_total",
			Help: "Scrapper operation duration in milliseconds.",
			Buckets: []float64{
				5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000,
			},
		},
		[]string{"scope", "scope_type"},
	)

	APIRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "Total number of incoming Scrapper API requests.",
		},
		[]string{"source"},
	)
)

func Register() {
	prometheus.MustRegister(
		LinksOnTrackTotal,
		RequestDurationMs,
		APIRequestsTotal,
	)
}
