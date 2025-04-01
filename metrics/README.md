# Metrics Component

This directory contains the metrics collection and reporting functionality for the web scraper.

## Overview

The metrics component is responsible for:

1. Collecting performance and operational metrics
2. Exposing metrics for Prometheus scraping
3. Tracking crawler statistics (pages scraped, errors, etc.)
4. Providing data for monitoring and alerting

## Key Metrics

The metrics collector tracks:

- **Scrape Rate**: Number of pages scraped per minute
- **Error Rate**: Percentage of scraping attempts that result in errors
- **Queue Size**: Number of URLs waiting to be processed
- **Response Times**: Time taken to fetch and process pages
- **Worker Utilization**: How busy the crawler workers are

## Prometheus Integration

The metrics are exposed via a Prometheus-compatible endpoint at `/metrics`, which can be scraped by a Prometheus server for visualization in tools like Grafana.

## Usage

The metrics collector is initialized in `main.go`:

```go
// Example of how metrics collector is initialized
metricsCollector := metrics.NewMetricsCollector()
```

Other components like the crawler use the metrics collector to record events:

```go
// Example of recording a successful scrape
metricsCollector.RecordScrape(url, duration, success)
```

The metrics endpoint is registered in the API server:

```go
// Example of registering the metrics endpoint
mux.Handle("/metrics", promhttp.Handler())
```
