# MTA API Examples

## Testing the API Endpoints

Start the server first:
```bash
cd ~/code/mta-api
export MTA_API_KEY=your-api-key
./mta-server -port=8080
```

## 1. API Information

```bash
# Get API info
curl http://localhost:8080/

# Pretty print with jq
curl -s http://localhost:8080/ | jq .
```

Expected response:
```json
{
  "title": "MTA-API",
  "readme": "Visit https://github.com/user/mta-api for more info"
}
```

## 2. Get Stations by Location

```bash
# Times Square area
curl "http://localhost:8080/by-location?lat=40.7580&lon=-73.9855"

# Grand Central area
curl "http://localhost:8080/by-location?lat=40.7527&lon=-73.9772"

# Union Square area
curl "http://localhost:8080/by-location?lat=40.7359&lon=-73.9911"

# Pretty print with jq
curl -s "http://localhost:8080/by-location?lat=40.7580&lon=-73.9855" | jq .
```

Expected response format:
```json
{
  "data": [
    {
      "id": "127",
      "name": "Times Sq-42 St",
      "location": [40.755477, -73.987691],
      "routes": ["N", "Q", "R", "W", "S", "1", "2", "3", "7"],
      "N": [
        {
          "route": "N",
          "time": "2024-01-15T14:32:00Z"
        }
      ],
      "S": [
        {
          "route": "R",
          "time": "2024-01-15T14:31:00Z"
        }
      ],
      "stops": {
        "127N": [40.755983, -73.986229],
        "127S": [40.75529, -73.987495]
      },
      "last_update": "2024-01-15T14:30:00Z"
    }
  ],
  "updated": "2024-01-15T14:30:00Z"
}
```

## 3. Get Stations by Route

```bash
# Get all stations on the N train
curl http://localhost:8080/by-route/N

# Get all stations on the 6 train
curl http://localhost:8080/by-route/6

# Get all stations on the L train
curl http://localhost:8080/by-route/L

# Case insensitive - these work too
curl http://localhost:8080/by-route/n
curl http://localhost:8080/by-route/q

# Pretty print
curl -s http://localhost:8080/by-route/N | jq '.data[].name'
```

## 4. Get Stations by IDs

```bash
# Get single station
curl http://localhost:8080/by-id/127

# Get multiple stations (Times Square, Grand Central, Union Square)
curl http://localhost:8080/by-id/127,631,635

# Pretty print station names
curl -s http://localhost:8080/by-id/127,631,635 | jq '.data[].name'
```

## 5. Get All Routes

```bash
# Get list of all routes
curl http://localhost:8080/routes

# Pretty print
curl -s http://localhost:8080/routes | jq .
```

Expected response:
```json
{
  "data": ["1", "2", "3", "4", "5", "6", "7", "L", "N", "Q", "R", "S", "W"],
  "updated": "2024-01-15T14:30:00Z"
}
```

## 6. Get Service Alerts

```bash
# Get current service alerts
curl http://localhost:8080/alerts

# Pretty print
curl -s http://localhost:8080/alerts | jq .
```

Expected response:
```json
{
  "data": [
    {
      "id": "alert1",
      "header": "Weekend Service Change",
      "description": "N/Q/R/W trains are running on a modified schedule this weekend",
      "routes": ["N", "Q", "R", "W"],
      "stations": ["127", "635"],
      "active_periods": [
        {
          "start": "2024-01-15T14:30:00Z",
          "end": "2024-01-15T16:30:00Z"
        }
      ]
    }
  ],
  "updated": "2024-01-15T14:30:00Z"
}
```

## Error Cases

```bash
# Missing parameters
curl "http://localhost:8080/by-location"
# Returns: {"error":"Missing lat/lon parameter"}

# Invalid parameters
curl "http://localhost:8080/by-location?lat=abc&lon=xyz"
# Returns: {"error":"Invalid lat parameter"}

# Non-existent route
curl http://localhost:8080/by-route/Z
# Returns: {"error":"route Z not found"}

# Non-existent station IDs
curl http://localhost:8080/by-id/999,000
# Returns: {"error":"no stations found for given IDs"}
```

## Testing with HTTPie

If you have [HTTPie](https://httpie.io/) installed:

```bash
# Get stations near Times Square
http GET localhost:8080/by-location lat==40.7580 lon==-73.9855

# Get N train stations
http GET localhost:8080/by-route/N

# Get specific stations
http GET localhost:8080/by-id/127,631

# Get routes
http GET localhost:8080/routes

# Get alerts
http GET localhost:8080/alerts
```

## Testing with a Script

Create a test script `test_api.sh`:

```bash
#!/bin/bash

BASE_URL="http://localhost:8080"

echo "=== Testing MTA API ==="
echo

echo "1. API Info:"
curl -s $BASE_URL | jq .
echo

echo "2. Stations near Times Square:"
curl -s "$BASE_URL/by-location?lat=40.7580&lon=-73.9855" | jq '.data[].name'
echo

echo "3. Stations on N route:"
curl -s "$BASE_URL/by-route/N" | jq '.data[].name'
echo

echo "4. Specific stations:"
curl -s "$BASE_URL/by-id/127,631" | jq '.data[] | {name, routes}'
echo

echo "5. All routes:"
curl -s "$BASE_URL/routes" | jq .data
echo

echo "6. Service alerts:"
curl -s "$BASE_URL/alerts" | jq '.data[].header'
echo
```

Make it executable and run:
```bash
chmod +x test_api.sh
./test_api.sh
```

## Load Testing with Apache Bench

```bash
# Test location endpoint (100 requests, 10 concurrent)
ab -n 100 -c 10 "http://localhost:8080/by-location?lat=40.7580&lon=-73.9855"

# Test route endpoint
ab -n 100 -c 10 http://localhost:8080/by-route/N

# Test with keep-alive
ab -n 1000 -c 20 -k http://localhost:8080/routes
```

## Testing with Postman

Import this collection:

```json
{
  "info": {
    "name": "MTA API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Get API Info",
      "request": {
        "method": "GET",
        "url": "{{baseUrl}}/"
      }
    },
    {
      "name": "Get Stations by Location",
      "request": {
        "method": "GET",
        "url": "{{baseUrl}}/by-location?lat=40.7580&lon=-73.9855"
      }
    },
    {
      "name": "Get Stations by Route",
      "request": {
        "method": "GET",
        "url": "{{baseUrl}}/by-route/N"
      }
    },
    {
      "name": "Get Stations by IDs",
      "request": {
        "method": "GET",
        "url": "{{baseUrl}}/by-id/127,631,635"
      }
    },
    {
      "name": "Get Routes",
      "request": {
        "method": "GET",
        "url": "{{baseUrl}}/routes"
      }
    },
    {
      "name": "Get Alerts",
      "request": {
        "method": "GET",
        "url": "{{baseUrl}}/alerts"
      }
    }
  ],
  "variable": [
    {
      "key": "baseUrl",
      "value": "http://localhost:8080",
      "type": "string"
    }
  ]
}
```

## Monitoring Response Times

```bash
# Simple timing
time curl -s "http://localhost:8080/by-location?lat=40.7580&lon=-73.9855" > /dev/null

# With detailed timing
curl -w "@curl-format.txt" -o /dev/null -s "http://localhost:8080/by-location?lat=40.7580&lon=-73.9855"
```

Create `curl-format.txt`:
```
    time_namelookup:  %{time_namelookup}s\n
       time_connect:  %{time_connect}s\n
    time_appconnect:  %{time_appconnect}s\n
   time_pretransfer:  %{time_pretransfer}s\n
      time_redirect:  %{time_redirect}s\n
 time_starttransfer:  %{time_starttransfer}s\n
                    ----------\n
         time_total:  %{time_total}s\n
```