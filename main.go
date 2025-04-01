package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MunishMummadi/web-scrapper/api"
	"github.com/MunishMummadi/web-scrapper/config"
	"github.com/MunishMummadi/web-scrapper/crawler"
	"github.com/MunishMummadi/web-scrapper/database"
	"github.com/MunishMummadi/web-scrapper/metrics"
	"github.com/MunishMummadi/web-scrapper/proxy"
	"github.com/MunishMummadi/web-scrapper/queue"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	configFile  string
	seedURL     string
	useMemQueue bool
)

func init() {
	flag.StringVar(&configFile, "config", "", "Path to configuration file")
	flag.StringVar(&seedURL, "seed", "", "Seed URL to start crawling")
	flag.BoolVar(&useMemQueue, "mem-queue", false, "Use in-memory queue instead of Redis (useful for testing)")
}

func main() {
	flag.Parse()

	// Load configuration
	log.Println("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create context that can be canceled on shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize metrics collector
	log.Println("Initializing metrics collector...")
	metricsCollector := metrics.NewMetricsCollector()

	// Initialize queue (Redis-based or in-memory)
	var q queue.Queue
	if useMemQueue {
		log.Println("Using in-memory queue (as requested)...")
		q = queue.NewMemoryQueue()
	} else {
		log.Println("Initializing Redis queue...")
		redisQueue, err := queue.NewRedisQueue(cfg.Redis)
		if err != nil {
			log.Printf("Failed to initialize Redis queue: %v", err)
			log.Println("Falling back to in-memory queue...")
			q = queue.NewMemoryQueue()
		} else {
			q = redisQueue
		}
	}
	defer q.Close()

	// Initialize SQLite storage
	log.Println("Initializing SQLite storage...")
	sqliteStorage, err := database.NewSQLiteStorage(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize SQLite storage: %v", err)
	}
	defer sqliteStorage.Close()

	// Initialize proxy manager
	log.Println("Initializing proxy manager...")
	proxyManager, err := proxy.NewManager(cfg.Proxies)
	if err != nil {
		log.Fatalf("Failed to initialize proxy manager: %v", err)
	}
	defer proxyManager.Close()

	// Initialize crawler
	log.Println("Initializing crawler...")
	c, err := crawler.NewCrawler(cfg, q, sqliteStorage, metricsCollector, proxyManager)
	if err != nil {
		log.Fatalf("Failed to initialize crawler: %v", err)
	}

	// Start crawler
	log.Println("Starting crawler...")
	c.Start(ctx)
	defer c.Stop()

	// If seed URL is provided, enqueue it
	if seedURL != "" {
		log.Printf("Enqueuing seed URL: %s", seedURL)
		if err := c.EnqueueURL(ctx, seedURL); err != nil {
			log.Printf("Failed to enqueue seed URL: %v", err)
		}
	}

	// Set up HTTP server for API and metrics
	apiServer := setupAPIServer(cfg, c, metricsCollector, sqliteStorage)

	// Start HTTP server in a goroutine
	go func() {
		serverAddr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
		log.Printf("Starting API server on %s...", serverAddr)
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown signal received, stopping services...")

	// Graceful shutdown of API server
	apiShutdownCtx, apiShutdownCancel := context.WithTimeout(context.Background(), cfg.API.ShutdownTimeout)
	defer apiShutdownCancel()
	if err := apiServer.Shutdown(apiShutdownCtx); err != nil {
		log.Printf("API server shutdown failed: %v", err)
	}

	// Cancel context to signal all workers to stop
	cancel()
	log.Println("All services stopped, exiting")
}

func setupAPIServer(cfg *config.Config, c *crawler.Crawler, m *metrics.MetricsCollector, storage database.Storage) *http.Server {
	mux := http.NewServeMux()

	// Serve static files for the web UI
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// API endpoint for submitting URLs
	mux.HandleFunc("/api/enqueue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		urlToScrape := r.FormValue("url")
		if urlToScrape == "" {
			http.Error(w, "URL parameter is required", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Enqueue the URL for crawling
		err := c.EnqueueURL(ctx, urlToScrape)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to enqueue URL: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "URL %s has been queued for crawling\n", urlToScrape)
	})

	// API endpoint for health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	// Data view handler for viewing scraped pages
	dataViewHandler := api.NewDataViewHandler(storage)
	dataViewHandler.RegisterRoutes(mux)
	
	// Home page redirects to data view
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/view/data", http.StatusFound)
	})

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
		Handler:      mux,
		ReadTimeout:  cfg.API.ReadTimeout,
		WriteTimeout: cfg.API.WriteTimeout,
	}
}