Task Service
============
Overview
- Gin-based REST API for managing users and tasks (todo/doing/done).
- Persistence in PostgreSQL; optional Redis cache-aside for list endpoints.
- OpenAPI served at `/openapi.yaml`; Swagger UI at `/swagger`; Prometheus metrics at `/metrics`; basic tracing via OpenTelemetry.

Architecture (why this shape)
- Layered split keeps concerns isolated: `internal/delivery/http` (transport/validation), `internal/domain` (pure business rules, cache hinting), `internal/infrastructure` (Postgres/Redis implementations). This makes it easy to swap adapters (e.g., different cache/DB) and to unit-test the domain with fakes.
- Metrics/tracing/pprof live in `internal/observability` and are injected into the router so observability can be turned on/off without touching domain logic.
- CLI entry (`cmd/taskservice`) is a thin wrapper around the service; all runtime flags/env (DB/Redis/pprof) flow through `internal/config`, keeping the server code deterministic.
- Cache-aside is optional: if Redis is down, the service continues with DB reads/writes, which keeps the happy path resilient while still allowing cached list endpoints when available.

Run (Docker)
- Dev with hot reload: `make docker-up` (uses profile dev, Air, mounts source).
- Prod-ish build: `make docker-up-prod`.
- Observability (Prometheus): append `--profile observability` (e.g., `docker compose -f deploy/docker-compose.yml --profile dev --profile observability up --build`). Prometheus UI: `http://localhost:9090`. Shortcut: `make docker-up-all`.
- Stop: `make docker-down`.
- Images built via `deploy/Dockerfile`; compose file at `deploy/docker-compose.yml`.

Run (local without Docker)
- Prereqs: Go 1.24, PostgreSQL, optional Redis.
- Env vars (required): `DATABASE_URL=postgres://...`; optional `HTTP_PORT` (default 8080), `REDIS_HOST` (default localhost:6379), `SERVICE_NAME`.
- Dev-only profiling: set `PPROF_ENABLED=1` to expose `/debug/pprof/*` while the API is running.
- Start API: `make run` (or `go run ./cmd/taskservice serve`).

API quickstart (port 8080)
- Create user: `curl -X POST http://localhost:8080/users -H "Content-Type: application/json" -d '{"name":"Ali","email":"ali@example.com"}'`
- Get user: `curl http://localhost:8080/users/1`
- Create task: `curl -X POST http://localhost:8080/tasks -H "Content-Type: application/json" -d '{"user_id":1,"title":"Buy milk","status":"todo"}'`
- List tasks (filters/pagination): `curl "http://localhost:8080/tasks?user_id=1&status=todo&page=1&page_size=20"`
- Update task: `curl -X PUT http://localhost:8080/tasks/1 -H "Content-Type: application/json" -d '{"user_id":1,"title":"Buy milk","status":"doing"}'`
- Delete task: `curl -X DELETE http://localhost:8080/tasks/1`
- OpenAPI: `curl http://localhost:8080/openapi.yaml`

Observability
- Prometheus metrics: `/metrics` (requests_total, request_latency_histogram, tasks_count).
- Tracing: configured via OpenTelemetry exporter; defaults to stdout when enabled.
- pprof (opt-in): set `PPROF_ENABLED=1` to expose `/debug/pprof/*` (profile, heap, etc.). Recommended for local/dev only.

Quick load + pprof (dev)
- Start with pprof exposed: `PPROF_ENABLED=1 make docker-up` (or `PPROF_ENABLED=1 DATABASE_URL=... make run`).
- Seed one user: `curl -X POST http://localhost:8080/users -H "Content-Type: application/json" -d '{"name":"bench","email":"bench@example.com"}'`.
- Drive a short load (requires `hey`, install via `go install github.com/rakyll/hey@latest`): `hey -z 15s -c 20 -m POST -H "Content-Type: application/json" -d '{"user_id":1,"title":"bench","status":"todo"}' http://localhost:8080/tasks`.
- Capture CPU profile while load runs: `go tool pprof -top http://localhost:8080/debug/pprof/profile?seconds=15` (or `go tool pprof -http=:0 http://localhost:8080/debug/pprof/profile?seconds=15` for the UI). Heap sample: `go tool pprof http://localhost:8080/debug/pprof/heap`.

Benchmarks (why/how)
- Why two flavors: Go micro-benchmarks give a quick CPU-only baseline for the service layer and cache hit/miss paths; the HTTP load test shows end-to-end behavior (Gin + validation + Postgres/Redis when available).
- Go micro-benchmarks: run `GOCACHE=$(PWD)/.gocache go test -bench=ListTasks -run ^$ ./internal/domain/task | tee benchmarks/list_tasks.txt`. The sample output from a local laptop is in `benchmarks/list_tasks.txt`.
  - `BenchmarkListTasksNoCache`: service `ListTasks` with in-memory repository and cache disabled.
  - `BenchmarkListTasksCached`: same inputs with cache warmed; measures the cached code path.
- HTTP load (end-to-end): use the "Quick load + pprof" commands above; you can persist the `hey` report with `... | tee benchmarks/hey_tasks.txt`. This hits the real HTTP stack and database, so keep pprof enabled if you want profiling while it runs.

Tests & coverage
- Run tests with isolated cache: `make test` (uses `GOCACHE=$(PWD)/.gocache`).
- Coverage summary per package: `make cover-packages` (defaults to app code only: `./internal/... ./pkg/...`).
- Coverage profile: `make cover` (writes `coverage.out`, prints function summary). Default package set currently reports ~78.8%. Example snippet:
  - `make cover` →
    - `internal`: config 81%, delivery/http 75.2%, domain/task 83.6%, domain/user 82.8%, infra/postgres 74.5%, infra/redis 77.3%, observability 84.9%
    - `pkg/httperr`: 100%
    - total: 78.8% statements (`coverage.out`)
- Want full-repo coverage (including helper binaries)? Run `COVER_PKGS=./... make cover` (drops because of CLI-only paths).

Notes
- DB schema migrations live in `internal/infrastructure/postgres/migrations.go` and SQL files under `internal/infrastructure/postgres/migrations/`. Compose will run them on startup.
- Docker builds use `-buildvcs=false` to avoid VCS metadata requirements in containers.
