package main

import (
	"context"
	"flag"
	"log"
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

	// Check for API key in environment if not provided
	if *apiKey == "" {
		*apiKey = os.Getenv("MTA_API_KEY")
	}
	if *apiKey == "" {
		log.Fatal("MTA API key required (use -api-key flag or MTA_API_KEY env var)")
	}

	// Create local client
	config := mta.Config{
		APIKey:         *apiKey,
		UpdateInterval: *updateInterval,
		StationsFile:   *stationsFile,
	}

	client, err := mta.NewLocal(config)
	if err != nil {
		log.Fatalf("Failed to create MTA client: %v", err)
	}
	defer client.Close()

	// Wait a moment for initial data
	log.Println("Waiting for initial data...")
	time.Sleep(2 * time.Second)

	// Create HTTP server
	r := mux.NewRouter()
	h := handlers.NewHandler(client)
	h.RegisterRoutes(r)

	// Add middleware
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("Server starting on port %s", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
