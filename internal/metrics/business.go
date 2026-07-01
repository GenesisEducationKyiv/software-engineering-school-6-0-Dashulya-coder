package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ScanCyclesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "scan_cycles_total",
		Help: "Total number of release-poll cycles executed.",
	})

	ScanDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "scan_duration_seconds",
		Help:    "Duration of a full release-poll cycle in seconds.",
		Buckets: prometheus.DefBuckets,
	})

	ReleasesDetectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "releases_detected_total",
		Help: "Total number of new releases detected across all repos.",
	})

	GitHubRateLimitHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "github_rate_limit_hits_total",
		Help: "Total number of times the GitHub API returned a rate-limit response.",
	})
)
