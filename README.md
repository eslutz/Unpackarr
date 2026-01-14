# Unpackarr Wrapper

[![Workflow Status](https://github.com/eslutz/unpackarr/actions/workflows/release.yml/badge.svg)](https://github.com/eslutz/unpackarr/actions/workflows/release.yml)
[![Security Check](https://github.com/eslutz/unpackarr/actions/workflows/security.yml/badge.svg)](https://github.com/eslutz/unpackarr/actions/workflows/security.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/eslutz/unpackarr)](https://goreportcard.com/report/github.com/eslutz/unpackarr)
[![License](https://img.shields.io/github/license/eslutz/unpackarr)](LICENSE)
[![Release](https://img.shields.io/github/v/release/eslutz/unpackarr?color=007ec6)](https://github.com/eslutz/unpackarr/releases/latest)

Container-native wrapper around [Unpackerr](https://github.com/Unpackerr/unpackerr) that adds enhanced health monitoring and container features.

## Overview

This project wraps the official [Unpackerr](https://github.com/Unpackerr/unpackerr) binary and adds:

- **Enhanced health checks** — Container-native health endpoints on port 9092
- **Process monitoring** — Monitors the Unpackerr subprocess status
- **Unified logging** — Streams Unpackerr logs with clear prefixes
- **Simple configuration** — Minimal wrapper config, full Unpackerr passthrough

All archive extraction functionality is provided by the official Unpackerr binary. This wrapper focuses on providing better container integration and health monitoring.

## Features

- **Wrapper Layer:**
  - HTTP health server on port 9092 (liveness, readiness, status, metrics)
  - Process monitoring and lifecycle management
  - Unified log streaming with prefixes
  - Minimal configuration (just HEALTH_PORT and LOG_LEVEL for wrapper)

- **Unpackerr (Official Binary):**
  - Archive extraction for Sonarr, Radarr, Lidarr, Readarr
  - Folder watching for standalone use
  - Webhook notifications
  - All standard Unpackerr features

## Quick Start

```bash
docker run -d \
  --name unpackarr \
  -e UN_SONARR_0_URL=http://sonarr:8989 \
  -e UN_SONARR_0_API_KEY=your-api-key \
  -e UN_RADARR_0_URL=http://radarr:7878 \
  -e UN_RADARR_0_API_KEY=your-api-key \
  -v /path/to/downloads:/downloads \
  -p 9092:9092 \
  -p 5656:5656 \
  ghcr.io/eslutz/unpackarr:latest
```

**Ports:**
- `9092` - Wrapper health endpoints
- `5656` - Unpackerr web UI and API

## Configuration

### Wrapper Configuration

The wrapper itself only needs two environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `HEALTH_PORT` | `9092` | HTTP health server port for wrapper |
| `LOG_LEVEL` | `INFO` | Log level for wrapper: DEBUG, INFO, WARN, ERROR |

### Unpackerr Configuration

All other environment variables are passed through to Unpackerr. See the [official Unpackerr documentation](https://unpackerr.zip/docs/install/configuration) for full configuration options.

**Common Unpackerr variables:**

| Variable | Example | Description |
| --- | --- | --- |
| `UN_SONARR_0_URL` | `http://sonarr:8989` | Sonarr URL |
| `UN_SONARR_0_API_KEY` | `your-api-key` | Sonarr API key |
| `UN_RADARR_0_URL` | `http://radarr:7878` | Radarr URL |
| `UN_RADARR_0_API_KEY` | `your-api-key` | Radarr API key |
| `UN_PARALLEL` | `1` | Concurrent extractions |
| `UN_DELETE_ORIG` | `false` | Delete archives after extraction |

See [Unpackerr's configuration docs](https://unpackerr.zip/docs/install/configuration) for all options.

## Architecture

```txt
┌─────────────────────────────────────────────────────────────────────┐
│                       Unpackarr Wrapper                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────┐                      ┌────────────────┐       │
│  │  Wrapper Layer  │                      │  Health Server │       │
│  │   (Go Binary)   │───────monitors──────►│   HTTP :9092   │       │
│  │                 │                      │  (/ping,       │       │
│  │  • Subprocess   │                      │   /health,     │       │
│  │    management   │                      │   /ready,      │       │
│  │  • Log stream   │                      │   /status,     │       │
│  │  • Monitoring   │                      │   /metrics)    │       │
│  └────────┬────────┘                      └────────────────┘       │
│           │                                                         │
│           │ spawns                                                  │
│           ▼                                                         │
│  ┌──────────────────────────────────────────────────────┐          │
│  │              Official Unpackerr Binary               │          │
│  │                                                       │          │
│  │  • Archive extraction (xtractr)                      │          │
│  │  • *arr app integration (Sonarr/Radarr/etc)         │          │
│  │  • Folder watching                                   │          │
│  │  • Webhook notifications                             │          │
│  │  • Web UI and API (:5656)                            │          │
│  │  • All standard Unpackerr features                   │          │
│  └──────────────────────────────────────────────────────┘          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Key Components:**

- **Wrapper Layer** — Go binary that manages the Unpackerr subprocess, provides enhanced health endpoints, and streams logs
- **Official Unpackerr** — The proven [Unpackerr](https://github.com/Unpackerr/unpackerr) binary handles all extraction and *arr integration
- **Health Server** — Wrapper-provided HTTP endpoints for container orchestration

**Design Principles:**

- **Focused wrapper** — Minimal Go code, leverages official Unpackerr
- **Enhanced monitoring** — Better health checks for Kubernetes/Docker
- **Unified logging** — Wrapper streams and prefixes Unpackerr logs
- **Full compatibility** — All Unpackerr features and configuration passthrough

## Health Endpoints

Wrapper-provided endpoints for container health monitoring:

| Endpoint | Purpose |
| --- | --- |
| `/ping` | Liveness check (wrapper is running) |
| `/health` | Basic health check |
| `/ready` | Readiness check (Unpackerr subprocess is running) |
| `/status` | Current wrapper and Unpackerr status (JSON) |
| `/metrics` | Prometheus-compatible metrics |

### Example `/status` Response

```json
{
  "wrapper": {
    "uptime_seconds": 16320
  },
  "unpackerr": {
    "status": "running",
    "pid": 42
  }
}
```

## Metrics

Wrapper-provided Prometheus-compatible metrics at `/metrics`:

```prometheus
# Wrapper metrics
unpackarr_wrapper_start_time_seconds
unpackarr_process_running{status}  # 1=running, 0=stopped
```

For full extraction metrics, Unpackerr exposes its own metrics on port 5656 at `/metrics`.

## Docker Compose

See [docker-compose.example.yml](docs/docker-compose.example.yml) for a complete example with Sonarr and Radarr.

```yaml
services:
  unpackarr:
    image: ghcr.io/eslutz/unpackarr:latest
    container_name: unpackarr
    environment:
      # Wrapper configuration
      - HEALTH_PORT=9092
      - LOG_LEVEL=INFO
      
      # Unpackerr configuration (passthrough)
      - UN_SONARR_0_URL=http://sonarr:8989
      - UN_SONARR_0_API_KEY=${SONARR_API_KEY}
      - UN_RADARR_0_URL=http://radarr:7878
      - UN_RADARR_0_API_KEY=${RADARR_API_KEY}
      - UN_PARALLEL=1
      - UN_DELETE_ORIG=false
    volumes:
      - /path/to/downloads:/downloads
    ports:
      - "9092:9092"  # Wrapper health endpoints
      - "5656:5656"  # Unpackerr web UI
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:9092/ready"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
```

## Contributing

Contributions are welcome! This wrapper project focuses on container integration and health monitoring. For extraction functionality improvements, please contribute to the upstream [Unpackerr](https://github.com/Unpackerr/unpackerr) project.

### Building from Source

```bash
# Clone the repository
git clone https://github.com/eslutz/unpackarr.git
cd unpackarr

# Install dependencies
go mod download

# Build wrapper binary
go build -o unpackarr-wrapper ./cmd/unpackarr

# Build Docker image
docker build -t unpackarr-wrapper .
```

### Development

```bash
# Run tests
go test ./...

# Run linter
golangci-lint run

# Note: The wrapper requires the Unpackerr binary at /usr/local/bin/unpackerr
# For local development, you can download it manually:
wget https://github.com/Unpackerr/unpackerr/releases/download/v0.14.5/unpackerr.linux-amd64.tar.gz
tar -xzf unpackerr.linux-amd64.tar.gz
sudo mv unpackerr /usr/local/bin/

# Then run the wrapper
export HEALTH_PORT=9092
export LOG_LEVEL=DEBUG
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

This project wraps and enhances the excellent:

- **[Unpackerr](https://github.com/Unpackerr/unpackerr)** - The official archive extraction tool for *arr applications that powers this wrapper
- **[golift.io/cnfg](https://github.com/golift/cnfg)** - Environment-based configuration used by the wrapper

All extraction functionality is provided by the official Unpackerr project.

## Related Projects

- **[Forwardarr](https://github.com/eslutz/forwardarr)** - Automatic port forwarding sync from Gluetun VPN to qBittorrent
- **[Torarr](https://github.com/eslutz/torarr)** - Tor SOCKS proxy container for the *arr stack with health monitoring
