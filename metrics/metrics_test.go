package metrics

import (
	"strings"
	"sync"
	"testing"
)

// --- Counter ---

func TestCounter_IncAndGet(t *testing.T) {
	var c Counter
	if got := c.Get(); got != 0 {
		t.Fatalf("initial Get() = %d, want 0", got)
	}
	c.Inc()
	c.Inc()
	if got := c.Get(); got != 2 {
		t.Fatalf("after 2x Inc(), Get() = %d, want 2", got)
	}
}

// --- CounterVec ---

func TestCounterVec_WithLabelValues_NewEntry(t *testing.T) {
	cv := NewCounterVec("test_counter", "help", "label1", "label2")
	c := cv.WithLabelValues("a", "b")
	if c == nil {
		t.Fatal("WithLabelValues returned nil")
	}
	c.Inc()
	if got := c.Get(); got != 1 {
		t.Fatalf("Get() = %d, want 1", got)
	}
}

func TestCounterVec_WithLabelValues_Reuse(t *testing.T) {
	cv := NewCounterVec("test_counter_reuse", "help", "label")
	c1 := cv.WithLabelValues("x")
	c2 := cv.WithLabelValues("x")
	if c1 != c2 {
		t.Fatal("same label values should return the same *Counter pointer")
	}
	c1.Inc()
	if got := c2.Get(); got != 1 {
		t.Fatalf("c2.Get() = %d, want 1 (should share state)", got)
	}
}

func TestCounterVec_WithLabelValues_DifferentLabels(t *testing.T) {
	cv := NewCounterVec("test_counter_diff", "help", "label")
	ca := cv.WithLabelValues("a")
	cb := cv.WithLabelValues("b")
	if ca == cb {
		t.Fatal("different label values should return different *Counter pointers")
	}
	ca.Inc()
	ca.Inc()
	cb.Inc()
	if got := ca.Get(); got != 2 {
		t.Fatalf("ca.Get() = %d, want 2", got)
	}
	if got := cb.Get(); got != 1 {
		t.Fatalf("cb.Get() = %d, want 1", got)
	}
}

func TestCounterVec_WithLabelValues_Concurrent(t *testing.T) {
	cv := NewCounterVec("test_counter_concurrent", "help", "label")
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			cv.WithLabelValues("k").Inc()
		}()
	}
	wg.Wait()
	if got := cv.WithLabelValues("k").Get(); got != goroutines {
		t.Fatalf("concurrent Inc: Get() = %d, want %d", got, goroutines)
	}
}

// --- HistogramVec ---

func TestHistogramVec_Observe(t *testing.T) {
	hv := NewHistogramVec("test_histogram", "help", "feed")
	obs := hv.WithLabelValues("feed1")
	obs.Observe(1.5)
	obs.Observe(2.5)

	hv.mu.RLock()
	defer hv.mu.RUnlock()
	key := "feed1"
	if got := hv.counts[key]; got != 2 {
		t.Fatalf("counts[%q] = %d, want 2", key, got)
	}
	if got := hv.sums[key]; got != 4.0 {
		t.Fatalf("sums[%q] = %f, want 4.0", key, got)
	}
}

func TestHistogramVec_WithLabelValues_SameKey(t *testing.T) {
	hv := NewHistogramVec("test_histogram_reuse", "help", "feed")
	obs1 := hv.WithLabelValues("f")
	obs2 := hv.WithLabelValues("f")
	obs1.Observe(1.0)
	obs2.Observe(1.0)

	hv.mu.RLock()
	defer hv.mu.RUnlock()
	if got := hv.sums["f"]; got != 2.0 {
		t.Fatalf("sums[\"f\"] = %f, want 2.0", got)
	}
}

// --- formatLabels ---

func TestFormatLabels_Normal(t *testing.T) {
	got := formatLabels([]string{"a", "b"}, []string{"x", "y"})
	want := `a="x",b="y"`
	if got != want {
		t.Fatalf("formatLabels = %q, want %q", got, want)
	}
}

func TestFormatLabels_MoreNamesThanValues(t *testing.T) {
	// When values are fewer than names, extra names are skipped
	got := formatLabels([]string{"a", "b"}, []string{"x"})
	want := `a="x"`
	if got != want {
		t.Fatalf("formatLabels (more names) = %q, want %q", got, want)
	}
}

func TestFormatLabels_Empty(t *testing.T) {
	got := formatLabels([]string{}, []string{})
	if got != "" {
		t.Fatalf("formatLabels (empty) = %q, want \"\"", got)
	}
}

// --- WritePrometheusFormat ---

func TestWritePrometheusFormat_EmptyMetrics(t *testing.T) {
	// Use fresh CounterVec / HistogramVec (no entries) to verify empty output
	var buf strings.Builder
	// writeCounter always outputs the scalar counter
	writeCounter(&buf, "test_metric", "help text", 0)
	out := buf.String()
	if !strings.Contains(out, "# HELP test_metric help text") {
		t.Errorf("expected HELP line, got: %s", out)
	}
	if !strings.Contains(out, "# TYPE test_metric counter") {
		t.Errorf("expected TYPE line, got: %s", out)
	}
	if !strings.Contains(out, "test_metric 0") {
		t.Errorf("expected value line, got: %s", out)
	}
}

func TestWriteCounterVec_Empty(t *testing.T) {
	cv := NewCounterVec("empty_cv", "help")
	var buf strings.Builder
	writeCounterVec(&buf, cv)
	if buf.Len() != 0 {
		t.Errorf("expected empty output for empty CounterVec, got: %s", buf.String())
	}
}

func TestWriteCounterVec_WithEntries(t *testing.T) {
	cv := NewCounterVec("my_counter", "my help", "status")
	cv.WithLabelValues("ok").Inc()
	cv.WithLabelValues("ok").Inc()
	cv.WithLabelValues("err").Inc()

	var buf strings.Builder
	writeCounterVec(&buf, cv)
	out := buf.String()

	if !strings.Contains(out, "# HELP my_counter my help") {
		t.Errorf("missing HELP line in: %s", out)
	}
	if !strings.Contains(out, `status="ok"`) {
		t.Errorf("missing ok label in: %s", out)
	}
	if !strings.Contains(out, `status="err"`) {
		t.Errorf("missing err label in: %s", out)
	}
}

func TestWriteHistogramVec_Empty(t *testing.T) {
	hv := NewHistogramVec("empty_hv", "help", "feed")
	var buf strings.Builder
	writeHistogramVec(&buf, hv)
	if buf.Len() != 0 {
		t.Errorf("expected empty output for empty HistogramVec, got: %s", buf.String())
	}
}

func TestWriteHistogramVec_WithEntries(t *testing.T) {
	hv := NewHistogramVec("feed_duration", "duration help", "feed")
	hv.WithLabelValues("myfeed").Observe(0.5)
	hv.WithLabelValues("myfeed").Observe(1.5)

	var buf strings.Builder
	writeHistogramVec(&buf, hv)
	out := buf.String()

	if !strings.Contains(out, "# HELP feed_duration duration help") {
		t.Errorf("missing HELP line in: %s", out)
	}
	if !strings.Contains(out, "feed_duration_sum") {
		t.Errorf("missing _sum line in: %s", out)
	}
	if !strings.Contains(out, "feed_duration_count") {
		t.Errorf("missing _count line in: %s", out)
	}
	if !strings.Contains(out, `feed="myfeed"`) {
		t.Errorf("missing feed label in: %s", out)
	}
}

func TestWritePrometheusFormat_Integration_Empty(t *testing.T) {
	var buf strings.Builder
	WritePrometheusFormat(&buf)
	out := buf.String()
	if !strings.Contains(out, "nicovideo_rss_nico_retries_total") {
		t.Errorf("Expected output to contain nicovideo_rss_nico_retries_total, got: %s", out)
	}
}

func TestWritePrometheusFormat_Integration_WithData(t *testing.T) {
	// Increment global counters
	HTTPRequestCount.WithLabelValues("/test", "200").Inc()
	NicoRetryCount.Inc()
	
	var buf strings.Builder
	WritePrometheusFormat(&buf)
	out := buf.String()
	
	if !strings.Contains(out, `nicovideo_rss_http_requests_total{endpoint="/test",status="200"}`) {
		t.Errorf("Missing HTTPRequestCount metric in output: %s", out)
	}
	if !strings.Contains(out, "nicovideo_rss_nico_retries_total") {
		t.Errorf("Missing NicoRetryCount metric in output: %s", out)
	}
}

func TestCounterVec_WithLabelValues_DoubleCheckLock(t *testing.T) {
	// We want to force multiple goroutines to hit the lock block simultaneously.
	// Since we can't deterministically pause goroutines at exactly the right spot,
	// we run this many times to ensure the race condition occurs and covers the double check lock.
	for try := 0; try < 10000; try++ {
		cv := NewCounterVec("test_double_check", "help", "label")
		var wg sync.WaitGroup
		const goroutines = 10
		wg.Add(goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				cv.WithLabelValues("race_key").Inc()
			}()
		}
		wg.Wait()
	}
}
