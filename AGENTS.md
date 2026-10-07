# Orbit Load Balancer

Minimal HTTP load balancer in Go, configured via `orbit.yaml`.

## Architecture

- `cmd/orbit`: Main load balancer (listens per `listeners` list in config; default `:8080`).
- `cmd/mockserver`: Standalone test backend server (use `-port` flag).
- `internal/config`: YAML config loader (`orbit.yaml`) with defaults, strict unknown-field rejection, and validation; generates kebab-case backend names when omitted.
- `internal/backend`: Lock-free pool of configured backends with atomic health statuses.
- `internal/balancer`: Lock-free round-robin implementation using a `sync/atomic` counter.
- `internal/healthcheck`: Per-backend HTTP health checks (path/interval/timeout configurable per backend).
- `internal/proxy`: `httputil.ReverseProxy` routing traffic to the next healthy backend.
- `internal/app`: Shared startup wiring (`app.NewHandler`) used by `cmd/orbit` and the e2e tests.
- `test/e2e`: End-to-end tests (mock backends + real HTTP requests) covering rotation, failover, and 503 when all backends are down.

## Commands

- **Build**: `go build ./...`
- **Test**: `go test ./...`
- **E2E test**: `go test -race ./test/e2e/`
- **Lint**: `gofmt -s -w . && go vet ./...`

## Live Testing Workflow

To test rotation and failover manually:

1. Start mock servers:
   `go run ./cmd/mockserver -port 9001 &`
   `go run ./cmd/mockserver -port 9002 &`
   `go run ./cmd/mockserver -port 9003 &`
2. Start the balancer:
   `go run ./cmd/orbit -config orbit.yaml &`
3. Test rotation: `for i in 1 2 3; do curl http://localhost:8080/; done`
4. Kill a mock server (e.g. 9002) to observe health check failover (flagged within ~3s).
