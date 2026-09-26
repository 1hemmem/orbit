package main

import (
	"log/slog"
	"net/http"
	"time"

	"orbit/internal/backend"
	"orbit/internal/balancer"
	"orbit/internal/healthcheck"
	"orbit/internal/proxy"
)

func main() {
	pool := backend.NewPool("localhost:9001", "localhost:9002", "localhost:9003")
	rr := balancer.New(pool)
	healthcheck.Start(pool, 3*time.Second)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      proxy.NewHandler(rr),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	slog.Info("orbit listening", "addr", srv.Addr)
	slog.Error("server stopped", "err", srv.ListenAndServe())
}
