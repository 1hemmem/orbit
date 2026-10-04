package app

import (
	"net/http"

	"orbit/internal/backend"
	"orbit/internal/balancer"
	"orbit/internal/config"
	"orbit/internal/healthcheck"
	"orbit/internal/proxy"
)

func NewHandler(cfg *config.Config) http.Handler {
	backends := make([]*backend.Backend, 0, len(cfg.Backends))
	for _, b := range cfg.Backends {
		hc := b.EffectiveHealthCheck(cfg.HealthCheck)
		backends = append(backends, &backend.Backend{
			Name:           b.Name,
			Addr:           b.Addr(),
			HealthPath:     hc.Path,
			HealthInterval: hc.Interval,
			HealthTimeout:  hc.Timeout,
		})
	}
	pool := backend.NewPool(backends...)
	healthcheck.Start(pool)
	return proxy.NewHandler(balancer.New(pool))
}
