package timing

import (
	"math"
	"testing"
	"time"
)

func TestClientTrace(t *testing.T) {
	timing := &Timing{StartTime: time.Now()}
	trace := timing.ClientTrace()
	if trace == nil {
		t.Fatal("ClientTrace returned nil")
	}

	// Verify the trace fields are populated
	if trace.DNSStart == nil {
		t.Error("DNSStart hook is nil")
	}
	if trace.DNSDone == nil {
		t.Error("DNSDone hook is nil")
	}
	if trace.ConnectStart == nil {
		t.Error("ConnectStart hook is nil")
	}
	if trace.ConnectDone == nil {
		t.Error("ConnectDone hook is nil")
	}
	if trace.TLSHandshakeStart == nil {
		t.Error("TLSHandshakeStart hook is nil")
	}
	if trace.TLSHandshakeDone == nil {
		t.Error("TLSHandshakeDone hook is nil")
	}
	if trace.GotFirstResponseByte == nil {
		t.Error("GotFirstResponseByte hook is nil")
	}
}

func TestComputeStatsEmpty(t *testing.T) {
	result := ComputeStats([]Timing{}, time.Second)
	if result != nil {
		t.Fatal("expected nil for empty timings")
	}
}

func TestComputeStatsBasic(t *testing.T) {
	timings := []Timing{
		{Total: 100 * time.Millisecond, StatusCode: 200, ResponseSize: 1000},
		{Total: 200 * time.Millisecond, StatusCode: 200, ResponseSize: 2000},
		{Total: 300 * time.Millisecond, StatusCode: 200, ResponseSize: 3000},
		{Total: 400 * time.Millisecond, StatusCode: 500, ResponseSize: 0, Error: "server error"},
		{Total: 500 * time.Millisecond, StatusCode: 404, ResponseSize: 500},
	}

	result := ComputeStats(timings, 5*time.Second)
	if result == nil {
		t.Fatal("ComputeStats returned nil")
	}

	if result.TotalRequests != 5 {
		t.Errorf("expected 5 total requests, got %d", result.TotalRequests)
	}
	if result.SuccessfulRequests != 3 {
		t.Errorf("expected 3 successful, got %d", result.SuccessfulRequests)
	}
	if result.FailedRequests != 2 {
		t.Errorf("expected 2 failed, got %d", result.FailedRequests)
	}

	// Check error percentage
	if math.Abs(result.ErrorPct-40.0) > 0.01 {
		t.Errorf("expected ~40%% error rate, got %.2f%%", result.ErrorPct)
	}

	// Check RPS
	if math.Abs(result.RPS-1.0) > 0.01 {
		t.Errorf("expected ~1.0 RPS, got %.2f", result.RPS)
	}

	// Check bytes transferred
	if result.BytesTransferred != 6500 {
		t.Errorf("expected 6500 bytes, got %d", result.BytesTransferred)
	}

	// Check status codes
	if result.StatusCodes[200] != 3 {
		t.Errorf("expected 3 x 200, got %d", result.StatusCodes[200])
	}
	if result.StatusCodes[404] != 1 {
		t.Errorf("expected 1 x 404, got %d", result.StatusCodes[404])
	}
	if result.StatusCodes[500] != 1 {
		t.Errorf("expected 1 x 500, got %d", result.StatusCodes[500])
	}
}

func TestPercentiles(t *testing.T) {
	timings := make([]Timing, 100)
	for i := 0; i < 100; i++ {
		timings[i] = Timing{
			Total:        time.Duration(i+1) * time.Millisecond,
			StatusCode:   200,
			ResponseSize: 100,
		}
	}

	result := ComputeStats(timings, time.Second)
	if result == nil {
		t.Fatal("ComputeStats returned nil")
	}

	// P50 ≈ 50ms
	if result.LatencyP50 < 45*time.Millisecond || result.LatencyP50 > 55*time.Millisecond {
		t.Errorf("P50 expected ~50ms, got %v", result.LatencyP50)
	}
	// P99 ≈ 99ms
	if result.LatencyP99 < 94*time.Millisecond || result.LatencyP99 > 104*time.Millisecond {
		t.Errorf("P99 expected ~99ms, got %v", result.LatencyP99)
	}
	// Min = 1ms
	if result.LatencyMin != 1*time.Millisecond {
		t.Errorf("min expected 1ms, got %v", result.LatencyMin)
	}
	// Max = 100ms
	if result.LatencyMax != 100*time.Millisecond {
		t.Errorf("max expected 100ms, got %v", result.LatencyMax)
	}
}

func TestTimingBreakdown(t *testing.T) {
	timings := []Timing{
		{
			Total:        100 * time.Millisecond,
			DNSResolve:   10 * time.Millisecond,
			TCPConnect:   20 * time.Millisecond,
			TLSHandshake: 30 * time.Millisecond,
			FirstByte:    40 * time.Millisecond,
			StatusCode:   200,
		},
		{
			Total:        200 * time.Millisecond,
			DNSResolve:   15 * time.Millisecond,
			TCPConnect:   25 * time.Millisecond,
			TLSHandshake: 35 * time.Millisecond,
			FirstByte:    45 * time.Millisecond,
			StatusCode:   200,
		},
	}

	result := ComputeStats(timings, 2*time.Second)
	if result == nil {
		t.Fatal("ComputeStats returned nil")
	}

	// DNS
	if result.DNSMin != 10*time.Millisecond {
		t.Errorf("DNS min expected 10ms, got %v", result.DNSMin)
	}
	if result.DNSMax != 15*time.Millisecond {
		t.Errorf("DNS max expected 15ms, got %v", result.DNSMax)
	}
	if result.DNSMean != 12500*time.Microsecond {
		t.Errorf("DNS mean expected 12.5ms, got %v", result.DNSMean)
	}

	// TCP
	if result.TCPMin != 20*time.Millisecond {
		t.Errorf("TCP min expected 20ms, got %v", result.TCPMin)
	}
	if result.TCPMax != 25*time.Millisecond {
		t.Errorf("TCP max expected 25ms, got %v", result.TCPMax)
	}

	// TLS
	if result.TLSMin != 30*time.Millisecond {
		t.Errorf("TLS min expected 30ms, got %v", result.TLSMin)
	}
	if result.TLSMax != 35*time.Millisecond {
		t.Errorf("TLS max expected 35ms, got %v", result.TLSMax)
	}

	// First Byte
	if result.FirstByteMin != 40*time.Millisecond {
		t.Errorf("FB min expected 40ms, got %v", result.FirstByteMin)
	}
	if result.FirstByteMax != 45*time.Millisecond {
		t.Errorf("FB max expected 45ms, got %v", result.FirstByteMax)
	}
}

func TestSingleTiming(t *testing.T) {
	timings := []Timing{
		{Total: 42 * time.Millisecond, StatusCode: 200, ResponseSize: 512},
	}

	result := ComputeStats(timings, time.Second)
	if result == nil {
		t.Fatal("ComputeStats returned nil")
	}

	if result.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", result.TotalRequests)
	}
	if result.LatencyMin != 42*time.Millisecond {
		t.Errorf("min expected 42ms, got %v", result.LatencyMin)
	}
	if result.LatencyMax != 42*time.Millisecond {
		t.Errorf("max expected 42ms, got %v", result.LatencyMax)
	}
	if result.LatencyMean != 42*time.Millisecond {
		t.Errorf("mean expected 42ms, got %v", result.LatencyMean)
	}
	if result.LatencyP50 != 42*time.Millisecond {
		t.Errorf("P50 expected 42ms, got %v", result.LatencyP50)
	}
	if result.RPS != 1.0 {
		t.Errorf("RPS expected 1.0, got %.2f", result.RPS)
	}
}

func TestSkippedDurationFields(t *testing.T) {
	timings := []Timing{
		{
			Total:        50 * time.Millisecond,
			StatusCode:   200,
			ResponseSize: 100,
			DNSResolve:   0, // Skip DNS
			TCPConnect:   0, // Skip TCP
			TLSHandshake: 0, // Skip TLS
			FirstByte:    0, // Skip FirstByte
		},
	}

	result := ComputeStats(timings, time.Second)
	if result == nil {
		t.Fatal("ComputeStats returned nil")
	}

	// When all breakdown fields are 0, they should remain 0
	if result.DNSMin != 0 {
		t.Errorf("expected DNS min 0, got %v", result.DNSMin)
	}
	if result.TCPMin != 0 {
		t.Errorf("expected TCP min 0, got %v", result.TCPMin)
	}
	if result.TLSMin != 0 {
		t.Errorf("expected TLS min 0, got %v", result.TLSMin)
	}
	if result.FirstByteMin != 0 {
		t.Errorf("expected FB min 0, got %v", result.FirstByteMin)
	}
}
