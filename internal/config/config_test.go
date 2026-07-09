package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadScenario(t *testing.T) {
	content := `name: test-scenario
method: POST
url: https://httpbin.org/post
concurrency: 5
duration: 5s
timeout: 10s
keep_alive: true
headers:
  Content-Type: application/json
body: '{"key": "value"}'
max_p99: 500ms
max_error_pct: 1.0
min_rps: 100
`
	tmpFile, err := os.CreateTemp("", "scenario-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	s, err := LoadScenario(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadScenario failed: %v", err)
	}

	if s.Name != "test-scenario" {
		t.Errorf("expected name 'test-scenario', got %q", s.Name)
	}
	if s.Method != "POST" {
		t.Errorf("expected method 'POST', got %q", s.Method)
	}
	if s.URL != "https://httpbin.org/post" {
		t.Errorf("expected URL 'https://httpbin.org/post', got %q", s.URL)
	}
	if s.Concurrency != 5 {
		t.Errorf("expected concurrency 5, got %d", s.Concurrency)
	}
	if s.Duration.Duration != 5*time.Second {
		t.Errorf("expected duration 5s, got %v", s.Duration.Duration)
	}
	if s.Timeout.Duration != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", s.Timeout.Duration)
	}
	if !s.KeepAlive {
		t.Error("expected keep_alive true")
	}
	if s.MaxP99.Duration != 500*time.Millisecond {
		t.Errorf("expected max_p99 500ms, got %v", s.MaxP99.Duration)
	}
	if s.MaxErrorPct != 1.0 {
		t.Errorf("expected max_error_pct 1.0, got %f", s.MaxErrorPct)
	}
	if s.MinRPS != 100 {
		t.Errorf("expected min_rps 100, got %f", s.MinRPS)
	}
	if s.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type header 'application/json', got %q", s.Headers["Content-Type"])
	}
}

func TestLoadScenarioDefaults(t *testing.T) {
	content := `url: https://example.com
`
	tmpFile, err := os.CreateTemp("", "scenario-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	s, err := LoadScenario(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadScenario failed: %v", err)
	}

	if s.Method != "GET" {
		t.Errorf("expected default method 'GET', got %q", s.Method)
	}
	if s.Concurrency != 10 {
		t.Errorf("expected default concurrency 10, got %d", s.Concurrency)
	}
	if s.Duration.Duration != 10*time.Second {
		t.Errorf("expected default duration 10s, got %v", s.Duration.Duration)
	}
	if s.Timeout.Duration != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", s.Timeout.Duration)
	}
}

func TestLoadScenarioNotFound(t *testing.T) {
	_, err := LoadScenario("/nonexistent/file.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadScenarioInvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "scenario-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("{{invalid yaml}}")
	tmpFile.Close()

	_, err = LoadScenario(tmpFile.Name())
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadURLsFromFile(t *testing.T) {
	content := `https://example.com
https://httpbin.org/get
# this is a comment
  https://api.example.com/v1
`
	tmpFile, err := os.CreateTemp("", "urls-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	urls, err := LoadURLsFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadURLsFromFile failed: %v", err)
	}

	if len(urls) != 3 {
		t.Fatalf("expected 3 URLs, got %d", len(urls))
	}
	if urls[0] != "https://example.com" {
		t.Errorf("expected 'https://example.com', got %q", urls[0])
	}
	if urls[2] != "https://api.example.com/v1" {
		t.Errorf("expected 'https://api.example.com/v1', got %q", urls[2])
	}
}

func TestDurationMarshalYAML(t *testing.T) {
	d := Duration{5 * time.Second}
	v, err := d.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML failed: %v", err)
	}
	if v != "5s" {
		t.Errorf("expected '5s', got %v", v)
	}
}

func TestDurationUnmarshalYAML(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"5s", 5 * time.Second, false},
		{"100ms", 100 * time.Millisecond, false},
		{"1m30s", 90 * time.Second, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			yamlContent := `duration: ` + tt.input + "\n"
			tmpFile, err := os.CreateTemp("", "dur-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.WriteString(yamlContent)
			tmpFile.Close()

			s, err := LoadScenario(tmpFile.Name())
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.Duration.Duration != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, s.Duration.Duration)
			}
		})
	}
}