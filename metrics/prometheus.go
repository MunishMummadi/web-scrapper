package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector handles Prometheus metrics collection
type MetricsCollector struct {
	// Counters
	ScrapedPagesTotal      prometheus.Counter
	ScrapingErrorsTotal    prometheus.Counter
	QueuedURLsTotal        prometheus.Counter
	RobotsDisallowedTotal  prometheus.Counter
	CircuitBreakerTripsTotal prometheus.Counter
	ProxyFailuresTotal     prometheus.Counter

	// Gauges
	WorkersRunning         prometheus.Gauge
	QueueSize              prometheus.Gauge
	OpenCircuits           prometheus.Gauge
	HealthyProxies         prometheus.Gauge

	// Histograms
	ScrapingDuration       prometheus.Histogram
	ResponseSize           prometheus.Histogram

	// Summaries
	QueueLatency           prometheus.Summary
	ProcessingTime         prometheus.Summary
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		// Counters
		ScrapedPagesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "scraper_pages_scraped_total",
			Help: "The total number of pages scraped",
		}),
		ScrapingErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "scraper_errors_total",
			Help: "The total number of scraping errors",
		}),
		QueuedURLsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "scraper_urls_queued_total",
			Help: "The total number of URLs queued",
		}),
		RobotsDisallowedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "scraper_robots_disallowed_total",
			Help: "The total number of URLs disallowed by robots.txt",
		}),
		CircuitBreakerTripsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "scraper_circuit_breaker_trips_total",
			Help: "The total number of circuit breaker trips",
		}),
		ProxyFailuresTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "scraper_proxy_failures_total",
			Help: "The total number of proxy failures",
		}),

		// Gauges
		WorkersRunning: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "scraper_workers_running",
			Help: "The number of scraper workers currently running",
		}),
		QueueSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "scraper_queue_size",
			Help: "The current size of the scraping queue",
		}),
		OpenCircuits: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "scraper_open_circuits",
			Help: "The number of currently open circuits",
		}),
		HealthyProxies: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "scraper_healthy_proxies",
			Help: "The number of healthy proxies available",
		}),

		// Histograms
		ScrapingDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "scraper_scraping_duration_seconds",
			Help:    "The distribution of scraping durations",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // From 10ms to ~10s
		}),
		ResponseSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "scraper_response_size_bytes",
			Help:    "The distribution of response sizes",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 10), // From 1KB to ~1MB
		}),

		// Summaries
		QueueLatency: promauto.NewSummary(prometheus.SummaryOpts{
			Name:       "scraper_queue_latency_seconds",
			Help:       "The time URLs spend in the queue before being processed",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
		ProcessingTime: promauto.NewSummary(prometheus.SummaryOpts{
			Name:       "scraper_processing_time_seconds",
			Help:       "The time spent processing each URL",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
	}
}

// RecordScrapingDuration records the duration of a scraping operation
func (m *MetricsCollector) RecordScrapingDuration(duration time.Duration) {
	m.ScrapingDuration.Observe(duration.Seconds())
}

// RecordResponseSize records the size of a response
func (m *MetricsCollector) RecordResponseSize(size float64) {
	m.ResponseSize.Observe(size)
}

// RecordQueueLatency records how long a URL was in the queue
func (m *MetricsCollector) RecordQueueLatency(latency time.Duration) {
	m.QueueLatency.Observe(latency.Seconds())
}

// RecordProcessingTime records the time spent processing a URL
func (m *MetricsCollector) RecordProcessingTime(duration time.Duration) {
	m.ProcessingTime.Observe(duration.Seconds())
}

// IncrementScrapedPages increments the counter for scraped pages
func (m *MetricsCollector) IncrementScrapedPages() {
	m.ScrapedPagesTotal.Inc()
}

// IncrementScrapingErrors increments the counter for scraping errors
func (m *MetricsCollector) IncrementScrapingErrors() {
	m.ScrapingErrorsTotal.Inc()
}

// IncrementQueuedURLs increments the counter for queued URLs
func (m *MetricsCollector) IncrementQueuedURLs() {
	m.QueuedURLsTotal.Inc()
}

// IncrementRobotsDisallowed increments the counter for URLs disallowed by robots.txt
func (m *MetricsCollector) IncrementRobotsDisallowed() {
	m.RobotsDisallowedTotal.Inc()
}

// IncrementCircuitBreakerTrips increments the counter for circuit breaker trips
func (m *MetricsCollector) IncrementCircuitBreakerTrips() {
	m.CircuitBreakerTripsTotal.Inc()
}

// IncrementProxyFailures increments the counter for proxy failures
func (m *MetricsCollector) IncrementProxyFailures() {
	m.ProxyFailuresTotal.Inc()
}

// SetWorkersRunning sets the gauge for running workers
func (m *MetricsCollector) SetWorkersRunning(count int) {
	m.WorkersRunning.Set(float64(count))
}

// SetQueueSize sets the gauge for queue size
func (m *MetricsCollector) SetQueueSize(size int) {
	m.QueueSize.Set(float64(size))
}

// SetOpenCircuits sets the gauge for open circuits
func (m *MetricsCollector) SetOpenCircuits(count int) {
	m.OpenCircuits.Set(float64(count))
}

// SetHealthyProxies sets the gauge for healthy proxies
func (m *MetricsCollector) SetHealthyProxies(count int) {
	m.HealthyProxies.Set(float64(count))
}

// Handler returns an HTTP handler for exposing metrics
func Handler() http.Handler {
	return promhttp.Handler()
}
