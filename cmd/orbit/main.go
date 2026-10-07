package main

import (
	"flag"
	"log/slog"
	"net"
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

	handler := app.NewHandler(cfg)
	lns := make([]net.Listener, 0, len(cfg.Listeners))
	for _, l := range cfg.Listeners {
		ln, err := net.Listen("tcp", l.Addr)
		if err != nil {
			slog.Error("listen failed", "addr", l.Addr, "err", err)
			os.Exit(1)
		}
		lns = append(lns, ln)
	}

	errCh := make(chan error, len(lns))
	for _, ln := range lns {
		srv := &http.Server{
			Handler:      handler,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
		}
		slog.Info("orbit listening", "addr", ln.Addr(), "backends", len(cfg.Backends))
		go func() { errCh <- srv.Serve(ln) }()
	}
	slog.Error("server stopped", "err", <-errCh)
}
