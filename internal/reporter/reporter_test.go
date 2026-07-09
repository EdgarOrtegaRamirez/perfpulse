package reporter

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/EdgarOrtegaRamirez/perfpulse/internal/timing"
)

func TestNew(t *testing.T) {
	r := New(nil)
	if r == nil {
		t.Fatal("New returned nil")
	}
}

func TestTextReport(t *testing.T) {
	results := []*timing.Result{
		{
			Name:               "test",
			URL:                "https://example.com",
			Method:             "GET",
			Concurrency:        10,
			Duration:           "10s",
			TotalRequests:      100,
			SuccessfulRequests: 95,
			FailedRequests:     5,
			BytesTransferred:   102400,
			LatencyMin:         10 * time.Millisecond,
			LatencyMedian:      50 * time.Millisecond,
			LatencyMean:        55 * time.Millisecond,
			LatencyP75:         75 * time.Millisecond,
			LatencyP90:         90 * time.Millisecond,
			LatencyP95:         95 * time.Millisecond,
			LatencyP99:         99 * time.Millisecond,
			LatencyMax:         200 * time.Millisecond,
			DNSMean:            5 * time.Millisecond,
			TCPMean:            10 * time.Millisecond,
			TLSMean:            15 * time.Millisecond,
			FirstByteMean:      30 * time.Millisecond,
			RPS:                10.0,
			BytesPerSec:        10240,
			ErrorPct:           5.0,
			StatusCodes:        map[int]int{200: 95, 500: 5},
		},
	}

	r := New(results)
	output := r.textReport()

	// Check key sections exist
	checks := []string{
		"PerfPulse Report",
		"example.com",
		"Summary",
		"Latency Distribution",
		"Timing Breakdown",
		"Status Codes",
		"P99",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("expected report to contain %q", check)
		}
	}
}

func TestJSONReport(t *testing.T) {
	results := []*timing.Result{
		{
			Name:          "test-json",
			URL:           "https://httpbin.org/get",
			Method:        "POST",
			TotalRequests: 50,
			RPS:           5.0,
			ErrorPct:      0.0,
		},
	}

	r := New(results)
	output := r.jsonReport()

	var decoded []*timing.Result
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(decoded) != 1 {
		t.Fatalf("expected 1 result, got %d", len(decoded))
	}
	if decoded[0].Name != "test-json" {
		t.Errorf("expected name 'test-json', got %q", decoded[0].Name)
	}
}

func TestMarkdownReport(t *testing.T) {
	results := []*timing.Result{
		{
			Name:               "benchmark-md",
			URL:                "https://api.example.com",
			Method:             "GET",
			Concurrency:        20,
			Duration:           "30s",
			TotalRequests:      300,
			SuccessfulRequests: 299,
			FailedRequests:     1,
			BytesTransferred:   512000,
			LatencyMin:         5 * time.Millisecond,
			LatencyMedian:      25 * time.Millisecond,
			LatencyMean:        30 * time.Millisecond,
			LatencyP75:         40 * time.Millisecond,
			LatencyP90:         50 * time.Millisecond,
			LatencyP95:         60 * time.Millisecond,
			LatencyP99:         80 * time.Millisecond,
			LatencyMax:         150 * time.Millisecond,
			RPS:                10.0,
			BytesPerSec:        17066.67,
			ErrorPct:           0.33,
			StatusCodes:        map[int]int{200: 299, 500: 1},
		},
	}

	r := New(results)
	output := r.markdownReport()

	checks := []string{
		"PerfPulse Report",
		"benchmark-md",
		"Summary",
		"Latency",
		"Timing Breakdown",
		"Status Codes",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("expected markdown to contain %q", check)
		}
	}
}

func TestPrintTextToFile(t *testing.T) {
	results := []*timing.Result{
		{
			Name:          "file-test",
			URL:           "https://example.com",
			Method:        "GET",
			TotalRequests: 10,
			RPS:           1.0,
		},
	}

	tmpFile, err := os.CreateTemp("", "report-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	r := New(results)
	if err := r.Print("text", tmpFile.Name()); err != nil {
		t.Fatalf("Print to file failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "PerfPulse Report") {
		t.Error("file output missing 'PerfPulse Report'")
	}
}

func TestPrintJSONToFile(t *testing.T) {
	results := []*timing.Result{
		{
			Name:          "json-file",
			URL:           "https://example.com",
			Method:        "GET",
			TotalRequests: 5,
		},
	}

	tmpFile, err := os.CreateTemp("", "report-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	r := New(results)
	if err := r.Print("json", tmpFile.Name()); err != nil {
		t.Fatalf("Print JSON to file failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	var decoded []*timing.Result
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid JSON in file: %v", err)
	}

	if decoded[0].Name != "json-file" {
		t.Errorf("expected name 'json-file', got %q", decoded[0].Name)
	}
}

func TestPrintMarkdownToFile(t *testing.T) {
	results := []*timing.Result{
		{
			Name:          "md-file",
			URL:           "https://example.com",
			Method:        "GET",
			TotalRequests: 5,
		},
	}

	tmpFile, err := os.CreateTemp("", "report-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	r := New(results)
	if err := r.Print("markdown", tmpFile.Name()); err != nil {
		t.Fatalf("Print markdown to file failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "PerfPulse Report") {
		t.Error("file output missing 'PerfPulse Report'")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestColorizeLatency(t *testing.T) {
	tests := []struct {
		d    time.Duration
		name string
	}{
		{50 * time.Millisecond, "fast"},
		{200 * time.Millisecond, "medium"},
		{1000 * time.Millisecond, "slow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorizeLatency(tt.d)
			if result == "" {
				t.Error("colorizeLatency returned empty string")
			}
			// Should contain the duration string
			if !strings.Contains(result, tt.d.String()) {
				t.Errorf("expected result to contain %q, got %q", tt.d.String(), result)
			}
		})
	}
}
