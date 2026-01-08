# Unpackarr Development Guide

## Architecture Overview

Unpackarr is a **stateless, event-driven extraction service** for the \*arr stack (Sonarr, Radarr, Lidarr, Readarr). Key architectural principles:

- **12-factor**: All configuration via environment variables using `golift.io/cnfg`, no config files
- **Polling-based**: Starr clients poll app queues at configurable intervals (default: 2m)
- **Queue-driven extraction**: Centralizes extraction through `internal/extract/Queue` using `golift.io/xtractr`
- **Callback pattern**: Extraction results flow back through callbacks to metrics and webhooks

### Component Structure

```
cmd/unpackarr/main.go          - Initialization, signal handling, component wiring
internal/config/               - Environment-based config loading (cnfg)
internal/extract/              - Queue (xtractr wrapper) + optional folder watcher
internal/starr/                - Per-app clients (sonarr, radarr, lidarr, readarr)
internal/health/               - HTTP server (/ping, /health, /metrics, /status)
internal/notify/               - Webhook notifications (Discord, Slack, Gotify, JSON)
```

**Data flow**: Starr client polls → filters by path/protocol → queues extraction → xtractr processes → callback → metrics + webhooks

## Critical Development Workflows

### Build & Test

```bash
go build -v ./cmd/unpackarr              # Basic build
go test ./...                            # Run all tests
go test -coverprofile=coverage.out ./... # Generate coverage (DON'T commit coverage.out)
golangci-lint run                        # Lint
```

**REQUIRED before completing work**:

1. Run `golangci-lint run` - must pass with 0 issues
2. Run `go test ./...` - all tests must pass
3. Ensure any new/modified code has proper unit test coverage
4. Fix all issues before declaring work complete

### Dependency Updates

```bash
go list -m -u all      # Check for available updates
go get -u ./...        # Update all dependencies
go mod tidy            # Remove unused dependencies and clean go.mod/go.sum
go test ./...          # Verify after updates
golangci-lint run      # Ensure no lint issues introduced
```

**Always ensure dependencies are up-to-date and clean**:

- Run `go mod tidy` after adding/removing imports to cleanup unused dependencies
- Check for outdated dependencies with `go list -m -u all` regularly
- Verify builds and tests pass after dependency updates

**Breaking change pattern**: After `golift.io/starr` v1.2.1 upgrade, `record.Protocol` changed from `string` to `starr.Protocol` type. Cast with `string(record.Protocol)` in all starr client files (`lidarr.go`, `radarr.go`, `readarr.go`, `sonarr.go`).

### Docker Build

```bash
docker build -t unpackarr:dev \
  --build-arg VERSION=dev \
  --build-arg COMMIT=$(git rev-parse HEAD) \
  --build-arg DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") .
```

## Project-Specific Conventions

### Config Struct Tags

Use `xml:` tags (not `json:`/`yaml:`) for cnfg environment parsing:

```go
type Config struct {
    HealthPort int    `xml:"health_port"`  // Maps to HEALTH_PORT env var
    LogLevel   string `xml:"log_level"`     // Maps to LOG_LEVEL env var
}
```

### Starr Client Pattern

Each starr app follows identical structure in `internal/starr/`:

1. Wrapper client embedding `*Client`
2. `New{App}()` constructor initializes base client + starts poller
3. `poll()` method: fetches queue → filters → queues extractions
4. **Type conversions**: Cast `starr.Protocol` to `string` when building `QueueItem`

### Error Handling

Use `formatError(app, operation, err)` helper in starr clients to wrap errors consistently.

### Health Endpoints

- `/ping`: Simple liveness (always 200)
- `/health`: Readiness (checks queue is alive)
- `/ready`: Deep check (verifies starr app connectivity)
- `/status`: JSON with queue state, app connections, uptime
- `/metrics`: Prometheus format

## Testing Patterns

- Tests live alongside implementation files (`*_test.go`)
- Use table-driven tests where appropriate (see `config_test.go`, `queue_test.go`)
- Mock external dependencies (starr apps, webhooks) in tests
- Coverage target: maintain existing coverage levels per package

## External Dependencies

### Core Libraries

- `golift.io/xtractr`: Archive extraction engine (pure Go using nwaples/rardecode, bodgit/sevenzip, etc.)
- `golift.io/starr`: Starr app API client library
- `golift.io/cnfg`: Environment-to-struct config parser

### Starr App Integration

- Poll queue endpoints (e.g., `/api/v3/queue?page=1&pageSize=100`)
- Filter by `TrackedDownloadState` (Radarr/Sonarr: `importPending`) or `Status` (all: `completed`)
- Match downloads against configured paths and protocols before extracting

## Common Pitfalls

1. **Environment variable naming** - MUST match struct tag format (e.g., `HEALTH_PORT`, not `HEALTHPORT`)
2. **Parallel extraction limit** - Default is 1, increase `EXTRACT_PARALLEL` for concurrent extractions
3. **Starr client registration** - Must call `server.RegisterClient()` in main.go for health checks
