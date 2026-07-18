package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Scenario defines a complete benchmark scenario in YAML.
type Scenario struct {
	Name        string        `yaml:"name"`
	Method      string        `yaml:"method"`
	URL         string        `yaml:"url"`
	Headers     map[string]string `yaml:"headers"`
	Body        string        `yaml:"body"`
	BodyFile    string        `yaml:"body_file"`
	Concurrency int           `yaml:"concurrency"`
	Duration    Duration      `yaml:"duration"`
	Requests    int           `yaml:"requests"`
	RampUp      Duration      `yaml:"ramp_up"`
	Timeout     Duration      `yaml:"timeout"`
	KeepAlive   bool          `yaml:"keep_alive"`
	HTTP2       bool          `yaml:"http2"`
	RateLimit   float64       `yaml:"rate_limit"`
	WarmUp      Duration      `yaml:"warm_up"`

	// Thresholds for CI mode
	MaxP99      Duration  `yaml:"max_p99"`
	MaxErrorPct float64   `yaml:"max_error_pct"`
	MinRPS      float64   `yaml:"min_rps"`
}

// Duration wraps time.Duration for YAML parsing.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

// Config holds the full configuration from flags + optional YAML file.
type Config struct {
	// Targets
	URLs      []string `yaml:"urls,omitempty"`
	URLFile   string   `yaml:"url_file,omitempty"`

	// Request configuration
	Method    string            `yaml:"method"`
	Headers   map[string]string `yaml:"headers"`
	Body      string            `yaml:"body"`
	BodyFile  string            `yaml:"body_file"`

	// Load configuration
	Concurrency int  `yaml:"concurrency"`
	Duration    Duration `yaml:"duration"`
	Requests    int     `yaml:"requests"`
	RampUp      Duration `yaml:"ramp_up"`
	Timeout     Duration `yaml:"timeout"`
	KeepAlive   bool     `yaml:"keep_alive"`
	HTTP2       bool     `yaml:"http2"`
	RateLimit   float64  `yaml:"rate_limit"`
	WarmUp      Duration `yaml:"warm_up"`

	// Output
	Format    string `yaml:"format"`
	Output    string `yaml:"output"`
	Verbose   bool   `yaml:"verbose"`

	// CI thresholds
	MaxP99      Duration `yaml:"max_p99"`
	MaxErrorPct float64  `yaml:"max_error_pct"`
	MinRPS      float64  `yaml:"min_rps"`
}

// LoadScenario reads a YAML scenario file.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading scenario file: %w", err)
	}
	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing scenario file: %w", err)
	}
	if s.Name == "" {
		s.Name = strings.TrimSuffix(path, ".yaml")
	}
	if s.Method == "" {
		s.Method = "GET"
	}
	if s.Concurrency <= 0 {
		s.Concurrency = 10
	}
	if s.Duration.Duration <= 0 && s.Requests <= 0 {
		s.Duration = Duration{Duration: 10 * time.Second}
	}
	if s.Timeout.Duration <= 0 {
		s.Timeout = Duration{Duration: 30 * time.Second}
	}
	return &s, nil
}

// LoadURLsFromFile reads URLs from a file (one per line).
func LoadURLsFromFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var urls []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}
	return urls, nil
}