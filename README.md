# Unpackarr

[![Workflow Status](https://github.com/eslutz/unpackarr/actions/workflows/release.yml/badge.svg)](https://github.com/eslutz/unpackarr/actions/workflows/release.yml)
[![Security Check](https://github.com/eslutz/unpackarr/actions/workflows/security.yml/badge.svg)](https://github.com/eslutz/unpackarr/actions/workflows/security.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/eslutz/unpackarr)](https://goreportcard.com/report/github.com/eslutz/unpackarr)
[![License](https://img.shields.io/github/license/eslutz/unpackarr)](LICENSE)
[![Release](https://img.shields.io/github/v/release/eslutz/unpackarr?color=007ec6)](https://github.com/eslutz/unpackarr/releases/latest)

Container-native archive extraction service for the *arr stack. Automatically extracts compressed downloads for Sonarr, Radarr, Lidarr, and Readarr.

## Features

- **Stateless** — No databases, no history, restart anytime
- **Environment-only config** — 12-factor compliant, no config files
- **Multi-app support** — Direct integration with Sonarr, Radarr, Lidarr, Readarr via [golift.io/starr](https://github.com/golift/starr)
- **Folder watching** — Optional directory scanning for standalone use
- **Webhook notifications** — Discord, Slack, Gotify, or custom JSON
- **Prometheus metrics** — Full observability with extraction stats
- **Container-native** — Alpine-based, healthchecks, non-root user

## Quick Start

```bash
docker run -d \
  --name unpackarr \
  -e SONARR_URL=http://sonarr:8989 \
  -e SONARR_API_KEY=your-api-key \
  -e RADARR_URL=http://radarr:7878 \
  -e RADARR_API_KEY=your-api-key \
  -v /path/to/downloads:/downloads \
  -p 8085:8085 \
  ghcr.io/eslutz/unpackarr:latest
```

## Configuration

All configuration is done via environment variables.

### Core Settings

Key settings (see [docs/.env.example](docs/.env.example) for all options):

| Variable | Default | Description |
| --- | --- | --- |
| `HEALTH_PORT` | `8085` | HTTP server port |
| `LOG_LEVEL` | `INFO` | Log level: DEBUG, INFO, WARN, ERROR |
| `EXTRACT_PARALLEL` | `1` | Concurrent extractions |
| `EXTRACT_DELETE_ORIG` | `true` | Delete archives after extraction |

### Timing

Duration values support: `s` (seconds), `m` (minutes), `h` (hours). Examples: `30s`, `5m`, `2h`, `90m`, `1h30m`

| Variable | Default | Description |
| --- | --- | --- |
| `POLL_INTERVAL` | `2m` | How often to check for new work (applies to both folder watching and *arr app queue polling) |
| `MARKER_CLEANUP_INTERVAL` | `1h` | How often to clean up orphaned marker files |

### Folder Watching

Standalone mode for apps not supported by `golift.io/starr` (e.g., Whisparr). See [docs/.env.example](docs/.env.example) for all options.

| Variable | Default | Description |
| --- | --- | --- |
| `FOLDER_WATCH_ENABLED` | `false` | Enable folder watching |
| `FOLDER_WATCH_PATHS` | `/downloads` | Comma-separated watch paths |

#### Marker Files

When `EXTRACT_DELETE_ORIG` is set to `false`, Unpackarr uses hidden marker files to prevent re-extracting archives across restarts. This is essential for users who keep archives for seeding torrents.

- **Format**: `.<archive-name>.unpackarr` (e.g., `.movie.rar.unpackarr`)
- **Created**: After successful extraction
- **Cleanup**: Orphaned markers (where the archive no longer exists) are automatically removed on startup and at the configured `MARKER_CLEANUP_INTERVAL`
- **Multi-part archives**: One marker per main archive file (e.g., only `.movie.rar.unpackarr` for `movie.rar`, `movie.r00`, etc.)

**Note**: When `EXTRACT_DELETE_ORIG=true`, marker files are not created since archives are deleted after extraction.

### Webhook Notifications

Optional notifications to Discord, Slack, Gotify, or custom JSON endpoints. See [docs/.env.example](docs/.env.example) for all options.

| Variable | Default | Description |
| --- | --- | --- |
| `WEBHOOK_URL` | | Webhook endpoint URL |
| `WEBHOOK_TEMPLATE` | `discord` | Template: discord, slack, gotify, json |
| `WEBHOOK_EVENTS` | `extracted,failed` | Events: queued, extracting, extracted, failed |

## Architecture

```txt
┌─────────────────────────────────────────────────────────────────────┐
│                            Unpackarr                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────┐  poll    ┌──────────────┐    ┌────────────────┐    │
│  │ Starr Apps  │─────────►│              │    │  Health Server │    │
│  │  (Sonarr,   │  queues  │  Extraction  │◄───│   HTTP :8085   │    │
│  │   Radarr,   │          │    Queue     │    │  (/ping,       │    │
│  │   Lidarr,   │          │              │    │   /metrics)    │    │
│  │   Readarr)  │          │  (xtractr)   │    └────────────────┘    │
│  └─────────────┘          └──────┬───────┘                          │
│                                  │                                  │
│  ┌─────────────┐  scan           │ extract                          │
│  │   Folder    │─────────────────┘                                  │
│  │   Watcher   │                  │                                 │
│  └─────────────┘                  ▼                                 │
│                           ┌───────────────┐                         │
│                           │   Callbacks   │                         │
│                           ├───────────────┤                         │
│                           │   • Metrics   │                         │
│                           │   • Webhooks  │                         │
│                           │   • Markers   │                         │
│                           └───────────────┘                         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Key Components:**

- **Starr Clients** — Poll *arr application queues at `POLL_INTERVAL` for completed downloads
- **Folder Watcher** — Scans configured directories for archives (standalone mode)
- **Extraction Queue** — Centralized queue using `golift.io/xtractr` with configurable parallelism
- **Health Server** — Exposes metrics, status, and health check endpoints
- **Callbacks** — Handle post-extraction actions (metrics recording, webhook notifications, marker file creation)

**Design Principles:**

- **Stateless design** — No persistence, safe to restart
- **Single binary** — Pure Go implementation, no external dependencies
- **Goroutine-based** — Concurrent polling and extraction
- **Channel-driven** — Clean communication between components

### *arr Apps (Sonarr, Radarr, Lidarr, Readarr)

These apps are supported via the [golift.io/starr](https://github.com/golift/starr) package. Each app uses the same pattern. Replace `{APP}` with `SONARR`, `RADARR`, `LIDARR`, or `READARR`. See [docs/.env.example](docs/.env.example) for detailed configuration.

| Variable | Default | Description |
| --- | --- | --- |
| `{APP}_URL` | | Base URL (e.g., <http://sonarr:8989>) |
| `{APP}_API_KEY` | | API key from app settings |
| `{APP}_PATHS` | `/downloads` | Comma-separated paths to monitor |

**Note**: Other *arr applications (e.g., Whisparr) not listed above can use the [Folder Watching](#folder-watching) feature for automatic extraction.

## Health Endpoints

| Endpoint | Purpose |
| --- | --- |
| `/ping` | Liveness check |
| `/health` | Readiness check |
| `/ready` | Deep health check (verifies starr connectivity) |
| `/status` | Current queue and app status (JSON) |
| `/metrics` | Prometheus metrics |

### Example `/status` Response

```json
{
  "queue": {
    "waiting": 2,
    "extracting": 1
  },
  "folder_watcher": {
    "enabled": true,
    "paths": ["/downloads"]
  },
  "apps": {
    "sonarr": {"connected": true, "queue_items": 5},
    "radarr": {"connected": true, "queue_items": 3}
  },
  "uptime_seconds": 16320
}
```

## Metrics

Prometheus-compatible metrics at `/metrics`:

```prometheus
# Extraction metrics
unpackarr_extractions_total{source,status}
unpackarr_extraction_duration_seconds{source}
unpackarr_bytes_extracted_total{source}
unpackarr_files_extracted_total{source}
unpackarr_archives_processed_total{source}

# Queue metrics
unpackarr_queue_size{state}
unpackarr_starr_queue_items{app}
unpackarr_starr_connected{app}

# System metrics
unpackarr_start_time_seconds
```

## Grafana Dashboard

A pre-built Grafana dashboard is available at [docs/unpackarr-grafana-dashboard.json](docs/unpackarr-grafana-dashboard.json). Import it into your Grafana instance to visualize:

- Extraction success rates and failure counts
- Queue size and state (waiting, extracting)
- Extraction throughput (bytes, files, archives)
- Starr app connection status and queue items
- Go runtime metrics (memory, goroutines, CPU)
- Uptime and extraction duration trends

## Docker Compose

See [docker-compose.example.yml](docs/docker-compose.example.yml) for a complete example with Sonarr and Radarr.

```yaml
services:
  unpackarr:
    image: ghcr.io/eslutz/unpackarr:latest
    container_name: unpackarr
    environment:
      - SONARR_URL=http://sonarr:8989
      - SONARR_API_KEY=${SONARR_API_KEY}
      - RADARR_URL=http://radarr:7878
      - RADARR_API_KEY=${RADARR_API_KEY}
    volumes:
      - /path/to/downloads:/downloads
    ports:
      - "8085:8085"
    restart: unless-stopped
```

## Webhook Templates

### Discord

```bash
WEBHOOK_URL=https://discord.com/api/webhooks/xxx
WEBHOOK_TEMPLATE=discord
```

### Slack

```bash
WEBHOOK_URL=https://hooks.slack.com/services/xxx
WEBHOOK_TEMPLATE=slack
```

### Gotify

```bash
WEBHOOK_URL=https://gotify.example.com/message?token=xxx
WEBHOOK_TEMPLATE=gotify
```

### Custom JSON

```bash
WEBHOOK_TEMPLATE=json
```

JSON payload format:

```json
{
  "event": "extracted",
  "name": "Movie.Name.2024",
  "source": "radarr",
  "success": true,
  "started": "2024-01-05T10:30:00Z",
  "elapsed": "45s",
  "archives": 1,
  "files": 12,
  "size": 1073741824
}
```

## Supported Archive Formats

- RAR (.rar, .r00-r99)
- ZIP (.zip)
- 7-Zip (.7z)
- TAR (.tar, .tar.gz, .tgz)
- GZIP (.gz)
- BZIP2 (.bz2)
- ISO (.iso)

## Contributing

Contributions are welcome! Please follow these guidelines when submitting changes.

### Building from Source

```bash
# Clone the repository
git clone https://github.com/eslutz/unpackarr.git
cd unpackarr

# Install dependencies
go mod download

# Build binary
go build -o unpackarr ./cmd/unpackarr

# Build Docker image
docker build -t unpackarr .
```

With version information:

```bash
VERSION=0.1.0
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags="-s -w \
  -X github.com/eslutz/unpackarr/pkg/version.Version=$VERSION \
  -X github.com/eslutz/unpackarr/pkg/version.Commit=$COMMIT \
  -X github.com/eslutz/unpackarr/pkg/version.Date=$DATE" \
  -o unpackarr ./cmd/unpackarr
```

### Development

```bash
# Run tests
go test ./...

# Run tests with race detector and coverage
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# View coverage report
go tool cover -func=coverage.out

# Run linter
golangci-lint run

# Run locally
export SONARR_URL=http://localhost:8989
export SONARR_API_KEY=your-key
go run ./cmd/unpackarr
```

Before submitting a pull request:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run linters and tests
6. Submit a pull request

See our [Pull Request Template](.github/PULL_REQUEST_TEMPLATE.md) for more details.

## Security

Security is a top priority for this project. If you discover a security vulnerability, please follow responsible disclosure practices.

**Reporting Vulnerabilities:**

Please report security vulnerabilities through GitHub Security Advisories:
<https://github.com/eslutz/unpackarr/security/advisories/new>

Alternatively, you can view our [Security Policy](.github/SECURITY.md) for additional contact methods and guidelines.

**Security Best Practices:**

- Keep your installation up to date with the latest releases
- Use strong, unique API keys for *arr application integrations
- Avoid exposing the health/metrics port to the public internet
- Review and understand the volume mount permissions
- Regularly monitor logs for suspicious activity
- Ensure proper file permissions on watch directories

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

You are free to use, modify, and distribute this software under the terms of the MIT License.

## Acknowledgments

This project is built with and inspired by excellent open-source software:

- **[golift.io/xtractr](https://github.com/golift/xtractr)** - Archive extraction library for Go
- **[golift.io/starr](https://github.com/golift/starr)** - Starr application API clients
- **[golift.io/cnfg](https://github.com/golift/cnfg)** - Environment-based configuration
- **[Prometheus](https://prometheus.io/)** - Monitoring system and time series database
- **[unpackerr](https://github.com/Unpackerr/unpackerr)** - The original archive extraction tool for *arr applications (inspiration for this project)

## Related Projects

- **[Forwardarr](https://github.com/eslutz/forwardarr)** - Automatic port forwarding sync from Gluetun VPN to qBittorrent
- **[Torarr](https://github.com/eslutz/torarr)** - Tor SOCKS proxy container for the *arr stack with health monitoring
