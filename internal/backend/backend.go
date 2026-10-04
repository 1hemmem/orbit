package backend

import (
	"sync/atomic"
	"time"
)

type Backend struct {
	Name           string
	Addr           string
	HealthPath     string
	HealthInterval time.Duration
	HealthTimeout  time.Duration
	healthy        atomic.Bool
}

func (b *Backend) Healthy() bool     { return b.healthy.Load() }
func (b *Backend) SetHealthy(v bool) { b.healthy.Store(v) }

type Pool struct {
	backends []*Backend
}

func New(addr string) *Backend { return &Backend{Addr: addr} }

func NewPool(backends ...*Backend) *Pool {
	for _, b := range backends {
		b.healthy.Store(true)
	}
	return &Pool{backends: backends}
}

func (p *Pool) All() []*Backend { return p.backends }

func (p *Pool) HealthyBackends() []*Backend {
	out := make([]*Backend, 0, len(p.backends))
	for _, b := range p.backends {
		if b.Healthy() {
			out = append(out, b)
		}
	}
	return out
}
