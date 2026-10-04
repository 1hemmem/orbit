package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"orbit/internal/app"
	"orbit/internal/config"
)

func main() {
	configPath := flag.String("config", "orbit.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("config failed", "err", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:         cfg.Listen,
		Handler:      app.NewHandler(cfg),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	slog.Info("orbit listening", "addr", srv.Addr, "backends", len(cfg.Backends))
	slog.Error("server stopped", "err", srv.ListenAndServe())
}
