# wproxy

HTTP reverse proxy with caching and rate limiting.

## Features

- Reverse proxy to upstream services
- Cache with ETag support (RFC 7234)
- Rate limiting (per-IP or per-API-key)
- Prometheus metrics
- Structured logging

## Install

```bash
go build -o proxy ./cmd/proxy
```

## Usage

```bash
# Start upstream
go run test/testserver/main.go &

# Start proxy
PROXY_UPSTREAM_URL=http://localhost:19000 ./proxy

# Test
curl http://localhost:8080/test
```

## Config

`config.yaml`:

```yaml
server:
  port: 8080

upstream:
  url: "http://localhost:9000"

cache:
  enabled: true
  max_size: 104857600

ratelimit:
  enabled: true
  requests_per_second: 100
  burst: 200
```

Run: `./proxy -config config.yaml`

Or use env vars:

```bash
export PROXY_UPSTREAM_URL=http://localhost:9000
export PROXY_SERVER_PORT=8080
./proxy
```

## Endpoints

- `/` - Proxy to upstream
- `/health` - Health check
- `/ready` - Readiness check
- `:9090/metrics` - Prometheus metrics

## Docker

```bash
docker build -t wproxy .
docker run -p 8080:8080 -p 9090:9090 -e PROXY_UPSTREAM_URL=http://upstream wproxy
```

## Tests

```bash
go test ./...              # Run tests
go test -bench=. ./...     # Benchmarks
```

See [BENCHMARKS.md](BENCHMARKS.md) for performance results.

## License

MIT
