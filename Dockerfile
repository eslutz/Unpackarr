# Build the wrapper
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
RUN go build -ldflags="-s -w \
  -X github.com/eslutz/unpackarr/pkg/version.Version=${VERSION} \
  -X github.com/eslutz/unpackarr/pkg/version.Commit=${COMMIT} \
  -X github.com/eslutz/unpackarr/pkg/version.Date=${DATE}" \
  -o /unpackarr-wrapper ./cmd/unpackarr

# Download the official Unpackerr binary
# To update to the latest version, check: https://github.com/Unpackerr/unpackerr/releases/latest
FROM alpine:3.23 AS unpackerr
ARG UNPACKERR_VERSION=0.14.5
RUN apk add --no-cache curl && \
    curl -fsSL "https://github.com/Unpackerr/unpackerr/releases/download/v${UNPACKERR_VERSION}/unpackerr.amd64.linux.gz" -o /tmp/unpackerr.gz && \
    gunzip /tmp/unpackerr.gz && \
    mv /tmp/unpackerr /usr/local/bin/unpackerr && \
    chmod +x /usr/local/bin/unpackerr

# Final image
FROM alpine:3.23
RUN apk add --no-cache ca-certificates wget \
  && adduser -D -u 1000 unpackarr
USER unpackarr
COPY --from=builder /unpackarr-wrapper /usr/local/bin/unpackarr-wrapper
COPY --from=unpackerr /usr/local/bin/unpackerr /usr/local/bin/unpackerr
EXPOSE 9092 5656
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
  CMD wget -q --spider http://localhost:9092/ping || exit 1
ENTRYPOINT ["/usr/local/bin/unpackarr-wrapper"]
