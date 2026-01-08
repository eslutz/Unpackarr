# Unpackarr

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

### Folder Watching

Standalone mode for apps not supported by `golift.io/starr` (e.g., Whisparr). See [docs/.env.example](docs/.env.example) for all options.

| Variable | Default | Description |
| --- | --- | --- |
| `WATCH_ENABLED` | `false` | Enable folder watching |
| `WATCH_PATHS` | `/downloads` | Comma-separated watch paths |
| `WATCH_INTERVAL` | `30s` | Directory scan interval |

#### Marker Files

When `EXTRACT_DELETE_ORIG` is set to `false`, Unpackarr uses hidden marker files to prevent re-extracting archives across restarts. This is essential for users who keep archives for seeding torrents.

- **Format**: `.<archive-name>.unpackarr` (e.g., `.movie.rar.unpackarr`)
- **Created**: After successful extraction
- **Cleanup**: Orphaned markers (where the archive no longer exists) are automatically removed on startup and at the configured `WATCH_CLEANUP_INTERVAL`
- **Multi-part archives**: One marker per main archive file (e.g., only `.movie.rar.unpackarr` for `movie.rar`, `movie.r00`, etc.)

**Note**: When `EXTRACT_DELETE_ORIG=true`, marker files are not created since archives are deleted after extraction.

### Timing

Timing configuration for \*arr app integrations (polling intervals, delays, retries). See [docs/.env.example](docs/.env.example) for all timing options.

### Webhook Notifications

Optional notifications to Discord, Slack, Gotify, or custom JSON endpoints. See [docs/.env.example](docs/.env.example) for all options.

| Variable | Default | Description |
| --- | --- | --- |
| `WEBHOOK_URL` | | Webhook endpoint URL |
| `WEBHOOK_TEMPLATE` | `discord` | Template: discord, slack, gotify, json |
| `WEBHOOK_EVENTS` | `extracted,failed` | Events: queued, extracting, extracted, failed |

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

## Architecture

- **Stateless design** — No persistence, safe to restart
- **Single binary** — Pure Go implementation, no external dependencies
- **Goroutine-based** — Concurrent polling and extraction
- **Channel-driven** — Clean communication between components

## Supported Archive Formats

- RAR (.rar, .r00-r99)
- ZIP (.zip)
- 7-Zip (.7z)
- TAR (.tar, .tar.gz, .tgz)
- GZIP (.gz)
- BZIP2 (.bz2)
- ISO (.iso)

## Contributing

### Building

```bash
go build -o unpackarr ./cmd/unpackarr
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
# Install dependencies
go mod download

# Run locally
export SONARR_URL=http://localhost:8989
export SONARR_API_KEY=your-key
go run ./cmd/unpackarr

# Build Docker image
docker build -t unpackarr .

# Run tests
go test ./...
```

## Credits

Built with:

- [golift.io/xtractr](https://github.com/golift/xtractr) — Archive extraction
- [golift.io/starr](https://github.com/golift/starr) — Starr API clients
- [golift.io/cnfg](https://github.com/golift/cnfg) — Environment configuration
