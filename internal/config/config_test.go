package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "orbit.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(writeConfig(t, `
backends:
  - host: localhost
    port: 9001
  - host: localhost
    port: 9002
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":8080" {
		t.Errorf("listen: want :8080, got %q", cfg.Listen)
	}
	if cfg.Balancer.Algorithm != "round_robin" {
		t.Errorf("algorithm: want round_robin, got %q", cfg.Balancer.Algorithm)
	}
	want := HealthCheck{Interval: 3 * time.Second, Timeout: 2 * time.Second, Path: "/"}
	if cfg.HealthCheck != want {
		t.Errorf("health_check: want %+v, got %+v", want, cfg.HealthCheck)
	}
	if len(cfg.Backends) != 2 {
		t.Fatalf("backends: want 2, got %d", len(cfg.Backends))
	}
	if cfg.Backends[0].Addr() != "localhost:9001" {
		t.Errorf("addr: want localhost:9001, got %q", cfg.Backends[0].Addr())
	}
	for _, b := range cfg.Backends {
		if !namePattern.MatchString(b.Name) {
			t.Errorf("generated name %q is not kebab-case", b.Name)
		}
	}
}

func TestLoadFull(t *testing.T) {
	cfg, err := Load(writeConfig(t, `
listen: "127.0.0.1:9000"
balancer:
  algorithm: round_robin
health_check:
  interval: 5s
  timeout: 1s
  path: /healthz
backends:
  - name: zeta
    host: 10.0.0.1
    port: 8081
    health_check:
      interval: 10s
      timeout: 3s
      path: /ready
  - name: kappa
    host: 10.0.0.2
    port: 8082
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:9000" {
		t.Errorf("listen: got %q", cfg.Listen)
	}
	if cfg.Backends[0].Name != "zeta" || cfg.Backends[1].Name != "kappa" {
		t.Errorf("names: got %q, %q", cfg.Backends[0].Name, cfg.Backends[1].Name)
	}
	zeta := cfg.Backends[0].EffectiveHealthCheck(cfg.HealthCheck)
	if zeta != (HealthCheck{Interval: 10 * time.Second, Timeout: 3 * time.Second, Path: "/ready"}) {
		t.Errorf("zeta health check: got %+v", zeta)
	}
	kappa := cfg.Backends[1].EffectiveHealthCheck(cfg.HealthCheck)
	if kappa != cfg.HealthCheck {
		t.Errorf("kappa should inherit global health check, got %+v", kappa)
	}
}

func TestGeneratedNamesUnique(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("backends:\n")
	for i := 0; i < 10; i++ {
		sb.WriteString("  - host: localhost\n    port: 9001\n")
	}
	cfg, err := Load(writeConfig(t, sb.String()))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, b := range cfg.Backends {
		if seen[b.Name] {
			t.Fatalf("duplicate generated name %q", b.Name)
		}
		seen[b.Name] = true
	}
}

func TestBackendAddrIPv6(t *testing.T) {
	b := Backend{Host: "::1", Port: 9001}
	if got := b.Addr(); got != "[::1]:9001" {
		t.Errorf("want [::1]:9001, got %q", got)
	}
}

func TestLoadUnknownField(t *testing.T) {
	_, err := Load(writeConfig(t, "bogus: true\nbackends:\n  - host: localhost\n    port: 9001\n"))
	if err == nil {
		t.Fatal("want error for unknown field")
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{"no backends", "listen: \":8080\"\n", "at least one backend"},
		{"missing host", "backends:\n  - port: 9001\n", "host: required"},
		{"port zero", "backends:\n  - host: localhost\n    port: 0\n", "port: must be between"},
		{"port too big", "backends:\n  - host: localhost\n    port: 70000\n", "port: must be between"},
		{"bad name", "backends:\n  - name: Bad Name\n    host: localhost\n    port: 9001\n", "must be kebab-case"},
		{"duplicate names", "backends:\n  - name: a\n    host: localhost\n    port: 9001\n  - name: a\n    host: localhost\n    port: 9002\n", "duplicate"},
		{"bad algorithm", "balancer:\n  algorithm: random\nbackends:\n  - host: localhost\n    port: 9001\n", "only \"round_robin\" is supported"},
		{"bad listen", "listen: \"8080\"\nbackends:\n  - host: localhost\n    port: 9001\n", "listen"},
		{"zero global interval", "health_check:\n  interval: 0s\nbackends:\n  - host: localhost\n    port: 9001\n", "health_check.interval"},
		{"bad global path", "health_check:\n  path: health\nbackends:\n  - host: localhost\n    port: 9001\n", "must start with /"},
		{"bad backend path", "backends:\n  - host: localhost\n    port: 9001\n    health_check:\n      path: ready\n", "must start with /"},
		{"negative backend interval", "backends:\n  - host: localhost\n    port: 9001\n    health_check:\n      interval: -1s\n", "interval: must be positive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(writeConfig(t, tt.content))
			if err == nil {
				t.Fatalf("want error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %q", tt.wantErr, err)
			}
		})
	}
}
