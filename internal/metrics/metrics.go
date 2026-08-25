package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

const namespace = "flexprice"

// HTTP metrics
var (
	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests.",
	}, []string{"method", "path", "status_code"})
	
	HTTPRequestLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "http",
		Name:      "request_latency_seconds",
		Help:      "HTTP request latency in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})
	
	HTTPRequestFailuresTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "http",
		Name:      "request_failures_total",
		Help:      "Total number of HTTP request failures (status >= 400).",
	}, []string{"method", "path", "error_code"})
)

// Kafka metrics
var (
	KafkaMessagesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "kafka",
		Name:      "messages_total",
		Help:      "Total number of Kafka messages processed.",
	}, []string{"consumer", "topic", "status"})
	
	KafkaMessageLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "kafka",
		Name:      "message_latency_seconds",
		Help:      "Kafka message processing latency in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"consumer", "topic"})
	
	KafkaMessageFailuresTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "kafka",
		Name:      "message_failures_total",
		Help:      "Total number of Kafka message processing failures.",
	}, []string{"consumer", "topic", "error_code"})
	
	BenefitLedgerDuplicatesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "benefit_ledger",
		Name:      "duplicates_total",
		Help:      "Total number of duplicate Ledger duplicates.",
	})
)

func Init() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestLatency,
		HTTPRequestFailuresTotal,
		KafkaMessagesTotal,
		KafkaMessageLatency,
		KafkaMessageFailuresTotal,
		BenefitLedgerDuplicatesTotal,
	)
}

func Module() fx.Option {
	return fx.Invoke(func() {
		Init()
	})
}
