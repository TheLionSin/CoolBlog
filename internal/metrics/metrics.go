package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Publisher metrics
	OutboxFetched *prometheus.CounterVec
	OutboxSent    prometheus.Counter
	OutboxRetry   prometheus.Counter
	OutboxDead    prometheus.Counter

	// Consumer metrics
	ConsumerProcessed prometheus.Counter
	ConsumerErrors    prometheus.Counter
)

// Init инициализирует метрики и регистрирует их.
// Эту функцию надо вызвать ОДИН РАЗ в начале main.go
func Init() {
	// 1. Создаем счетчики
	OutboxFetched = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blog_outbox_fetched_total",
		Help: "Total number of events fetched from DB",
	}, []string{"status"},
	)

	OutboxSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "blog_outbox_sent_total",
		Help: "Total number of events sent to Kafka",
	})

	OutboxRetry = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "blog_outbox_retry_total",
		Help: "Total number of retries in publisher",
	})

	OutboxDead = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "blog_outbox_dead_total",
		Help: "Total number of dead events (DLQ)",
	})

	ConsumerProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "blog_consumer_processed_total",
		Help: "Total number of events processed by consumer",
	})

	ConsumerErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "blog_consumer_errors_total",
		Help: "Total number of errors in consumer",
	})

	// 2. Регистрируем их в Прометеусе
	// Если забыть этот шаг, метрики будут считаться, но не будут видны.
	prometheus.MustRegister(OutboxFetched)
	prometheus.MustRegister(OutboxSent)
	prometheus.MustRegister(OutboxRetry)
	prometheus.MustRegister(OutboxDead)
	prometheus.MustRegister(ConsumerProcessed)
	prometheus.MustRegister(ConsumerErrors)
}
