package healthcheck

import (
	"log/slog"
	"net/http"
	"time"

	"orbit/internal/backend"
)

func Start(pool *backend.Pool) {
	for _, b := range pool.All() {
		go check(b)
	}
}

func check(b *backend.Backend) {
	client := &http.Client{Timeout: b.HealthTimeout}
	for range time.Tick(b.HealthInterval) {
		ok := ping(client, b)
		if was := b.Healthy(); was != ok {
			b.SetHealthy(ok)
			slog.Info("backend health changed", "name", b.Name, "addr", b.Addr, "healthy", ok)
		}
	}
}

func ping(client *http.Client, b *backend.Backend) bool {
	resp, err := client.Get("http://" + b.Addr + b.HealthPath)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
