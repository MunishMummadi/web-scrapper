# Database Component

This directory contains the storage interfaces and implementations for the web scraper.

## Overview

The database component is responsible for:

1. Providing a storage interface for scraped data
2. Implementing SQLite storage for persistent data
3. Managing database connections and transactions
4. Supporting pagination for data retrieval

## Key Features

- **Storage Interface**: Defines methods for storing and retrieving scraped data
- **SQLite Implementation**: Provides a persistent storage solution using SQLite
- **Pagination Support**: Supports retrieving data in paginated form
- **Content Hashing**: Stores content hashes to detect changes in scraped pages

## Implementation Details

As noted in previous work, the SQLite storage implementation includes methods for:
- `GetScrapedPagesCount`: Returns the total number of scraped pages in the database
- `GetScrapedPagesPaginated`: Retrieves a paginated list of scraped pages with limit and offset parameters

These methods are essential for the web interface to display data properly.

## Schema

The database schema includes tables for:
- Scraped pages (URL, content, timestamp, hash)
- Metadata (crawler statistics, settings)

## Usage

The storage is initialized in `main.go`:

```go
// Example of how storage is initialized
sqliteStorage, err := database.NewSQLiteStorage(cfg.Database)
if err != nil {
    log.Fatalf("Failed to initialize SQLite storage: %v", err)
}
defer sqliteStorage.Close()
```

Other components like the API and crawler use the storage interface to store and retrieve data.
