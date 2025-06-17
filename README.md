# mta-go

A Go implementation of an MTA real-time subway data API, inspired by [MTAPI](https://github.com/jonthornton/MTAPI).

## Features

- Real-time subway train arrival data
- Station queries by location, route, or ID
- Service alerts
- Both HTTP server and local library modes
- Clean, idiomatic Go implementation
- Automatic feed updates with configurable intervals

## Installation

```bash
go get github.com/jusunglee/mta-go
```

## Usage

### Get an MTA API Key

Sign up for an API key at https://api.mta.info/

### Server Mode

Run as an HTTP server:

```bash
# Using environment variable
export MTA_API_KEY=your-api-key
go run cmd/server/main.go

# Or using flag
go run cmd/server/main.go -api-key=your-api-key -port=8080
```

### Local Mode

Use as a library in your Go code:

```go
package main

import (
    "fmt"
    "log"
    "github.com/jusunglee/mta-go/pkg/mta"
)

func main() {
    config := mta.DefaultConfig()
    config.APIKey = "your-api-key"
    
    client, err := mta.NewLocal(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Get nearest stations
    stations, err := client.GetStationsByLocation(40.7527, -73.9772, 5)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, station := range stations {
        fmt.Printf("%s: %v\n", station.Name, station.Routes)
    }
}
```

## API Endpoints

When running in server mode:

- `GET /` - API information
- `GET /by-location?lat={latitude}&lon={longitude}` - Get 5 nearest stations
- `GET /by-route/{route}` - Get all stations on a route
- `GET /by-id/{id1},{id2},...` - Get stations by IDs
- `GET /routes` - List all available routes
- `GET /alerts` - Get service alerts

## Building

```bash
# Build both server and local binaries
make build

# Run tests
make test

# Run server
make run-server MTA_API_KEY=your-api-key

# Run local example
make run-local MTA_API_KEY=your-api-key
```

## Docker Deployment

### Quick Start

```bash
# Copy environment template
cp .env.example .env
# Edit .env and add your MTA_API_KEY

# Build and run
docker-compose up --build
```

### Manual Docker

```bash
# Build image
docker build -t mta-go .

# Run container
docker run -p 8080:8080 -e MTA_API_KEY=your_key_here mta-go
```

## Configuration

The following environment variables are supported:

- `MTA_API_KEY` - Your MTA API key (required)
- `UPDATE_INTERVAL` - Feed update interval (default: 60s)
- `PORT` - Server port (default: 8080)

## Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────────┐
│                        MTA Go Architecture                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐        │
│  │     API     │    │    Feed     │    │    Store    │        │
│  │  Handlers   │────│   Manager   │────│  (Memory)   │        │
│  └─────────────┘    └─────────────┘    └─────────────┘        │
│         │                   │                   │               │
│         │                   │                   │               │
│    HTTP Requests       MTA APIs            In-Memory DB         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Core Components

**1. Feed Manager (`internal/feed/`)**
- **Purpose**: Data acquisition and ETL (Extract, Transform, Load)
- **Responsibilities**:
  - Downloads GTFS static data (stations, routes) from MTA S3 buckets
  - Fetches real-time GTFS-RT protobuf feeds every 60 seconds
  - Parses CSV files and protobuf messages
  - Transforms data into internal models
  - Updates the Store with fresh data

**2. Store (`internal/store/`)**
- **Purpose**: High-performance in-memory database
- **Responsibilities**:
  - Thread-safe data storage using `sync.RWMutex`
  - Pre-built indices for fast queries (by location, route, ID)
  - Concurrent read access for API requests
  - Atomic updates from Feed Manager

**3. Data Flow**

```
External Sources → Feed Manager → Store → API Handlers → JSON Response
       ↓              ↓          ↓          ↓
1. MTA GTFS-RT     Parse &    Cache &    Query &      HTTP
2. MTA GTFS ZIP    Transform  Index     Filter       Response
```

### Feed Manager Details

**Static Data Loading (Every 6 Hours)**
```go
// Downloads ZIP files from MTA S3
loadStaticGTFSData() {
    download("gtfs_supplemented.zip")  // Preferred: includes service changes
    extract() → parseGTFSData() {
        parseStops()     // Station locations
        parseRoutes()    // Route-to-station mapping
        parseTrips()     // Trip definitions
        parseStopTimes() // Stop sequences
    }
    store.UpdateStations()
}
```

**Real-Time Data Updates (Every 60 Seconds)**
```go
updateRealTimeData() {
    for each feedURL in FeedURLs {
        data := fetchFeed(feedURL)           // GTFS-RT protobuf
        feedMessage := unmarshal(data)
        
        for entity in feedMessage.Entity {
            processTripUpdate()              // Train arrivals
            processAlert()                   // Service alerts
        }
    }
    
    sortAndLimitTrains()                     // Clean up data
    store.UpdateStations()                   // Atomic update
}
```

### Store Architecture

**Data Structures**
```go
type Store struct {
    mu              sync.RWMutex                    // Concurrent access control
    stations        map[string]*Station             // O(1) ID lookup
    stationsByRoute map[string][]*Station           // Pre-indexed route queries
    routes          []string                        // Sorted route list
    alerts          []Alert                         // Service alerts
}
```

**Query Optimization**
- **By Location**: Haversine distance calculation with proximity sorting
- **By Route**: Pre-built index for instant route-based filtering
- **By ID**: Direct hash map lookup for batch queries
- **Concurrency**: Multiple readers, single writer pattern

**Why RWMutex?**
```go
// High-frequency reads (API requests)
store.RLock()                    // Shared lock - multiple concurrent readers
stations := store.GetStations()
store.RUnlock()

// Low-frequency writes (feed updates)
store.Lock()                     // Exclusive lock - blocks all access
store.UpdateStations(newData)
store.Unlock()
```

### Data Update Cycle

**Producer-Consumer Pattern**
1. **Feed Manager** (Producer): Runs background goroutine with ticker
2. **Store** (Consumer): Receives atomic updates
3. **API Handlers** (Readers): Query current data

**Update Frequencies**
- **Static GTFS**: 6 hours (station locations, routes, schedules)
- **Real-time GTFS-RT**: 60 seconds (train arrivals, service alerts)
- **API Responses**: On-demand (serves cached data)

**Error Handling**
- **Static data failure**: App fails to start (critical)
- **Real-time failure**: Continues with last known data (graceful degradation)
- **Individual feed failure**: Processes other feeds (fault tolerance)

**Memory Management**
- **Train arrivals**: Limited to next 10 per direction
- **Old arrivals**: Filtered out (>1 minute past)
- **Duplicate trains**: Deduplication by route + time
- **Station copies**: Prevents data races during updates

### Threading Model

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Goroutine 1   │    │   Goroutine 2   │    │   Goroutine N   │
│  Feed Manager   │    │  HTTP Handler   │    │  HTTP Handler   │
│   (Writer)      │    │   (Reader)      │    │   (Reader)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │ store.Lock()          │ store.RLock()         │ store.RLock()
         ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Store (RWMutex)                          │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │              Thread-Safe Data Structures                    │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Development

### Project Structure

```
mta-go/
├── cmd/
│   ├── server/          # HTTP server
│   └── local/           # CLI example
├── internal/
│   ├── feed/            # Feed fetching and ETL
│   ├── store/           # In-memory database
│   ├── models/          # Domain models
│   └── gtfsrt/          # GTFS-RT protobuf definitions
├── pkg/
│   └── mta/             # Public API
├── api/
│   └── handlers/        # HTTP handlers
└── .github/
    └── workflows/       # CI/CD pipeline
```

### Git Hooks

The repository includes pre-commit hooks for code formatting:

```bash
# The hook is already installed and will run automatically
# It runs gofmt -s -w on all staged Go files
git commit -m "your changes"
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test ./internal/feed -v
```

### Large File Handling

Test data uses Git LFS for files >100MB:

```bash
# After cloning, download LFS files
git lfs pull
```

## License

MIT

## Acknowledgments

Inspired by [MTAPI](https://github.com/jonthornton/MTAPI) by Jon Thornton.# mta-go
