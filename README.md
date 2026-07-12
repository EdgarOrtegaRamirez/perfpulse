# PerfPulse

**HTTP API Performance Benchmarking CLI**

PerfPulse is a modern HTTP API performance benchmarking tool that measures request latency, throughput, and error rates with detailed timing breakdown (DNS, TCP, TLS, first byte). It supports CI/CD mode with configurable thresholds for automated performance regression testing.

## Features

- 🔄 **Concurrent benchmarking** with configurable worker count
- ⏱ **Detailed timing breakdown** — DNS, TCP, TLS, first byte, total
- 📊 **Rich output formats** — colorized text, JSON, Markdown
- 📈 **Percentile reporting** — P50, P75, P90, P95, P99 latency
- 🎯 **CI/CD mode** — exit with error code on threshold violations
- 📝 **YAML scenario files** — reusable benchmark configurations
- 🔗 **Multi-URL benchmarking** — test multiple endpoints sequentially
- 🛑 **Graceful shutdown** on SIGINT/SIGTERM
- ⚡ **HTTP/1.1 and HTTP/2 support**
- 📋 **Status code distribution** tracking
- 🚦 **Rate limiting** — cap requests per second for sustained load testing
- 🌡 **Warm-up phase** — prime connections before taking measurements to avoid cold-start skew

## Installation

### From source

```bash
git clone https://github.com/EdgarOrtegaRamirez/perfpulse.git
cd perfpulse
go build -o perfpulse ./cmd/perfpulse
```

### Pre-built binaries

Download the latest release from the [releases page](https://github.com/EdgarOrtegaRamirez/perfpulse/releases).

## Quick Start

### Basic usage

```bash
# Benchmark a single endpoint
perfpulse https://api.example.com

# With custom concurrency and duration
perfpulse -c 50 -d 30s https://api.example.com/endpoint

# JSON output to file
perfpulse -f json -o results.json https://api.example.com

# Rate-limited benchmark (100 req/s max)
perfpulse --rate-limit 100 https://api.example.com

# With warm-up phase (5 seconds of priming requests before measuring)
perfpulse --warm-up 5s -d 30s https://api.example.com

# Combine rate limit and warm-up for realistic sustained load testing
perfpulse --rate-limit 50 --warm-up 3s -c 10 -d 60s https://api.example.com
```

### YAML scenario file

Create `benchmark.yaml`:

```yaml
name: my-api-benchmark
url: https://api.example.com/v1/users
method: GET
concurrency: 20
duration: 30s
timeout: 10s
keep_alive: true
headers:
  Authorization: Bearer token
  Accept: application/json
max_p99: 500ms
max_error_pct: 1.0
min_rps: 100
```

Run:

```bash
perfpulse --scenario benchmark.yaml
```

### CI/CD mode

```bash
perfpulse --ci --max-p99 500ms --max-error-pct 1 --min-rps 100 https://api.example.com
```

Exit code 0 if all thresholds pass, 1 if any fail.

### Multi-URL benchmarking

```bash
# URLs from file
perfpulse --url-file urls.txt

# URL file + additional URL from argument
perfpulse --url-file urls.txt https://fallback.example.com
```

## Output Formats

### Text (default)

Colorized terminal output with latency distribution, timing breakdown, and status codes.

### JSON

Machine-readable output for programmatic consumption:

```bash
perfpulse -f json https://api.example.com
```

### Markdown

Great for CI pipeline reports:

```bash
perfpulse -f markdown -o report.md https://api.example.com
```

## Configuration

### CLI Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--concurrency` | `-c` | 10 | Number of concurrent workers |
| `--duration` | `-d` | 10s | Test duration (e.g., 30s, 1m) |
| `--requests` | `-n` | 0 | Fixed request count (overrides duration) |
| `--method` | `-X` | GET | HTTP method |
| `--header` | `-H` | | Custom header (can repeat) |
| `--body` | `-b` | | Request body |
| `--body-file` | | | Read body from file |
| `--timeout` | `-t` | 30s | Request timeout |
| `--keep-alive` | `-k` | true | HTTP keep-alive |
| `--http2` | | false | Enable HTTP/2 |
| `--format` | `-f` | text | Output format (text/json/markdown) |
| `--output` | `-o` | | Write output to file |
| `--ramp-up` | `-r` | 0s | Ramp-up duration |
| `--url-file` | `-U` | | File with URLs (one per line) |
| `--scenario` | `-s` | | YAML scenario file |
| `--ci` | | false | CI mode (exit code on threshold failure) |
| `--max-p99` | | 0 | P99 latency threshold |
| `--max-error-pct` | | 0 | Max error rate percentage |
| `--min-rps` | | 0 | Minimum requests per second |
| `--verbose` | `-v` | false | Verbose output |

## Architecture

```
perfpulse/
├── cmd/perfpulse/        # CLI entry point (cobra)
├── internal/
│   ├── config/           # YAML scenario parsing, config types
│   ├── runner/           # HTTP request executor, concurrency
│   ├── timing/           # Request timing breakdown, statistics
│   └── reporter/         # Output formatting (text/JSON/markdown)
├── .github/workflows/    # CI configuration
└── tests/                # Integration tests
```

The tool uses `net/http/httptrace.ClientTrace` for per-request DNS, TCP, TLS, and first-byte timing, with bubble-sort-based percentile computation in `internal/timing`.

## Security

- No hardcoded secrets or tokens
- Input validation for URLs, durations, and file paths
- Configurable timeouts prevent hanging
- Safe file operations (no path traversal)
- See `SECURITY.md` for details

## License

MIT — see [LICENSE](LICENSE)