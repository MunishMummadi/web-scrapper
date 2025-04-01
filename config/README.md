# Configuration Component

This directory contains the configuration management for the web scraper.

## Overview

The configuration component is responsible for:

1. Loading configuration from environment variables and/or config files
2. Providing a structured configuration object to other components
3. Validating configuration values

## Configuration Options

The configuration includes settings for:

- **API**: Host, port, timeouts
- **Database**: SQLite connection parameters
- **Redis**: Connection parameters for the job queue
- **Crawler**: Concurrency, timeouts, rate limits
- **Proxies**: Proxy server configuration

## Environment Variables

The scraper uses environment variables for configuration, which can be set in a `.env` file. See `.env.example` for available options.

## Usage

The configuration is loaded at application startup:

```go
// Example of how configuration is loaded
cfg, err := config.Load()
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}
```

Other components receive their specific configuration sections as needed.
