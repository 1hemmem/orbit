package healthcheck

import (
	"log/slog"
	"net/http"
	"time"

	"orbit/internal/backend"
)

func Start(pool *backend.Pool, interval time.Duration) {
	client := &http.Client{Timeout: 2 * time.Second}
	go func() {
		for range time.Tick(interval) {
			for _, b := range pool.All() {
				ok := ping(client, b.Addr)
				if was := b.Healthy(); was != ok {
					b.SetHealthy(ok)
					slog.Info("backend health changed", "addr", b.Addr, "healthy", ok)
				}
			}
		}
	}()
}

func ping(client *http.Client, addr string) bool {
	resp, err := client.Get("http://" + addr)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
