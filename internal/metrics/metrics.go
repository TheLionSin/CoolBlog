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
