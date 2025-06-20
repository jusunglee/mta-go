package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jusunglee/mta-go/api/handlers"
	"github.com/jusunglee/mta-go/pkg/mta"
)

func main() {
	var (
		port           = flag.String("port", "8080", "Server port")
		apiKey         = flag.String("api-key", "", "MTA API key")
		updateInterval = flag.Duration("update-interval", 60*time.Second, "Feed update interval")
		stationsFile   = flag.String("stations-file", "data/stations.json", "Stations JSON file")
	)
	flag.Parse()

	// Fallback to environment variable if API key not provided via flag
	if *apiKey == "" {
		*apiKey = os.Getenv("MTA_API_KEY")
	}
	if *apiKey == "" {
		slog.Error("MTA API key required (use -api-key flag or MTA_API_KEY env var)")
		os.Exit(1)
	}

	config := mta.Config{
		APIKey:         *apiKey,
		UpdateInterval: *updateInterval,
		StationsFile:   *stationsFile,
	}

	client, err := mta.NewLocal(config)
	if err != nil {
		slog.Error("Failed to create MTA client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// Allow time for feed manager to fetch initial station data
	slog.Info("Waiting for initial data...")
	time.Sleep(2 * time.Second)

	r := mux.NewRouter()
	h := handlers.NewHandler(client)
	h.RegisterRoutes(r)

	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in goroutine for graceful shutdown
	go func() {
		slog.Info("Server starting", "port", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Block until interrupt signal received
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Allow 30 seconds for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}

// loggingMiddleware logs HTTP requests with method, URI, and response time
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("HTTP request", "method", r.Method, "uri", r.RequestURI, "duration", time.Since(start))
	})
}

// corsMiddleware enables CORS for web browser access
// Allows all origins since this is a public transit API
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
