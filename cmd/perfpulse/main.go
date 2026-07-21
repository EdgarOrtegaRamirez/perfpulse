package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/EdgarOrtegaRamirez/perfpulse/internal/config"
	"github.com/EdgarOrtegaRamirez/perfpulse/internal/reporter"
	"github.com/EdgarOrtegaRamirez/perfpulse/internal/runner"
	"github.com/EdgarOrtegaRamirez/perfpulse/internal/timing"
)

var (
	method       string
	concurrency  int
	duration     string
	requests     int
	headers      []string
	body         string
	bodyFile     string
	timeout      string
	keepAlive    bool
	http2        bool
	format       string
	output       string
	rampUp       string
	verbose      bool
	urlFile      string
	scenarioFile string
	ciMode       bool
	maxP99       string
	maxErrorPct  float64
	minRPS       float64
	rateLimit    float64
	warmUp       string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "perfpulse [flags] <url>",
		Short: "PerfPulse — HTTP API Performance Benchmarking CLI",
		Long: `PerfPulse is a modern HTTP API performance benchmarking CLI tool.

It measures request latency, throughput, and error rates with detailed
timing breakdown (DNS, TCP, TLS, first byte). Supports CI/CD mode with
configurable thresholds.

Examples:
  perfpulse https://api.example.com
  perfpulse -c 50 -d 30s https://api.example.com/endpoint
  perfpulse --scenario benchmark.yaml
  perfpulse --url-file urls.txt -f json -o results.json
  perfpulse --ci --max-p99 500ms --max-error-pct 1 https://api.example.com`,
		Args: cobra.MaximumNArgs(1),
		RunE: runBenchmark,
	}

	// Global flags
	rootCmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method")
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of concurrent workers")
	rootCmd.Flags().StringVarP(&duration, "duration", "d", "10s", "Test duration (e.g., 30s, 1m, 1000)")
	rootCmd.Flags().IntVarP(&requests, "requests", "n", 0, "Fixed number of requests (overrides duration)")
	rootCmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Custom header (can be specified multiple times)")
	rootCmd.Flags().StringVarP(&body, "body", "b", "", "Request body")
	rootCmd.Flags().StringVar(&bodyFile, "body-file", "", "Read request body from file")
	rootCmd.Flags().StringVarP(&timeout, "timeout", "t", "30s", "Request timeout")
	rootCmd.Flags().BoolVarP(&keepAlive, "keep-alive", "k", true, "Use HTTP keep-alive")
	rootCmd.Flags().BoolVar(&http2, "http2", false, "Enable HTTP/2")
	rootCmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text, json, markdown")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Write output to file")
	rootCmd.Flags().StringVarP(&rampUp, "ramp-up", "r", "0s", "Ramp-up duration")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().StringVarP(&urlFile, "url-file", "U", "", "File with URLs (one per line)")
	rootCmd.Flags().StringVarP(&scenarioFile, "scenario", "s", "", "YAML scenario file")
	rootCmd.Flags().BoolVar(&ciMode, "ci", false, "CI mode (exit with error on threshold failure)")
	rootCmd.Flags().StringVar(&maxP99, "max-p99", "0", "P99 latency threshold (e.g., 500ms)")
	rootCmd.Flags().Float64Var(&maxErrorPct, "max-error-pct", 0, "Max error rate percentage")
	rootCmd.Flags().Float64Var(&minRPS, "min-rps", 0, "Minimum requests per second")
	rootCmd.Flags().Float64Var(&rateLimit, "rate-limit", 0, "Max requests per second (0 = unlimited)")
	rootCmd.Flags().StringVar(&warmUp, "warm-up", "0s", "Warm-up duration before measuring (e.g., 5s, 10s)")

	_ = rootCmd.Execute()
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	cfg := &config.Config{
		Method:      method,
		Concurrency: concurrency,
		Requests:    requests,
		Headers:     make(map[string]string),
		Body:        body,
		BodyFile:    bodyFile,
		KeepAlive:   keepAlive,
		HTTP2:       http2,
		Format:      format,
		Output:      output,
		Verbose:     verbose,
		MaxErrorPct: maxErrorPct,
		MinRPS:      minRPS,
		RateLimit:   rateLimit,
	}

	// Parse headers
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			cfg.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Parse durations
	if d, err := time.ParseDuration(duration); err == nil {
		cfg.Duration = config.Duration{Duration: d}
	}
	if t, err := time.ParseDuration(timeout); err == nil {
		cfg.Timeout = config.Duration{Duration: t}
	}
	if r, err := time.ParseDuration(rampUp); err == nil {
		cfg.RampUp = config.Duration{Duration: r}
	}
	if p99, err := time.ParseDuration(maxP99); err == nil {
		cfg.MaxP99 = config.Duration{Duration: p99}
	}
	if w, err := time.ParseDuration(warmUp); err == nil {
		cfg.WarmUp = config.Duration{Duration: w}
	}

	// Collect URLs
	var urls []string

	if scenarioFile != "" {
		return runScenarioFile(cfg, scenarioFile)
	}

	if urlFile != "" {
		loaded, err := config.LoadURLsFromFile(urlFile)
		if err != nil {
			return fmt.Errorf("loading URL file: %w", err)
		}
		urls = append(urls, loaded...)
	}

	if len(args) > 0 {
		urls = append(urls, args[0])
	}

	if len(urls) == 0 {
		return fmt.Errorf("no target URL provided. Use: perfpulse [flags] <url>")
	}

	// Run benchmark
	var results []*timing.Result

	runner_ := runner.New(cfg)

	for _, targetURL := range urls {
		if err := runner.ValidateURL(targetURL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", targetURL, err)
			continue
		}

		scenario := &config.Scenario{
			Name:        targetURL,
			URL:         targetURL,
			Method:      cfg.Method,
			Headers:     cfg.Headers,
			Body:        cfg.Body,
			BodyFile:    cfg.BodyFile,
			Concurrency: cfg.Concurrency,
			Duration:    cfg.Duration,
			Requests:    cfg.Requests,
			Timeout:     cfg.Timeout,
			KeepAlive:   cfg.KeepAlive,
			HTTP2:       cfg.HTTP2,
			RateLimit:   cfg.RateLimit,
			WarmUp:      cfg.WarmUp,
			MaxP99:      cfg.MaxP99,
			MaxErrorPct: cfg.MaxErrorPct,
			MinRPS:      cfg.MinRPS,
		}

		// Handle Ctrl+C gracefully
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Fprintln(os.Stderr, "\nReceived interrupt, stopping benchmark...")
			runner_.Stop()
		}()

		result, err := runner_.RunScenario(scenario)
		if err != nil {
			return fmt.Errorf("benchmark failed: %w", err)
		}
		results = append(results, result)
	}

	// Print report
	report := reporter.New(results)
	if err := report.Print(cfg.Format, cfg.Output); err != nil {
		return fmt.Errorf("printing report: %w", err)
	}

	// CI mode: check thresholds
	if ciMode {
		var allFailures []string
		for _, result := range results {
			if result == nil {
				continue
			}
			scenario := &config.Scenario{
				MaxP99:      cfg.MaxP99,
				MaxErrorPct: cfg.MaxErrorPct,
				MinRPS:      cfg.MinRPS,
			}
			failures := runner.CheckThresholds(result, scenario)
			if len(failures) > 0 {
				fmt.Fprintf(os.Stderr, "\nThreshold failures for %s:\n", result.URL)
				for _, f := range failures {
					fmt.Fprintf(os.Stderr, "  ✗ %s\n", f)
				}
				allFailures = append(allFailures, failures...)
			}
		}
		if len(allFailures) > 0 {
			os.Exit(1)
		}
		fmt.Println("\n✓ All thresholds passed")
	}

	return nil
}

func runScenarioFile(cfg *config.Config, path string) error {
	s, err := config.LoadScenario(path)
	if err != nil {
		return fmt.Errorf("loading scenario: %w", err)
	}

	// CLI flags override scenario values
	if cfg.Method != "" && cfg.Method != "GET" {
		s.Method = cfg.Method
	}
	if cfg.Concurrency > 0 {
		s.Concurrency = cfg.Concurrency
	}
	if cfg.Duration.Duration > 0 {
		s.Duration = cfg.Duration
	}
	if cfg.Requests > 0 {
		s.Requests = cfg.Requests
	}
	if cfg.Timeout.Duration > 0 {
		s.Timeout = cfg.Timeout
	}
	if len(cfg.Headers) > 0 {
		s.Headers = cfg.Headers
	}
	if cfg.Body != "" {
		s.Body = cfg.Body
	}
	if cfg.BodyFile != "" {
		s.BodyFile = cfg.BodyFile
	}
	if cfg.KeepAlive {
		s.KeepAlive = true
	}
	if cfg.HTTP2 {
		s.HTTP2 = true
	}

	runner_ := runner.New(cfg)
	result, err := runner_.RunScenario(s)
	if err != nil {
		return err
	}

	report := reporter.New([]*timing.Result{result})
	if err := report.Print(cfg.Format, cfg.Output); err != nil {
		return fmt.Errorf("printing report: %w", err)
	}

	if ciMode {
		failures := runner.CheckThresholds(result, s)
		if len(failures) > 0 {
			fmt.Fprintf(os.Stderr, "\nThreshold failures:\n")
			for _, f := range failures {
				fmt.Fprintf(os.Stderr, "  ✗ %s\n", f)
			}
			os.Exit(1)
		}
		fmt.Println("\n✓ All thresholds passed")
	}

	return nil
}
