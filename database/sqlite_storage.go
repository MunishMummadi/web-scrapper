package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MunishMummadi/web-scrapper/config"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Storage defines the interface for data persistence
type Storage interface {
	SaveScrapedData(ctx context.Context, url string, scrapedAt time.Time, contentHash string) error
	GetLastScrapeTime(ctx context.Context, url string) (time.Time, error)
	GetScrapedPages(ctx context.Context, limit int) ([]Page, error)
	GetScrapedPagesCount(ctx context.Context) (int, error)
	GetScrapedPagesPaginated(ctx context.Context, limit int, offset int) ([]Page, error)
	Close() error
}

// SQLiteStorage implements the Storage interface using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite-based storage
func NewSQLiteStorage(cfg config.DatabaseConfig) (Storage, error) {
	// Ensure the directory for the database file exists
	dbDir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory %s: %w", dbDir, err)
	}

	db, err := sql.Open("sqlite3", cfg.FilePath+"?_journal_mode=WAL") // Use WAL mode for better concurrency
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database at %s: %w", cfg.FilePath, err)
	}

	// Ping DB to ensure connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	// Create table if it doesn't exist
	query := `
	CREATE TABLE IF NOT EXISTS scraped_pages (
		url TEXT PRIMARY KEY,
		scraped_at TIMESTAMP NOT NULL,
		content_hash TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_scraped_at ON scraped_pages (scraped_at);
	`
	_, err = db.Exec(query)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create scraped_pages table: %w", err)
	}

	return &SQLiteStorage{db: db}, nil
}

// Page represents a scraped web page
type Page struct {
	URL         string
	ScrapedAt   time.Time
	ContentHash string
}

// SaveScrapedData saves metadata about a scraped page
func (s *SQLiteStorage) SaveScrapedData(ctx context.Context, url string, scrapedAt time.Time, contentHash string) error {
	query := `
	INSERT INTO scraped_pages (url, scraped_at, content_hash)
	VALUES (?, ?, ?)
	ON CONFLICT(url) DO UPDATE SET
		scraped_at = excluded.scraped_at,
		content_hash = excluded.content_hash;
	`
	_, err := s.db.ExecContext(ctx, query, url, scrapedAt, contentHash)
	if err != nil {
		return fmt.Errorf("failed to save scraped data for url %s: %w", url, err)
	}
	return nil
}

// GetLastScrapeTime retrieves the last time a URL was scraped.
// Returns sql.ErrNoRows if the URL has not been scraped.
func (s *SQLiteStorage) GetLastScrapeTime(ctx context.Context, url string) (time.Time, error) {
	query := `SELECT scraped_at FROM scraped_pages WHERE url = ?`
	var scrapedAt time.Time
	err := s.db.QueryRowContext(ctx, query, url).Scan(&scrapedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, err // Return zero time and ErrNoRows specifically
		}
		return time.Time{}, fmt.Errorf("failed to get last scrape time for url %s: %w", url, err)
	}
	return scrapedAt, nil
}

// GetScrapedPages retrieves a list of scraped pages
func (s *SQLiteStorage) GetScrapedPages(ctx context.Context, limit int) ([]Page, error) {
	// Create query with limit
	query := `SELECT url, scraped_at, content_hash FROM scraped_pages ORDER BY scraped_at DESC LIMIT ?`
	
	// Execute query
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query scraped pages: %w", err)
	}
	defer rows.Close()
	
	// Parse rows into Page structs
	var pages []Page
	for rows.Next() {
		var page Page
		if err := rows.Scan(&page.URL, &page.ScrapedAt, &page.ContentHash); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		pages = append(pages, page)
	}
	
	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}
	
	return pages, nil
}

// GetScrapedPagesCount returns the total count of scraped pages
func (s *SQLiteStorage) GetScrapedPagesCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM scraped_pages`
	
	var count int
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count scraped pages: %w", err)
	}
	
	return count, nil
}

// GetScrapedPagesPaginated retrieves a paginated list of scraped pages
func (s *SQLiteStorage) GetScrapedPagesPaginated(ctx context.Context, limit int, offset int) ([]Page, error) {
	// Create query with limit and offset
	query := `SELECT url, scraped_at, content_hash FROM scraped_pages ORDER BY scraped_at DESC LIMIT ? OFFSET ?`
	
	// Execute query
	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query scraped pages with pagination: %w", err)
	}
	defer rows.Close()
	
	// Parse rows into Page structs
	var pages []Page
	for rows.Next() {
		var page Page
		if err := rows.Scan(&page.URL, &page.ScrapedAt, &page.ContentHash); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		pages = append(pages, page)
	}
	
	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}
	
	return pages, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
