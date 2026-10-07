package e2e

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"orbit/internal/app"
	"orbit/internal/config"
)

func startMock(t *testing.T, name string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello from "+name)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func writeConfig(t *testing.T, mocks []*httptest.Server) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("health_check:\n  interval: 50ms\n  timeout: 500ms\n  path: \"/\"\n")
	sb.WriteString("backends:\n")
	for i, ts := range mocks {
		addr := ts.Listener.Addr().(*net.TCPAddr)
		fmt.Fprintf(&sb, "  - name: mock-%d\n    host: %s\n    port: %d\n", i, addr.IP.String(), addr.Port)
	}
	path := filepath.Join(t.TempDir(), "orbit.yaml")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func startBalancer(t *testing.T, cfgPath string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewUnstartedServer(app.NewHandler(cfg))
	var conns atomic.Int32
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			conns.Add(1)
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)
	return srv, &conns
}

func get(t *testing.T, client *http.Client, url string) (int, string) {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, strings.TrimSpace(string(body))
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func TestRotationPerRequest(t *testing.T) {
	mocks := []*httptest.Server{startMock(t, "mock-0"), startMock(t, "mock-1"), startMock(t, "mock-2")}
	srv, conns := startBalancer(t, writeConfig(t, mocks))
	client := srv.Client()

	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		status, body := get(t, client, srv.URL)
		if status != http.StatusOK {
			t.Fatalf("status: want 200, got %d", status)
		}
		seen[body] = true
	}
	if len(seen) != 3 {
		t.Fatalf("want 3 distinct backends, got %v", seen)
	}
	if got := conns.Load(); got != 1 {
		t.Fatalf("want requests over a single keep-alive connection, got %d connections", got)
	}
}

func TestFailover(t *testing.T) {
	mocks := []*httptest.Server{startMock(t, "mock-0"), startMock(t, "mock-1"), startMock(t, "mock-2")}
	srv, _ := startBalancer(t, writeConfig(t, mocks))
	client := srv.Client()

	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		_, body := get(t, client, srv.URL)
		seen[body] = true
	}
	if len(seen) != 3 {
		t.Fatalf("initial rotation: want 3 backends, got %v", seen)
	}

	mocks[1].Close()

	waitFor(t, 5*time.Second, func() bool {
		bodies := map[string]bool{}
		for i := 0; i < 4; i++ {
			status, body := get(t, client, srv.URL)
			if status != http.StatusOK {
				return false
			}
			bodies[body] = true
		}
		return len(bodies) == 2 && !bodies["hello from mock-1"]
	})
}

func TestServiceUnavailableWhenAllDown(t *testing.T) {
	mocks := []*httptest.Server{startMock(t, "mock-0"), startMock(t, "mock-1")}
	srv, _ := startBalancer(t, writeConfig(t, mocks))
	client := srv.Client()

	for _, m := range mocks {
		m.Close()
	}
	waitFor(t, 5*time.Second, func() bool {
		status, _ := get(t, client, srv.URL)
		return status == http.StatusServiceUnavailable
	})
}
