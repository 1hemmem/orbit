package backend

import "sync/atomic"

type Backend struct {
	Addr    string
	healthy atomic.Bool
}

func (b *Backend) Healthy() bool     { return b.healthy.Load() }
func (b *Backend) SetHealthy(v bool) { b.healthy.Store(v) }

type Pool struct {
	backends []*Backend
}

func NewPool(addrs ...string) *Pool {
	p := &Pool{}
	for _, addr := range addrs {
		b := &Backend{Addr: addr}
		b.healthy.Store(true)
		p.backends = append(p.backends, b)
	}
	return p
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
