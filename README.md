# Orbit : An Application Load Balancer

Minimal HTTP load balancer in Go, configured via `orbit.yaml`.

## Roadmap

- [x] HTTP reverse proxy with per-request round-robin balancing
- [x] Backend pool with lock-free atomic health status
- [x] Per-backend HTTP health checks (path/interval/timeout configurable)
- [x] YAML config: defaults, strict unknown-field rejection, validation, generated kebab-case names
- [x] E2E tests: rotation, failover, 503 when all backends are down
- [ ] Multiple listeners (config becomes a `listeners:` list)
- [ ] HTTPS/TLS termination (cert/key per listener, HTTP→HTTPS redirect)
- [ ] Backend metrics: in-flight requests, request/error counters, latency
- [ ] More algorithms: least outstanding, weighted round robin, random
- [ ] Health check thresholds (consecutive healthy/unhealthy checks)
- [ ] Host/path routing rules (ALB-style target groups)
- [ ] Sticky sessions (cookie-based affinity)
- [ ] Ops polish: graceful shutdown, X-Forwarded-\* headers, status endpoint
