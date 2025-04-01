package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
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

// TemplateData represents the common data structure for all templates
type TemplateData struct {
	Title      string
	ActivePage string
	Stats      *StatsData
	Pages      []PageData
	// Pagination data
	CurrentPage int
	TotalPages  int
	PageNumbers []int
	Limit       int
	// For dashboard
	RecentPages []PageData
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
	tmpl    *template.Template
}

// NewDataViewHandler creates a new handler for viewing data
func NewDataViewHandler(storage database.Storage) *DataViewHandler {
	// Load all templates
	tmpl := template.Must(template.ParseGlob(filepath.Join("api", "templates", "*.html")))
	
	return &DataViewHandler{
		storage: storage,
		tmpl:    tmpl,
	}
}

// RegisterRoutes registers the data view routes
func (h *DataViewHandler) RegisterRoutes(mux *http.ServeMux) {
	// Template routes
	mux.HandleFunc("/", h.handleDashboard)
	mux.HandleFunc("/view/data", h.handleDataView)
	mux.HandleFunc("/scrape", h.handleScrapeView)
	mux.HandleFunc("/settings", h.handleSettingsView)
	
	// API routes
	mux.HandleFunc("/api/data", h.handleAPIData)
	mux.HandleFunc("/api/stats", h.handleAPIStats)
	mux.HandleFunc("/api/jobs", h.handleAPIJobs)
	mux.HandleFunc("/api/settings", h.handleAPISettings)
}

// handleDashboard renders the dashboard view
func (h *DataViewHandler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Check if this is the root path, otherwise 404
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	// Get recent scraped pages from storage (limit to 5)
	pages, err := h.storage.GetScrapedPages(ctx, 5)
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
	
	// Get mock stats data - in a real implementation, these would come from storage
	stats := &StatsData{
		TotalUrls:  len(pages),
		QueuedUrls: 0, // Would come from queue in a real implementation
		CrawlRate:  10, // Would be calculated in a real implementation
		ErrorRate:  "5",  // Would be calculated in a real implementation
	}
	
	data := TemplateData{
		Title:       "Dashboard",
		ActivePage:  "dashboard",
		Stats:       stats,
		RecentPages: pageData,
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, fmt.Sprintf("Template rendering error: %v", err), http.StatusInternalServerError)
	}
}

// handleDataView renders the data view page
func (h *DataViewHandler) handleDataView(w http.ResponseWriter, r *http.Request) {
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
	
	// Calculate total pages
	totalPages := (totalCount + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
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
	
	// Generate page numbers for pagination UI
	pageNumbers := make([]int, 0)
	if totalPages <= 7 {
		// Less than 7 pages, show them all
		for i := 1; i <= totalPages; i++ {
			pageNumbers = append(pageNumbers, i)
		}
	} else {
		// More than 7 pages, show current page and neighbors
		if page <= 4 {
			// Near the beginning
			for i := 1; i <= 5; i++ {
				pageNumbers = append(pageNumbers, i)
			}
			pageNumbers = append(pageNumbers, -1) // Placeholder for ellipsis
			pageNumbers = append(pageNumbers, totalPages)
		} else if page >= totalPages-3 {
			// Near the end
			pageNumbers = append(pageNumbers, 1)
			pageNumbers = append(pageNumbers, -1) // Placeholder for ellipsis
			for i := totalPages - 4; i <= totalPages; i++ {
				pageNumbers = append(pageNumbers, i)
			}
		} else {
			// Middle
			pageNumbers = append(pageNumbers, 1)
			pageNumbers = append(pageNumbers, -1) // Placeholder for ellipsis
			for i := page - 1; i <= page + 1; i++ {
				pageNumbers = append(pageNumbers, i)
			}
			pageNumbers = append(pageNumbers, -1) // Placeholder for ellipsis
			pageNumbers = append(pageNumbers, totalPages)
		}
	}
	
	// Get mock stats data
	stats := &StatsData{
		TotalUrls:  totalCount,
		QueuedUrls: 0,
		CrawlRate:  10,
		ErrorRate:  "5",
	}
	
	data := TemplateData{
		Title:       "Scraped Data",
		ActivePage:  "data",
		Stats:       stats,
		Pages:       pageData,
		CurrentPage: page,
		TotalPages:  totalPages,
		PageNumbers: pageNumbers,
		Limit:       limit,
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "data_view.html", data); err != nil {
		http.Error(w, fmt.Sprintf("Template rendering error: %v", err), http.StatusInternalServerError)
	}
}

// handleScrapeView renders the scrape form page
func (h *DataViewHandler) handleScrapeView(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Title:      "Scrape URLs",
		ActivePage: "scrape",
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "scrape.html", data); err != nil {
		http.Error(w, fmt.Sprintf("Template rendering error: %v", err), http.StatusInternalServerError)
	}
}

// handleSettingsView renders the settings page
func (h *DataViewHandler) handleSettingsView(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Title:      "Settings",
		ActivePage: "settings",
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "settings.html", data); err != nil {
		http.Error(w, fmt.Sprintf("Template rendering error: %v", err), http.StatusInternalServerError)
	}
}

// handleAPIData returns scraped data as JSON
func (h *DataViewHandler) handleAPIData(w http.ResponseWriter, r *http.Request) {
	// Get limit parameter, default to 100
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	// Get page parameter, default to 1
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}
	
	offset := (page - 1) * limit

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get scraped pages from storage
	pages, err := h.storage.GetScrapedPagesPaginated(ctx, limit, offset)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to get scraped pages: %v", err),
		})
		return
	}

	// Convert to our API format
	pageData := make([]PageData, 0, len(pages))
	for _, page := range pages {
		pageData = append(pageData, PageData{
			URL:         page.URL,
			ScrapedAt:   page.ScrapedAt,
			ContentHash: page.ContentHash,
		})
	}

	// Return as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pageData)
}

// handleAPIStats returns stats as JSON for the dashboard
func (h *DataViewHandler) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	// Get total count for stats
	totalCount, err := h.storage.GetScrapedPagesCount(ctx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to get page count: %v", err),
		})
		return
	}
	
	// In a real implementation, we would query the queue and metrics
	// For now, we'll return mock data
	stats := &StatsData{
		TotalUrls:  totalCount,
		QueuedUrls: 0,
		CrawlRate:  10,
		ErrorRate:  "5",
	}
	
	// Return as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// handleAPIJobs returns active jobs as JSON
func (h *DataViewHandler) handleAPIJobs(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, we would query the job store
	// For now, we'll return an empty array
	jobs := []interface{}{} // No active jobs for now
	
	// Return as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs": jobs,
	})
}

// handleAPISettings returns or updates settings
func (h *DataViewHandler) handleAPISettings(w http.ResponseWriter, r *http.Request) {
	// GET returns settings, POST updates settings
	if r.Method == http.MethodGet {
		// In a real implementation, we would load from a settings store
		// For now, we'll return default values
		settings := map[string]interface{}{
			"queue": map[string]interface{}{
				"timeout":          1,
				"workerTimeout":    1,
				"emptyQueueBackoff": 500,
			},
			"circuitBreaker": map[string]interface{}{
				"errorThreshold": 5,
				"resetTimeout":   15,
			},
			"rateLimit": map[string]interface{}{
				"globalRate":    120,
				"perDomainRate": 30,
			},
			"proxies": map[string]interface{}{
				"enabled": false,
				"list":    []string{},
				"testUrl": "https://httpbin.org/ip",
			},
			"userAgents": map[string]interface{}{
				"rotate": true,
				"list": []string{
					"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
					"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
				},
			},
		}
		
		// Return as JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(settings)
	} else {
		// In a real implementation, we would validate and save settings
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprintln(w, "Settings update not implemented yet")
	}
}
