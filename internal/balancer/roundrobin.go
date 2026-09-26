package balancer

import (
	"sync/atomic"

	"orbit/internal/backend"
)

type RoundRobin struct {
	pool  *backend.Pool
	count atomic.Uint32
}

func New(pool *backend.Pool) *RoundRobin {
	return &RoundRobin{pool: pool}
}

func (rr *RoundRobin) Next() string {
	healthy := rr.pool.HealthyBackends()
	if len(healthy) == 0 {
		return ""
	}
	i := rr.count.Add(1) - 1
	return healthy[i%uint32(len(healthy))].Addr
}
