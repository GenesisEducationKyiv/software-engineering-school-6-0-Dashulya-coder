package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var emailsSentTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "emails_sent_total",
		Help: "Total emails sent by type and outcome.",
	},
	[]string{"type", "status"},
)

func RecordEmail(emailType string, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	emailsSentTotal.WithLabelValues(emailType, status).Inc()
}
