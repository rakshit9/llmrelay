package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llmrelay_requests_total",
		Help: "Total number of proxied requests",
	}, []string{"model", "provider", "status"})

	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "llmrelay_request_duration_seconds",
		Help:    "End-to-end request latency",
		Buckets: []float64{.05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"model", "provider"})

	TokensTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llmrelay_tokens_total",
		Help: "Total tokens used",
	}, []string{"model", "provider", "type"}) // type: prompt | completion

	CacheHitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llmrelay_cache_hits_total",
		Help: "Total cache hits",
	}, []string{"cache_type"}) // cache_type: exact | semantic
)
