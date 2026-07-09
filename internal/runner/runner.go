package runner

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/EdgarOrtegaRamirez/perfpulse/internal/config"
	"github.com/EdgarOrtegaRamirez/perfpulse/internal/timing"
)

// Runner executes a benchmark scenario.
type Runner struct {
	config *config.Config
	client *http.Client

	// Results
	timings  []timing.Timing
	mu       sync.Mutex
	start    time.Time
	end      time.Time
	stopChan chan struct{}

	// Progress
	doneCount atomic.Int64

	// Callbacks
	OnRequest func(int) // called on each request completion
}

// New creates a new Runner from a Config.
func New(cfg *config.Config) *Runner {
	var transport *http.Transport

	if cfg.HTTP2 {
		transport = &http.Transport{
			MaxIdleConns:        cfg.Concurrency * 2,
			MaxConnsPerHost:     cfg.Concurrency * 2,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			ForceAttemptHTTP2:   true,
		}
	} else {
		transport = &http.Transport{
			MaxIdleConns:        cfg.Concurrency * 2,
			MaxConnsPerHost:     cfg.Concurrency * 2,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			DisableKeepAlives:   !cfg.KeepAlive,
		}
	}

	return &Runner{
		config: cfg,
		client: &http.Client{
			Timeout:   cfg.Timeout.Duration,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		stopChan: make(chan struct{}),
	}
}

// RunScenario executes a full scenario and returns the aggregated result.
func (r *Runner) RunScenario(s *config.Scenario) (*timing.Result, error) {
	if s.BodyFile != "" {
		data, err := os.ReadFile(s.BodyFile)
		if err != nil {
			return nil, fmt.Errorf("reading body file: %w", err)
		}
		s.Body = string(data)
	}

	targetURL := s.URL
	duration := s.Duration.Duration
	concurrency := s.Concurrency

	if r.config.Duration.Duration > 0 {
		duration = r.config.Duration.Duration
	}
	if r.config.Requests > 0 {
		s.Requests = r.config.Requests
	}
	if r.config.Concurrency > 0 {
		concurrency = r.config.Concurrency
	}

	r.timings = make([]timing.Timing, 0, 1000)
	r.start = time.Now()

	if s.Requests > 0 {
		r.runFixedRequests(targetURL, s.Method, s.Headers, s.Body, concurrency, s.Requests)
	} else {
		r.runTimed(targetURL, s.Method, s.Headers, s.Body, concurrency, duration)
	}

	r.end = time.Now()

	elapsed := r.end.Sub(r.start)
	result := timing.ComputeStats(r.timings, elapsed)
	if result != nil {
		result.Name = s.Name
		result.URL = targetURL
		result.Method = s.Method
		result.Concurrency = concurrency
		result.Duration = duration.Round(time.Millisecond).String()
		result.Requests = s.Requests
	}
	return result, nil
}

func (r *Runner) runFixedRequests(targetURL, method string, headers map[string]string, body string, concurrency, total int) {
	var wg sync.WaitGroup
	work := make(chan struct{}, total)
	for i := 0; i < total; i++ {
		work <- struct{}{}
	}
	close(work)

	sem := make(chan struct{}, concurrency)
	for range work {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			t := r.doRequest(targetURL, method, headers, body)
			r.mu.Lock()
			r.timings = append(r.timings, t)
			r.mu.Unlock()
			r.doneCount.Add(1)
			if r.OnRequest != nil {
				r.OnRequest(int(r.doneCount.Load()))
			}
		}()
	}
	wg.Wait()
}

func (r *Runner) runTimed(targetURL, method string, headers map[string]string, body string, concurrency int, duration time.Duration) {
	var wg sync.WaitGroup
	deadline := time.Now().Add(duration)

	sem := make(chan struct{}, concurrency)
	for {
		select {
		case <-r.stopChan:
			return
		default:
			if time.Now().After(deadline) {
				return
			}
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				t := r.doRequest(targetURL, method, headers, body)
				r.mu.Lock()
				r.timings = append(r.timings, t)
				r.mu.Unlock()
				r.doneCount.Add(1)
				if r.OnRequest != nil {
					r.OnRequest(int(r.doneCount.Load()))
				}
			}()
		}
	}
}

func (r *Runner) doRequest(targetURL, method string, headers map[string]string, body string) timing.Timing {
	var t timing.Timing
	t.StartTime = time.Now()

	var reqBody io.Reader
	if body != "" {
		reqBody = bytes.NewBufferString(body)
	}

	req, err := http.NewRequest(method, targetURL, reqBody)
	if err != nil {
		t.Error = err.Error()
		t.Total = time.Since(t.StartTime)
		return t
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Apply client trace
	trace := t.ClientTrace()
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := r.client.Do(req)
	if err != nil {
		t.Error = err.Error()
		t.Total = time.Since(t.StartTime)
		return t
	}
	defer resp.Body.Close()

	t.StatusCode = resp.StatusCode
	bodyBytes, _ := io.Copy(io.Discard, resp.Body)
	t.ResponseSize = bodyBytes
	t.Total = time.Since(t.StartTime)

	return t
}

// Stop signals the runner to stop early.
func (r *Runner) Stop() {
	close(r.stopChan)
}

// RunURLsBenchmark runs a benchmark against multiple URLs sequentially.
func RunURLsBenchmark(cfg *config.Config, urls []string) ([]*timing.Result, error) {
	var results []*timing.Result

	for _, targetURL := range urls {
		u, err := url.Parse(targetURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Skipping invalid URL %s: %v\n", targetURL, err)
			continue
		}
		if u.Scheme == "" {
			targetURL = "http://" + targetURL
		}

		method := cfg.Method
		if method == "" {
			method = "GET"
		}

		scenario := &config.Scenario{
			Name:        u.Host,
			URL:         targetURL,
			Method:      method,
			Headers:     cfg.Headers,
			Body:        cfg.Body,
			BodyFile:    cfg.BodyFile,
			Concurrency: cfg.Concurrency,
			Duration:    cfg.Duration,
			Requests:    cfg.Requests,
			RampUp:      cfg.RampUp,
			Timeout:     cfg.Timeout,
			KeepAlive:   cfg.KeepAlive,
			HTTP2:       cfg.HTTP2,
		}

		runner := New(cfg)
		result, err := runner.RunScenario(scenario)
		if err != nil {
			return nil, fmt.Errorf("benchmarking %s: %w", targetURL, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// CheckThresholds verifies results against CI thresholds.
func CheckThresholds(result *timing.Result, s *config.Scenario) []string {
	var failures []string

	if s.MaxP99.Duration > 0 && result.LatencyP99 > s.MaxP99.Duration {
		failures = append(failures, fmt.Sprintf(
			"P99 latency %.0fms exceeds max %.0fms",
			result.LatencyP99.Seconds()*1000,
			s.MaxP99.Duration.Seconds()*1000,
		))
	}
	if s.MaxErrorPct > 0 && result.ErrorPct > s.MaxErrorPct {
		failures = append(failures, fmt.Sprintf(
			"Error rate %.1f%% exceeds max %.1f%%",
			result.ErrorPct, s.MaxErrorPct,
		))
	}
	if s.MinRPS > 0 && result.RPS < s.MinRPS {
		failures = append(failures, fmt.Sprintf(
			"RPS %.1f below minimum %.1f",
			result.RPS, s.MinRPS,
		))
	}

	return failures
}

// ValidateURL checks that a URL is well-formed for benchmarking.
func ValidateURL(rawURL string) error {
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Host == "" {
		return fmt.Errorf("URL must have a host")
	}
	return nil
}

// nolint:gosec
var _ = &tls.Config{InsecureSkipVerify: false}