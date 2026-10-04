package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"orbit/internal/backend"
	"orbit/internal/balancer"
	"orbit/internal/config"
	"orbit/internal/healthcheck"
	"orbit/internal/proxy"
)

func main() {
	configPath := flag.String("config", "orbit.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("config failed", "err", err)
		os.Exit(1)
	}

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
	rr := balancer.New(pool)
	healthcheck.Start(pool)

	srv := &http.Server{
		Addr:         cfg.Listen,
		Handler:      proxy.NewHandler(rr),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	slog.Info("orbit listening", "addr", srv.Addr, "backends", len(backends))
	slog.Error("server stopped", "err", srv.ListenAndServe())
}
