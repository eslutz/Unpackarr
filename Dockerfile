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
    -o /unpackarr ./cmd/unpackarr

FROM alpine:3.21
RUN apk add --no-cache \
    ca-certificates \
    unrar \
    p7zip \
    unzip \
    wget \
    && adduser -D -u 1000 unpackarr
USER unpackarr
COPY --from=builder /unpackarr /usr/local/bin/
EXPOSE 8085
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
    CMD wget -q --spider http://localhost:8085/ping || exit 1
ENTRYPOINT ["/usr/local/bin/unpackarr"]
