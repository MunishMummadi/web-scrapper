# Proxy Component

This directory contains the proxy management functionality for the web scraper.

## Overview

The proxy component is responsible for:

1. Managing a pool of proxy servers
2. Rotating proxies for requests to avoid IP bans
3. Testing proxy health and availability
4. Providing HTTP clients configured with proxies

## Key Features

- **Proxy Pool**: Maintains a list of available proxy servers
- **Health Checking**: Periodically tests proxies to ensure they're working
- **Rotation Strategy**: Implements algorithms for selecting the best proxy for each request
- **Transparent Integration**: Provides HTTP clients that other components can use without worrying about proxy details

## Configuration

Proxies can be configured in the `.env` file or through environment variables:

```
PROXY_ENABLED=true
PROXY_LIST=http://user:pass@proxy1.example.com:8080,http://user:pass@proxy2.example.com:8080
PROXY_TEST_URL=https://httpbin.org/ip
```

## Usage

The proxy manager is initialized in `main.go`:

```go
// Example of how proxy manager is initialized
proxyManager, err := proxy.NewManager(cfg.Proxies)
if err != nil {
    log.Fatalf("Failed to initialize proxy manager: %v", err)
}
defer proxyManager.Close()
```

The crawler uses the proxy manager to get HTTP clients for making requests:

```go
// Example of getting an HTTP client with a proxy
client, err := proxyManager.GetClient()
if err != nil {
    // Handle error
}
// Use client for HTTP requests
```
