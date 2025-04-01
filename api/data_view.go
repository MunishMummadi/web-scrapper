package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MunishMummadi/web-scrapper/database"
)

// PageData represents data for a scraped page
type PageData struct {
	URL         string    `json:"url"`
	ScrapedAt   time.Time `json:"scraped_at"`
	ContentHash string    `json:"content_hash"`
}

// StatsData represents stats for the dashboard
type StatsData struct {
	TotalUrls  int    `json:"total_urls"`
	QueuedUrls int    `json:"queued_urls"`
	CrawlRate  int    `json:"crawl_rate"`
	ErrorRate  string `json:"error_rate"`
}

// DataViewHandler handles requests to view scraped data
type DataViewHandler struct {
	storage database.Storage
}

// NewDataViewHandler creates a new handler for viewing data
func NewDataViewHandler(storage database.Storage) *DataViewHandler {
	return &DataViewHandler{
		storage: storage,
	}
}

// RegisterRoutes registers the data view routes
func (h *DataViewHandler) RegisterRoutes(mux *http.ServeMux) {
	// Simple routes
	mux.HandleFunc("/", h.handleDashboard)
	
	// API routes
	mux.HandleFunc("/api/data", h.handleAPIData)
	mux.HandleFunc("/api/stats", h.handleAPIStats)
	mux.HandleFunc("/api/jobs", h.handleAPIJobs)
	mux.HandleFunc("/api/settings", h.handleAPISettings)
}

// handleDashboard renders a simple dashboard view
func (h *DataViewHandler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Check if this is the root path, otherwise 404
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	// Get recent scraped pages from storage (limit to 10)
	pages, err := h.storage.GetScrapedPages(ctx, 10)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get scraped pages: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Get total count of pages
	totalCount, err := h.storage.GetScrapedPagesCount(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get page count: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Generate a simple HTML response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Web Scraper Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
        h1 { color: #333; }
        table { border-collapse: collapse; width: 100%%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        tr:nth-child(even) { background-color: #f9f9f9; }
        .stats { margin-bottom: 20px; }
    </style>
</head>
<body>
    <h1>Web Scraper Dashboard</h1>
    
    <div class="stats">
        <h2>Stats</h2>
        <p>Total URLs: %d</p>
    </div>
    
    <div>
        <h2>Recent Scraped Pages</h2>
        <table>
            <tr>
                <th>URL</th>
                <th>Scraped At</th>
                <th>Content Hash</th>
            </tr>
`, totalCount)
	
	// Add table rows for each page
	for _, page := range pages {
		fmt.Fprintf(w, `
            <tr>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
            </tr>
`, page.URL, page.ScrapedAt.Format(time.RFC3339), page.ContentHash)
	}
	
	// Close the HTML
	fmt.Fprintf(w, `
        </table>
    </div>
    
    <div>
        <h2>Add URL to Scrape</h2>
        <form action="/api/enqueue" method="post">
            <input type="text" name="url" placeholder="Enter URL to scrape" style="width: 300px;">
            <button type="submit">Enqueue</button>
        </form>
    </div>
</body>
</html>`)
}

// handleAPIData returns scraped data as JSON
func (h *DataViewHandler) handleAPIData(w http.ResponseWriter, r *http.Request) {
	// Get pagination parameters
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 20 // Default to 20 per page
	}
	
	offset := (page - 1) * limit
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	// Get total count of pages for pagination
	totalCount, err := h.storage.GetScrapedPagesCount(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get page count: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Get pages for current page
	pages, err := h.storage.GetScrapedPagesPaginated(ctx, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get scraped pages: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Convert to our view model
	pageData := make([]PageData, 0, len(pages))
	for _, page := range pages {
		pageData = append(pageData, PageData{
			URL:         page.URL,
			ScrapedAt:   page.ScrapedAt,
			ContentHash: page.ContentHash,
		})
	}
	
	// Create response with pagination info
	response := struct {
		TotalCount  int       `json:"total_count"`
		TotalPages  int       `json:"total_pages"`
		CurrentPage int       `json:"current_page"`
		Limit       int       `json:"limit"`
		Offset      int       `json:"offset"`
		Data        []PageData `json:"data"`
	}{
		TotalCount:  totalCount,
		TotalPages:  (totalCount + limit - 1) / limit,
		CurrentPage: page,
		Limit:       limit,
		Offset:      offset,
		Data:        pageData,
	}
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// handleAPIStats returns stats as JSON for the dashboard
func (h *DataViewHandler) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	// Get total count of pages
	totalCount, err := h.storage.GetScrapedPagesCount(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get page count: %v", err), http.StatusInternalServerError)
		return
	}
	
	// In a real implementation, these would come from metrics or other sources
	stats := StatsData{
		TotalUrls:  totalCount,
		QueuedUrls: 0, // Would come from queue in a real implementation
		CrawlRate:  10, // Would be calculated in a real implementation
		ErrorRate:  "5%", // Would be calculated in a real implementation
	}
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// handleAPIJobs returns active jobs as JSON
func (h *DataViewHandler) handleAPIJobs(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would get active jobs from a job manager
	jobs := []struct {
		ID     string `json:"id"`
		URL    string `json:"url"`
		Status string `json:"status"`
	}{}
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// handleAPISettings returns or updates settings
func (h *DataViewHandler) handleAPISettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Return current settings (mock data for now)
		settings := struct {
			MaxConcurrency int    `json:"max_concurrency"`
			UserAgent      string `json:"user_agent"`
			RespectRobots  bool   `json:"respect_robots"`
		}{
			MaxConcurrency: 5,
			UserAgent:      "WebScraper/1.0",
			RespectRobots:  true,
		}
		
		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(settings); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}
	} else if r.Method == http.MethodPost {
		// Parse settings from request body
		var settings struct {
			MaxConcurrency int    `json:"max_concurrency"`
			UserAgent      string `json:"user_agent"`
			RespectRobots  bool   `json:"respect_robots"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, fmt.Sprintf("Failed to decode request: %v", err), http.StatusBadRequest)
			return
		}
		
		// Validate settings
		if settings.MaxConcurrency < 1 {
			http.Error(w, "Max concurrency must be at least 1", http.StatusBadRequest)
			return
		}
		
		if settings.UserAgent == "" {
			http.Error(w, "User agent cannot be empty", http.StatusBadRequest)
			return
		}
		
		// In a real implementation, these would be saved to a settings store
		
		// Return success response
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Settings updated successfully")
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
