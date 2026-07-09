# AGENTS.md — PerfPulse

## Overview
PerfPulse is a Go-based HTTP API performance benchmarking CLI. It measures request latency, throughput, and error rates using `net/http/httptrace` for detailed timing breakdowns.

## Key Files

| File | Purpose |
|------|---------|
| `cmd/perfpulse/main.go` | CLI entry point (cobra) |
| `internal/config/config.go` | YAML scenario parsing, Duration wrapper type |
| `internal/runner/runner.go` | HTTP request executor with concurrent workers |
| `internal/timing/timing.go` | httptrace.ClientTrace, aggregated statistics, percentile computation |
| `internal/reporter/reporter.go` | Text/JSON/Markdown output formatting |

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/fatih/color` — terminal color output
- `gopkg.in/yaml.v3` — YAML parsing for scenario files

## Build & Test

```bash
go build -o perfpulse ./cmd/perfpulse
go test ./...
go vet ./...
```

## Architecture Notes

1. **Config**: `Scenario` holds a full benchmark definition. `Config` is the CLI-level config that can override scenario fields. `Duration` wraps `time.Duration` with custom YAML marshal/unmarshal.

2. **Runner**: Uses goroutine pool with a semaphore channel for concurrency control. Supports both timed runs (wall clock deadline) and fixed-request-count runs. Graceful shutdown via `stopChan`.

3. **Timing**: `Timing` struct holds per-request breakdown. `ComputeStats` aggregates into `Result` with sorted percentiles. Uses bubble sort (intentional — small n, simple, dependency-free).

4. **Reporter**: Three output modes. Text uses `fatih/color` for ANSI highlighting. JSON uses `encoding/json`. Markdown produces GitHub-flavored tables.

5. **CI Mode**: `CheckThresholds` compares results against scenario thresholds. Called in `runBenchmark` after report generation. Returns non-zero exit code on failure.

## Adding Features

- **New output format**: Add a method to `Report` (e.g., `htmlReport()`) and wire it in `Print()`.
- **New request method**: Extend `doRequest()` in runner — it already handles any method via `http.NewRequest`.
- **New threshold**: Add field to `Scenario`, check it in `CheckThresholds`.

## Known Limitations

- Bubble sort for percentiles — fine for <10k requests, switch to `sort.Slice` for larger datasets
- No distributed/load-generator mode (single-machine only)
- No WebSocket support
- No streaming responses

## NaN Builders API Integration (Optional)

For projects that use PerfPulse with AI analysis, the NaN Builders API is available:

```go
import "github.com/openai/openai-go"

client := openai.NewClient(option.WithAPIKey(os.Getenv("NAN_API_KEY")), option.WithBaseURL("https://api.nan.builders/v1"))
```

See `.env.example` for configuration.