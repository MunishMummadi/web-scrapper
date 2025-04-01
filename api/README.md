# API Component

This directory contains the API and web interface components of the web scraper.

## Overview

The API component provides:

1. A simple web interface for viewing scraped data and submitting new URLs
2. RESTful API endpoints for programmatic interaction with the scraper
3. Handlers for various operations like enqueueing URLs and retrieving data

## Key Files

- `data_view.go`: Implements the DataViewHandler which provides both the web interface and API endpoints

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Simple dashboard showing recent scraped pages and stats |
| `/api/data` | GET | Get scraped data as JSON with pagination support |
| `/api/stats` | GET | Get scraper statistics |
| `/api/jobs` | GET | List active scraping jobs |
| `/api/settings` | GET/POST | Get or update settings |

## Web Interface

The web interface is intentionally kept simple, with a basic HTML dashboard that:

1. Shows statistics about the scraper
2. Displays recently scraped pages
3. Provides a form to submit new URLs for scraping

## Usage

The API component is initialized in `main.go` and registered with the HTTP server. It uses the storage interface to retrieve and store data.

```go
// Example of how the API is initialized
dataViewHandler := api.NewDataViewHandler(storage)
dataViewHandler.RegisterRoutes(mux)
```
