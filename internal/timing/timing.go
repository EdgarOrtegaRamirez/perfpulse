package timing

import (
	"crypto/tls"
	"net/http/httptrace"
	"time"
)

// Timing holds the breakdown of an individual HTTP request timing.
type Timing struct {
	DNSResolve   time.Duration `json:"dns_resolve"`
	TCPConnect   time.Duration `json:"tcp_connect"`
	TLSHandshake time.Duration `json:"tls_handshake"`
	FirstByte    time.Duration `json:"first_byte"`
	Total        time.Duration `json:"total"`
	StatusCode   int           `json:"status_code"`
	ResponseSize int64         `json:"response_size"`
	Error        string        `json:"error,omitempty"`
	StartTime    time.Time     `json:"-"`
}

// ClientTrace returns an httptrace.ClientTrace that populates Timing fields.
func (t *Timing) ClientTrace() *httptrace.ClientTrace {
	var dnsStart, tcpStart, tlsStart time.Time

	return &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			t.DNSResolve = time.Since(dnsStart)
		},
		ConnectStart: func(_, _ string) {
			tcpStart = time.Now()
		},
		ConnectDone: func(_, _ string, err error) {
			if err == nil {
				t.TCPConnect = time.Since(tcpStart)
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			if err == nil {
				t.TLSHandshake = time.Since(tlsStart)
			}
		},
		GotFirstResponseByte: func() {
			t.FirstByte = time.Since(t.StartTime)
		},
	}
}

// Result holds the aggregated results of a benchmark run.
type Result struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Method      string `json:"method"`
	Concurrency int    `json:"concurrency"`
	Duration    string `json:"duration"`
	Requests    int    `json:"requests"`

	TotalRequests      int   `json:"total_requests"`
	SuccessfulRequests int   `json:"successful_requests"`
	FailedRequests     int   `json:"failed_requests"`
	BytesTransferred   int64 `json:"bytes_transferred"`

	LatencyMin    time.Duration `json:"latency_min"`
	LatencyMax    time.Duration `json:"latency_max"`
	LatencyMean   time.Duration `json:"latency_mean"`
	LatencyMedian time.Duration `json:"latency_median"`
	LatencyP50    time.Duration `json:"latency_p50"`
	LatencyP75    time.Duration `json:"latency_p75"`
	LatencyP90    time.Duration `json:"latency_p90"`
	LatencyP95    time.Duration `json:"latency_p95"`
	LatencyP99    time.Duration `json:"latency_p99"`

	DNSMin        time.Duration `json:"dns_min"`
	DNSMax        time.Duration `json:"dns_max"`
	DNSMean       time.Duration `json:"dns_mean"`
	TCPMin        time.Duration `json:"tcp_min"`
	TCPMax        time.Duration `json:"tcp_max"`
	TCPMean       time.Duration `json:"tcp_mean"`
	TLSMin        time.Duration `json:"tls_min"`
	TLSMax        time.Duration `json:"tls_max"`
	TLSMean       time.Duration `json:"tls_mean"`
	FirstByteMin  time.Duration `json:"first_byte_min"`
	FirstByteMax  time.Duration `json:"first_byte_max"`
	FirstByteMean time.Duration `json:"first_byte_mean"`

	RPS         float64     `json:"rps"`
	BytesPerSec float64     `json:"bytes_per_sec"`
	ErrorPct    float64     `json:"error_pct"`
	StatusCodes map[int]int `json:"status_codes"`
}

// ComputeStats computes aggregate statistics from a slice of timings.
func ComputeStats(timings []Timing, duration time.Duration) *Result {
	if len(timings) == 0 {
		return nil
	}

	r := &Result{
		StatusCodes: make(map[int]int),
	}

	latencies := make([]time.Duration, len(timings))
	dnsTimes := make([]time.Duration, 0, len(timings))
	tcpTimes := make([]time.Duration, 0, len(timings))
	tlsTimes := make([]time.Duration, 0, len(timings))
	fbTimes := make([]time.Duration, 0, len(timings))

	var totalLat, totalDNS, totalTCP, totalTLS, totalFB time.Duration

	for i, t := range timings {
		latencies[i] = t.Total
		totalLat += t.Total
		r.TotalRequests++
		r.BytesTransferred += t.ResponseSize

		if t.Error != "" || t.StatusCode >= 400 {
			r.FailedRequests++
		} else {
			r.SuccessfulRequests++
		}
		r.StatusCodes[t.StatusCode]++

		if t.DNSResolve > 0 {
			dnsTimes = append(dnsTimes, t.DNSResolve)
			totalDNS += t.DNSResolve
		}
		if t.TCPConnect > 0 {
			tcpTimes = append(tcpTimes, t.TCPConnect)
			totalTCP += t.TCPConnect
		}
		if t.TLSHandshake > 0 {
			tlsTimes = append(tlsTimes, t.TLSHandshake)
			totalTLS += t.TLSHandshake
		}
		if t.FirstByte > 0 {
			fbTimes = append(fbTimes, t.FirstByte)
			totalFB += t.FirstByte
		}
	}

	// Sort for percentiles
	sortDurations(latencies)
	sortDurations(dnsTimes)
	sortDurations(tcpTimes)
	sortDurations(tlsTimes)
	sortDurations(fbTimes)

	n := len(latencies)
	r.LatencyMin = latencies[0]
	r.LatencyMax = latencies[n-1]
	r.LatencyMean = totalLat / time.Duration(n)
	r.LatencyMedian = percentile(latencies, 50)
	r.LatencyP50 = percentile(latencies, 50)
	r.LatencyP75 = percentile(latencies, 75)
	r.LatencyP90 = percentile(latencies, 90)
	r.LatencyP95 = percentile(latencies, 95)
	r.LatencyP99 = percentile(latencies, 99)

	if len(dnsTimes) > 0 {
		r.DNSMin = dnsTimes[0]
		r.DNSMax = dnsTimes[len(dnsTimes)-1]
		r.DNSMean = totalDNS / time.Duration(len(dnsTimes))
	}
	if len(tcpTimes) > 0 {
		r.TCPMin = tcpTimes[0]
		r.TCPMax = tcpTimes[len(tcpTimes)-1]
		r.TCPMean = totalTCP / time.Duration(len(tcpTimes))
	}
	if len(tlsTimes) > 0 {
		r.TLSMin = tlsTimes[0]
		r.TLSMax = tlsTimes[len(tlsTimes)-1]
		r.TLSMean = totalTLS / time.Duration(len(tlsTimes))
	}
	if len(fbTimes) > 0 {
		r.FirstByteMin = fbTimes[0]
		r.FirstByteMax = fbTimes[len(fbTimes)-1]
		r.FirstByteMean = totalFB / time.Duration(len(fbTimes))
	}

	durSec := duration.Seconds()
	if durSec > 0 {
		r.RPS = float64(r.TotalRequests) / durSec
		r.BytesPerSec = float64(r.BytesTransferred) / durSec
	}
	if r.TotalRequests > 0 {
		r.ErrorPct = float64(r.FailedRequests) / float64(r.TotalRequests) * 100
	}

	return r
}

func percentile(sorted []time.Duration, pct float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * pct / 100.0)
	return sorted[idx]
}

func sortDurations(d []time.Duration) {
	n := len(d)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if d[i] > d[j] {
				d[i], d[j] = d[j], d[i]
			}
		}
	}
}
