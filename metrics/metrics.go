package metrics

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
)

type Counter struct {
	val atomic.Uint64
}

func (c *Counter) Inc() {
	c.val.Add(1)
}
func (c *Counter) Get() uint64 {
	return c.val.Load()
}

type CounterVec struct {
	mu     sync.RWMutex
	counts map[string]*Counter
	name   string
	help   string
	labels []string
}

func NewCounterVec(name, help string, labels ...string) *CounterVec {
	return &CounterVec{
		counts: make(map[string]*Counter),
		name:   name,
		help:   help,
		labels: labels,
	}
}

func (cv *CounterVec) WithLabelValues(lvs ...string) *Counter {
	key := strings.Join(lvs, ",")
	cv.mu.RLock()
	if c, ok := cv.counts[key]; ok {
		cv.mu.RUnlock()
		return c
	}
	cv.mu.RUnlock()

	cv.mu.Lock()
	defer cv.mu.Unlock()
	if c, ok := cv.counts[key]; ok {
		return c
	}
	c := &Counter{}
	cv.counts[key] = c
	return c
}

type Gauge struct {
	val atomic.Int64
}

func (g *Gauge) Set(v int64) {
	g.val.Store(v)
}

func (g *Gauge) Get() int64 {
	return g.val.Load()
}

type GaugeVec struct {
	mu     sync.RWMutex
	gauges map[string]*Gauge
	name   string
	help   string
	labels []string
}

func NewGaugeVec(name, help string, labels ...string) *GaugeVec {
	return &GaugeVec{
		gauges: make(map[string]*Gauge),
		name:   name,
		help:   help,
		labels: labels,
	}
}

func (gv *GaugeVec) WithLabelValues(lvs ...string) *Gauge {
	key := strings.Join(lvs, ",")
	gv.mu.RLock()
	if g, ok := gv.gauges[key]; ok {
		gv.mu.RUnlock()
		return g
	}
	gv.mu.RUnlock()

	gv.mu.Lock()
	defer gv.mu.Unlock()
	if g, ok := gv.gauges[key]; ok {
		return g
	}
	g := &Gauge{}
	gv.gauges[key] = g
	return g
}

type HistogramVec struct {
	mu     sync.RWMutex
	sums   map[string]float64
	counts map[string]uint64
	name   string
	help   string
	labels []string
}

func NewHistogramVec(name, help string, labels ...string) *HistogramVec {
	return &HistogramVec{
		sums:   make(map[string]float64),
		counts: make(map[string]uint64),
		name:   name,
		help:   help,
		labels: labels,
	}
}

type HistogramVecObserver struct {
	hv  *HistogramVec
	key string
}

func (ho *HistogramVecObserver) Observe(v float64) {
	ho.hv.mu.Lock()
	defer ho.hv.mu.Unlock()
	ho.hv.sums[ho.key] += v
	ho.hv.counts[ho.key]++
}

func (hv *HistogramVec) WithLabelValues(lvs ...string) *HistogramVecObserver {
	key := strings.Join(lvs, ",")
	return &HistogramVecObserver{hv: hv, key: key}
}

var (
	HTTPRequestCount   = NewCounterVec("nicovideo_rss_http_requests_total", "Total HTTP requests by endpoint and status code", "endpoint", "status")
	HTMLParseCount     = NewCounterVec("nicovideo_rss_html_parse_total", "Total number of HTML parses by status", "status")
	NicoRequestCount   = NewCounterVec("nicovideo_rss_nico_requests_total", "Total number of requests to Nicovideo by status code", "status")
	NicoRetryCount     = &Counter{}
	CacheHitCount      = NewCounterVec("nicovideo_rss_cache_hits_total", "Total number of cache hits and misses", "status")
	FeedUpdateCount         = NewCounterVec("nicovideo_rss_feed_updates_total", "Total number of feed updates by feed name and status", "feed", "status")
	FeedUpdateDuration      = NewHistogramVec("nicovideo_rss_feed_update_duration_seconds", "Duration of feed updates in seconds", "feed")
	FeedLastUpdateTimestamp = NewGaugeVec("nicovideo_rss_feed_last_update_timestamp_seconds", "Timestamp of the last feed update in seconds since epoch", "feed")
)

// WritePrometheusFormat writes all metrics in Prometheus text format
func WritePrometheusFormat(w io.Writer) {
	writeCounter(w, "nicovideo_rss_nico_retries_total", "Total number of retry attempts made to Nicovideo", NicoRetryCount.Get())

	writeCounterVec(w, HTTPRequestCount)
	writeCounterVec(w, HTMLParseCount)
	writeCounterVec(w, NicoRequestCount)
	writeCounterVec(w, CacheHitCount)
	writeCounterVec(w, FeedUpdateCount)
	writeHistogramVec(w, FeedUpdateDuration)
	writeGaugeVec(w, FeedLastUpdateTimestamp)
}

func writeCounter(w io.Writer, name, help string, val uint64) {
	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w, "# TYPE %s counter\n", name)
	fmt.Fprintf(w, "%s %d\n", name, val)
}

func writeCounterVec(w io.Writer, cv *CounterVec) {
	cv.mu.RLock()
	defer cv.mu.RUnlock()
	if len(cv.counts) == 0 {
		return
	}
	fmt.Fprintf(w, "# HELP %s %s\n", cv.name, cv.help)
	fmt.Fprintf(w, "# TYPE %s counter\n", cv.name)

	for key, c := range cv.counts {
		lvs := strings.Split(key, ",")
		labelStr := formatLabels(cv.labels, lvs)
		fmt.Fprintf(w, "%s{%s} %d\n", cv.name, labelStr, c.Get())
	}
}

func writeHistogramVec(w io.Writer, hv *HistogramVec) {
	hv.mu.RLock()
	defer hv.mu.RUnlock()
	if len(hv.counts) == 0 {
		return
	}
	fmt.Fprintf(w, "# HELP %s %s\n", hv.name, hv.help)
	// We expose sum and count, not full histogram buckets to keep it simple.
	fmt.Fprintf(w, "# TYPE %s_sum counter\n", hv.name)
	for key := range hv.counts {
		lvs := strings.Split(key, ",")
		labelStr := formatLabels(hv.labels, lvs)
		fmt.Fprintf(w, "%s_sum{%s} %f\n", hv.name, labelStr, hv.sums[key])
		fmt.Fprintf(w, "%s_count{%s} %d\n", hv.name, labelStr, hv.counts[key])
	}
}

func formatLabels(names, values []string) string {
	var parts []string
	for i := range names {
		if i < len(values) {
			parts = append(parts, fmt.Sprintf(`%s="%s"`, names[i], values[i]))
		}
	}
	return strings.Join(parts, ",")
}

func writeGaugeVec(w io.Writer, gv *GaugeVec) {
	gv.mu.RLock()
	defer gv.mu.RUnlock()
	if len(gv.gauges) == 0 {
		return
	}
	fmt.Fprintf(w, "# HELP %s %s\n", gv.name, gv.help)
	fmt.Fprintf(w, "# TYPE %s gauge\n", gv.name)

	for key, g := range gv.gauges {
		lvs := strings.Split(key, ",")
		labelStr := formatLabels(gv.labels, lvs)
		fmt.Fprintf(w, "%s{%s} %d\n", gv.name, labelStr, g.Get())
	}
}
