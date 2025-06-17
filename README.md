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

## Configuration

The following environment variables are supported:

- `MTA_API_KEY` - Your MTA API key (required)
- `UPDATE_INTERVAL` - Feed update interval (default: 60s)
- `PORT` - Server port (default: 8080)

## Development

### Project Structure

```
mta-go/
├── cmd/
│   ├── server/          # HTTP server
│   └── local/           # CLI example
├── internal/
│   ├── feed/            # Feed fetching
│   ├── store/           # Data storage
│   ├── models/          # Domain models
│   └── proto/           # Protobuf definitions
├── pkg/
│   └── mta/             # Public API
└── api/
    └── handlers/        # HTTP handlers
```

### Example API Calls
Check [API_EXAMPLES.md](api/handlers/API_EXAMPLES.md) for examples of how to call the API.

### Running Tests

```bash
go test ./...
```

## License

MIT

## Acknowledgments

Inspired by [MTAPI](https://github.com/jonthornton/MTAPI) by Jon Thornton.# mta-go
