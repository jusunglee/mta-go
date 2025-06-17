package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jusunglee/mta-go/pkg/mta"
)

func main() {
	var (
		apiKey = flag.String("api-key", "", "MTA API key")
		lat    = flag.Float64("lat", 40.7527, "Latitude")
		lon    = flag.Float64("lon", -73.9772, "Longitude")
		route  = flag.String("route", "", "Route to query")
	)
	flag.Parse()

	// Fallback to environment variable if API key not provided via flag
	if *apiKey == "" {
		*apiKey = os.Getenv("MTA_API_KEY")
	}
	if *apiKey == "" {
		log.Fatal("MTA API key required (use -api-key flag or MTA_API_KEY env var)")
	}

	config := mta.DefaultConfig()
	config.APIKey = *apiKey

	client, err := mta.NewLocal(config)
	if err != nil {
		log.Fatalf("Failed to create MTA client: %v", err)
	}
	defer client.Close()

	// Allow feed manager time to populate initial data
	fmt.Println("Waiting for initial data...")
	time.Sleep(2 * time.Second)

	// Route-specific query mode
	if *route != "" {
		stations, err := client.GetStationsByRoute(*route)
		if err != nil {
			log.Fatalf("Failed to get stations for route %s: %v", *route, err)
		}

		fmt.Printf("\nStations on route %s:\n", *route)
		for _, station := range stations {
			fmt.Printf("- %s (%s)\n", station.Name, station.ID)
		}
		return
	}

	// Default location-based query mode
	stations, err := client.GetStationsByLocation(*lat, *lon, 5)
	if err != nil {
		log.Fatalf("Failed to get stations: %v", err)
	}

	fmt.Printf("\nNearest stations to (%.4f, %.4f):\n", *lat, *lon)
	for _, station := range stations {
		fmt.Printf("\n%s (%s)\n", station.Name, station.ID)
		fmt.Printf("  Routes: %v\n", station.Routes)

		if len(station.Trains.North) > 0 {
			fmt.Println("  Northbound:")
			for _, train := range station.Trains.North[:min(3, len(station.Trains.North))] {
				fmt.Printf("    %s - %s\n", train.Route, train.Time.Format("3:04 PM"))
			}
		}

		if len(station.Trains.South) > 0 {
			fmt.Println("  Southbound:")
			for _, train := range station.Trains.South[:min(3, len(station.Trains.South))] {
				fmt.Printf("    %s - %s\n", train.Route, train.Time.Format("3:04 PM"))
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
