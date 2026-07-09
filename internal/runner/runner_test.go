package runner

import (
	"testing"
	"time"

	"github.com/EdgarOrtegaRamirez/perfpulse/internal/config"
	"github.com/EdgarOrtegaRamirez/perfpulse/internal/timing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://example.com", false},
		{"http://example.com/path", false},
		{"example.com", false}, // gets auto-prefixed
		{"", true},
		{"://", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckThresholdsNone(t *testing.T) {
	result := &timing.Result{
		LatencyP99: 100 * time.Millisecond,
		ErrorPct:   0.5,
		RPS:        50.0,
	}

	scenario := &config.Scenario{}
	failures := CheckThresholds(result, scenario)
	if len(failures) != 0 {
		t.Errorf("expected 0 failures, got %d: %v", len(failures), failures)
	}
}

func TestCheckThresholdsP99Failure(t *testing.T) {
	result := &timing.Result{
		LatencyP99: 500 * time.Millisecond,
		ErrorPct:   0,
		RPS:        100,
	}

	scenario := &config.Scenario{
		MaxP99: config.Duration{Duration: 200 * time.Millisecond},
	}

	failures := CheckThresholds(result, scenario)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
	if len(failures) > 0 && failures[0] != "P99 latency 500ms exceeds max 200ms" {
		t.Errorf("unexpected failure message: %s", failures[0])
	}
}

func TestCheckThresholdsP99Pass(t *testing.T) {
	result := &timing.Result{
		LatencyP99: 100 * time.Millisecond,
		ErrorPct:   0,
		RPS:        100,
	}

	scenario := &config.Scenario{
		MaxP99: config.Duration{Duration: 200 * time.Millisecond},
	}

	failures := CheckThresholds(result, scenario)
	if len(failures) != 0 {
		t.Errorf("expected 0 failures, got %d", len(failures))
	}
}

func TestCheckThresholdsErrorPctFailure(t *testing.T) {
	result := &timing.Result{
		LatencyP99: 50 * time.Millisecond,
		ErrorPct:   5.0,
		RPS:        100,
	}

	scenario := &config.Scenario{
		MaxErrorPct: 1.0,
	}

	failures := CheckThresholds(result, scenario)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
}

func TestCheckThresholdsMinRPSFailure(t *testing.T) {
	result := &timing.Result{
		LatencyP99: 50 * time.Millisecond,
		ErrorPct:   0,
		RPS:        10,
	}

	scenario := &config.Scenario{
		MinRPS: 100,
	}

	failures := CheckThresholds(result, scenario)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
}

func TestCheckThresholdsMultiple(t *testing.T) {
	result := &timing.Result{
		LatencyP99: 500 * time.Millisecond,
		ErrorPct:   5.0,
		RPS:        10,
	}

	scenario := &config.Scenario{
		MaxP99:      config.Duration{Duration: 200 * time.Millisecond},
		MaxErrorPct: 1.0,
		MinRPS:      100,
	}

	failures := CheckThresholds(result, scenario)
	if len(failures) != 3 {
		t.Errorf("expected 3 failures, got %d: %v", len(failures), failures)
	}
}