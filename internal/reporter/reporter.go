package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/EdgarOrtegaRamirez/perfpulse/internal/timing"
)

// Report formats benchmark results. It supports text, JSON, and markdown output.
type Report struct {
	Results []*timing.Result `json:"results"`
	Format  string
}

// New creates a new Report for the given results.
func New(results []*timing.Result) *Report {
	return &Report{Results: results}
}

// Print outputs the report in the requested format.
func (r *Report) Print(format, outputPath string) error {
	r.Format = format

	var output string
	switch format {
	case "json":
		output = r.jsonReport()
	case "markdown", "md":
		output = r.markdownReport()
	default:
		output = r.textReport()
	}

	if outputPath != "" {
		return os.WriteFile(outputPath, []byte(output), 0644)
	}
	fmt.Print(output)
	return nil
}

func (r *Report) textReport() string {
	var b strings.Builder

	for _, res := range r.Results {
		if res == nil {
			continue
		}

		b.WriteString(color.CyanString("═══════════════════════════════════════════════\n"))
		b.WriteString(color.CyanString(fmt.Sprintf("  PerfPulse Report: %s\n", res.Name)))
		b.WriteString(color.CyanString("═══════════════════════════════════════════════\n"))
		b.WriteString(fmt.Sprintf("  URL:        %s\n", res.URL))
		b.WriteString(fmt.Sprintf("  Method:     %s\n", res.Method))
		b.WriteString(fmt.Sprintf("  Concurrency: %d\n", res.Concurrency))
		b.WriteString(fmt.Sprintf("  Duration:    %s\n", res.Duration))
		b.WriteString("\n")

		// Summary
		b.WriteString(color.YellowString("  Summary:\n"))
		b.WriteString(fmt.Sprintf("    Total requests:   %d\n", res.TotalRequests))
		b.WriteString(fmt.Sprintf("    Successful:       %d\n", res.SuccessfulRequests))
		b.WriteString(fmt.Sprintf("    Failed:           %d\n", res.FailedRequests))
		errorColor := color.GreenString
		if res.ErrorPct > 5 {
			errorColor = color.RedString
		} else if res.ErrorPct > 1 {
			errorColor = color.YellowString
		}
		b.WriteString(fmt.Sprintf("    Error rate:       %s\n", errorColor(fmt.Sprintf("%.2f%%", res.ErrorPct))))
		b.WriteString(fmt.Sprintf("    RPS:              %.1f\n", res.RPS))
		b.WriteString(fmt.Sprintf("    Throughput:       %.2f MB/s\n", res.BytesPerSec/1024/1024))
		b.WriteString(fmt.Sprintf("    Transferred:      %s\n", formatBytes(res.BytesTransferred)))
		b.WriteString("\n")

		// Latency
		b.WriteString(color.YellowString("  Latency Distribution:\n"))
		b.WriteString(fmt.Sprintf("    Min:      %s\n", colorizeLatency(res.LatencyMin)))
		b.WriteString(fmt.Sprintf("    Median:   %s\n", colorizeLatency(res.LatencyMedian)))
		b.WriteString(fmt.Sprintf("    Mean:     %s\n", colorizeLatency(res.LatencyMean)))
		b.WriteString(fmt.Sprintf("    P75:      %s\n", colorizeLatency(res.LatencyP75)))
		b.WriteString(fmt.Sprintf("    P90:      %s\n", colorizeLatency(res.LatencyP90)))
		b.WriteString(fmt.Sprintf("    P95:      %s\n", colorizeLatency(res.LatencyP95)))
		b.WriteString(fmt.Sprintf("    P99:      %s\n", colorizeLatency(res.LatencyP99)))
		b.WriteString(fmt.Sprintf("    Max:      %s\n", colorizeLatency(res.LatencyMax)))
		b.WriteString("\n")

		// Timing breakdown
		b.WriteString(color.YellowString("  Timing Breakdown (mean):\n"))
		if res.DNSMean > 0 {
			b.WriteString(fmt.Sprintf("    DNS:        %s\n", res.DNSMean))
		}
		if res.TCPMean > 0 {
			b.WriteString(fmt.Sprintf("    TCP:        %s\n", res.TCPMean))
		}
		if res.TLSMean > 0 {
			b.WriteString(fmt.Sprintf("    TLS:        %s\n", res.TLSMean))
		}
		if res.FirstByteMean > 0 {
			b.WriteString(fmt.Sprintf("    First Byte: %s\n", res.FirstByteMean))
		}
		b.WriteString("\n")

		// Status codes table
		if len(res.StatusCodes) > 0 {
			b.WriteString(color.YellowString("  Status Codes:\n"))
			codes := make([]int, 0, len(res.StatusCodes))
			for code := range res.StatusCodes {
				codes = append(codes, code)
			}
			sort.Ints(codes)
			for _, code := range codes {
				count := res.StatusCodes[code]
				codeColor := color.GreenString
				if code >= 400 && code < 500 {
					codeColor = color.YellowString
				} else if code >= 500 {
					codeColor = color.RedString
				}
				b.WriteString(fmt.Sprintf("    %s: %d\n", codeColor(fmt.Sprintf("%d", code)), count))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (r *Report) jsonReport() string {
	data, err := json.MarshalIndent(r.Results, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(data)
}

func (r *Report) markdownReport() string {
	var b strings.Builder

	for _, res := range r.Results {
		if res == nil {
			continue
		}

		b.WriteString(fmt.Sprintf("# PerfPulse Report: %s\n\n", res.Name))
		b.WriteString(fmt.Sprintf("- **URL:** `%s`\n", res.URL))
		b.WriteString(fmt.Sprintf("- **Method:** `%s`\n", res.Method))
		b.WriteString(fmt.Sprintf("- **Concurrency:** %d\n", res.Concurrency))
		b.WriteString(fmt.Sprintf("- **Duration:** %s\n", res.Duration))
		b.WriteString("\n")

		b.WriteString("## Summary\n\n")
		b.WriteString("| Metric | Value |\n")
		b.WriteString("|--------|-------|\n")
		b.WriteString(fmt.Sprintf("| Total requests | %d |\n", res.TotalRequests))
		b.WriteString(fmt.Sprintf("| Successful | %d |\n", res.SuccessfulRequests))
		b.WriteString(fmt.Sprintf("| Failed | %d |\n", res.FailedRequests))
		b.WriteString(fmt.Sprintf("| Error rate | %.2f%% |\n", res.ErrorPct))
		b.WriteString(fmt.Sprintf("| RPS | %.1f |\n", res.RPS))
		b.WriteString(fmt.Sprintf("| Throughput | %.2f MB/s |\n", res.BytesPerSec/1024/1024))
		b.WriteString("\n")

		b.WriteString("## Latency\n\n")
		b.WriteString("| Percentile | Value |\n")
		b.WriteString("|------------|-------|\n")
		b.WriteString(fmt.Sprintf("| Min | %s |\n", res.LatencyMin))
		b.WriteString(fmt.Sprintf("| P50 (Median) | %s |\n", res.LatencyMedian))
		b.WriteString(fmt.Sprintf("| P75 | %s |\n", res.LatencyP75))
		b.WriteString(fmt.Sprintf("| P90 | %s |\n", res.LatencyP90))
		b.WriteString(fmt.Sprintf("| P95 | %s |\n", res.LatencyP95))
		b.WriteString(fmt.Sprintf("| P99 | %s |\n", res.LatencyP99))
		b.WriteString(fmt.Sprintf("| Max | %s |\n", res.LatencyMax))
		b.WriteString("\n")

		b.WriteString("## Timing Breakdown (Mean)\n\n")
		b.WriteString("| Phase | Value |\n")
		b.WriteString("|-------|-------|\n")
		if res.DNSMean > 0 {
			b.WriteString(fmt.Sprintf("| DNS | %s |\n", res.DNSMean))
		}
		if res.TCPMean > 0 {
			b.WriteString(fmt.Sprintf("| TCP | %s |\n", res.TCPMean))
		}
		if res.TLSMean > 0 {
			b.WriteString(fmt.Sprintf("| TLS | %s |\n", res.TLSMean))
		}
		if res.FirstByteMean > 0 {
			b.WriteString(fmt.Sprintf("| First Byte | %s |\n", res.FirstByteMean))
		}
		b.WriteString("\n")

		if len(res.StatusCodes) > 0 {
			b.WriteString("## Status Codes\n\n")
			b.WriteString("| Code | Count |\n")
			b.WriteString("|------|-------|\n")
			codes := make([]int, 0, len(res.StatusCodes))
			for code := range res.StatusCodes {
				codes = append(codes, code)
			}
			sort.Ints(codes)
			for _, code := range codes {
				b.WriteString(fmt.Sprintf("| %d | %d |\n", code, res.StatusCodes[code]))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func colorizeLatency(d time.Duration) string {
	ms := d.Seconds() * 1000
	switch {
	case ms < 100:
		return color.GreenString(d.String())
	case ms < 500:
		return color.YellowString(d.String())
	default:
		return color.RedString(d.String())
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}