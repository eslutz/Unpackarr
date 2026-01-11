# Environment Variables

Unpackarr uses `golift.io/cnfg` for environment variable parsing. Variable names follow the struct hierarchy with underscore separators.

## General Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `HEALTH_PORT` | `9092` | Health check HTTP server port |
| `LOG_LEVEL` | `INFO` | Log level: `DEBUG`, `INFO`, `WARN`, `ERROR` |

## Extraction Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `EXTRACT_PARALLEL` | `1` | Number of parallel extractions |
| `EXTRACT_DELETE_ORIG` | `true` | Delete original archives after extraction |
| `EXTRACT_PASSWORDS` | | Comma-separated list of archive passwords |

## Timing Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `TIMING_POLL_INTERVAL` | `5m` | How often to poll *arr apps (e.g., `1m`, `30s`, `1h`) |
| `TIMING_STARR_TIMEOUT` | `30s` | API request timeout |

## Folder Watcher Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `WATCH_FOLDER_WATCH_ENABLED` | `false` | Enable folder watching |
| `WATCH_FOLDER_WATCH_PATHS` | `["/downloads"]` | Comma-separated paths to watch |
| `WATCH_MARKER_CLEANUP_INTERVAL` | `1h` | How often to clean orphaned markers |

## Webhook Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_URL` | | Webhook URL for notifications |
| `WEBHOOK_TEMPLATE` | `discord` | Template: `discord`, `slack`, `gotify`, `json` |
| `WEBHOOK_EVENTS` | `extracted,failed` | Comma-separated events to notify on |
| `WEBHOOK_TIMEOUT` | `10s` | Webhook request timeout |

## Sonarr Settings

| Variable | Required | Description |
|----------|----------|-------------|
| `SONARR_URL` | Yes | Sonarr URL (e.g., `http://sonarr:8989`) |
| `SONARR_API_KEY` | Yes | Sonarr API key |
| `SONARR_PATHS` | No | Comma-separated path prefixes to process (empty = all) |
| `SONARR_PROTOCOLS` | No | Comma-separated protocols to process: `torrent`, `usenet` (empty = all) |

## Radarr Settings

| Variable | Required | Description |
|----------|----------|-------------|
| `RADARR_URL` | Yes | Radarr URL (e.g., `http://radarr:7878`) |
| `RADARR_API_KEY` | Yes | Radarr API key |
| `RADARR_PATHS` | No | Comma-separated path prefixes to process (empty = all) |
| `RADARR_PROTOCOLS` | No | Comma-separated protocols to process: `torrent`, `usenet` (empty = all) |

## Lidarr Settings

| Variable | Required | Description |
|----------|----------|-------------|
| `LIDARR_URL` | Yes | Lidarr URL (e.g., `http://lidarr:8686`) |
| `LIDARR_API_KEY` | Yes | Lidarr API key |
| `LIDARR_PATHS` | No | Comma-separated path prefixes to process (empty = all) |
| `LIDARR_PROTOCOLS` | No | Comma-separated protocols to process: `torrent`, `usenet` (empty = all) |

## Readarr Settings

| Variable | Required | Description |
|----------|----------|-------------|
| `READARR_URL` | Yes | Readarr URL (e.g., `http://readarr:8787`) |
| `READARR_API_KEY` | Yes | Readarr API key |
| `READARR_PATHS` | No | Comma-separated path prefixes to process (empty = all) |
| `READARR_PROTOCOLS` | No | Comma-separated protocols to process: `torrent`, `usenet` (empty = all) |

## Docker Compose Example

```yaml
environment:
  # General
  HEALTH_PORT: 9092
  LOG_LEVEL: DEBUG

  # Extraction
  EXTRACT_PARALLEL: 2
  EXTRACT_DELETE_ORIG: true

  # Timing (Note: TIMING_ prefix required!)
  TIMING_POLL_INTERVAL: 1m
  TIMING_STARR_TIMEOUT: 30s

  # Radarr
  RADARR_URL: http://radarr:7878
  RADARR_API_KEY: your-api-key
  RADARR_PATHS: /media
  RADARR_PROTOCOLS: torrent

  # Sonarr
  SONARR_URL: http://sonarr:8989
  SONARR_API_KEY: your-api-key
  SONARR_PATHS: /media
  SONARR_PROTOCOLS: torrent
```

## Notes on cnfg Variable Naming

The `golift.io/cnfg` library uses struct field tags to determine environment variable names. For nested structs, it concatenates names with underscores:

- `Config.Timing.PollInterval` → `TIMING_POLL_INTERVAL`
- `Config.Extract.DeleteOrig` → `EXTRACT_DELETE_ORIG`
- `Config.Radarr.Paths` → `RADARR_PATHS`

This is why **`POLL_INTERVAL`** doesn't work - it must be **`TIMING_POLL_INTERVAL`**.
