package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	FlatsConsumed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "analyzer_flats_consumed_total",
		Help: "Total number of flat messages consumed from Kafka.",
	})

	SubscriptionsMatched = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "analyzer_subscriptions_matched_total",
		Help: "Total number of (flat, subscription) pairs that passed filter.",
	})

	MessagesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "analyzer_messages_sent_total",
		Help: "Total number of Telegram notifications successfully dispatched.",
	})

	MessagesFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "analyzer_messages_failed_total",
		Help: "Total number of Telegram notification send failures.",
	})

	ProcessDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "analyzer_flat_process_duration_seconds",
		Help:    "Time to fully process one flat message (all subscriptions).",
		Buckets: prometheus.DefBuckets,
	})
)

func init() {
	prometheus.MustRegister(
		FlatsConsumed,
		SubscriptionsMatched,
		MessagesSent,
		MessagesFailed,
		ProcessDuration,
	)
}