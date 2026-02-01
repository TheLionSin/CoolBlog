package metrics

import "sync/atomic"

var (
	OutboxFetched atomic.Uint64
	OutboxSent    atomic.Uint64
	OutboxDead    atomic.Uint64
	OutboxRetry   atomic.Uint64

	ConsumerProcessed atomic.Uint64
	ConsumerErrors    atomic.Uint64
)

func Inc(metric *atomic.Uint64) {
	metric.Add(1)
}
