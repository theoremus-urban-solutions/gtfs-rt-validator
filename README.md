# GTFS-Realtime Validator

A high-performance GTFS-Realtime validator written in Go that validates real-time transit feeds against their static GTFS schedule data.

## Features

- Validates GTFS-RT feeds (TripUpdates, VehiclePositions, Alerts) against GTFS static schedules
- Supports realtime-only validation mode (without static GTFS)
- Multiple output formats: console, JSON, summary, Java-compatible JSON
- Batch validation of multiple .pb files

## Installation

```bash
go get your.org/gtfsrt/validator
```

## CLI Usage

```bash
# Build the CLI
go build -o gtfsrt-validator ./cmd/gtfsrt-validator

# Full validation (GTFS Static + Realtime)
./gtfsrt-validator validate -gtfs schedule.zip -rt realtime.pb

# Multi-feed validation
./gtfsrt-validator validate -gtfs schedule.zip \
  -tu tripupdates.pb \
  -vp vehiclepositions.pb \
  -al alerts.pb

# Realtime-only validation (no static GTFS)
./gtfsrt-validator validate -rt-only \
  -tu tripupdates.pb \
  -vp vehiclepositions.pb

# Batch validation
./gtfsrt-validator validate -gtfs schedule.zip -rt-path ./feeds/ -format summary
```

### CLI Flags

| Flag | Description |
|------|-------------|
| `-gtfs` | Path to GTFS static zip file |
| `-rt` | Path to a single GTFS-RT .pb file |
| `-tu` | TripUpdates .pb file |
| `-vp` | VehiclePositions .pb file |
| `-al` | Alerts .pb file |
| `-rt-path` | Directory of .pb files for batch validation |
| `-rt-only` | Validate RT feeds without static GTFS (spec compliance only) |
| `-format` | Output format: `console`, `json`, `summary`, `javajson` |
| `-sort` | Sort order for batch mode: `name`, `date` |

## Library Integration

The validator can be used as a Go library for integration into your own applications.

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "os"

    rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
    validator "your.org/gtfsrt/validator"
)

func main() {
    // Create validator with static GTFS
    v, err := validator.NewValidator("schedule.zip", validator.Options{
        EnableShapes: true, // Enable shape-based validation (W024)
    })
    if err != nil {
        panic(err)
    }

    // Parse GTFS-RT feed (supports both binary and text protobuf)
    data, _ := os.ReadFile("tripupdates.pb")
    feed := &rt.FeedMessage{}
    if err := validator.UnmarshalProtoTextOrBinary(data, feed); err != nil {
        panic(err)
    }

    // Validate
    ctx := context.Background()
    valid, result := v.ValidateIteration(ctx, feed, nil)

    fmt.Printf("Valid: %v\n", valid)
    fmt.Printf("Errors: %d, Warnings: %d\n", len(result.Errors), len(result.Warnings))

    // Render results
    fmt.Println(validator.RenderConsole(result))
}
```

### Realtime-Only Validation

For fast validation without loading static GTFS (useful for quick spec compliance checks):

```go
// Create RT-only validator (no static GTFS required)
v := validator.NewRTOnlyValidator(validator.Options{})

valid, result := v.ValidateIteration(ctx, feed, nil)
```

### Multi-Feed Validation

Validate multiple RT feeds together (enables cross-feed checks like W003, W033):

```go
v, _ := validator.NewValidator("schedule.zip", validator.Options{})

// Load feeds
tuFeed, _ := loadFeed("tripupdates.pb")
vpFeed, _ := loadFeed("vehiclepositions.pb")
alFeed, _ := loadFeed("alerts.pb")

// Combine feeds for cross-feed validation
combined := validator.BuildCombined(tuFeed, vpFeed, alFeed)

// Validate with combined feed
valid, result := v.ValidateIteration(ctx, tuFeed, combined)
```

### Output Formats

```go
// Console output (human-readable)
fmt.Println(validator.RenderConsole(result))

// JSON output
jsonBytes, _ := validator.MarshalJSON(result)

// Summary (counts per rule)
summary := validator.Summarize(result)

// Java-compatible JSON (for compatibility with MobilityData validator)
compat := validator.ToCompat(result)
```

### Validator Interface

All validators implement the `FeedEntityValidator` interface:

```go
type FeedEntityValidator interface {
    Validate(ctx context.Context, now int64, meta *Metadata, feed, prev, combined *rt.FeedMessage) []ErrorList
}
```

Parameters:
- `now`: Current timestamp in milliseconds
- `meta`: Parsed static GTFS metadata (nil for RT-only mode)
- `feed`: Current GTFS-RT feed being validated
- `prev`: Previous feed iteration (for sequential checks like E017, E018)
- `combined`: Merged feed from multiple sources (for cross-feed checks)

### Validation Rules

See [RULES.md](RULES.md) for the complete list of validation rules.

## Benchmarks

Benchmarks run on Apple Silicon (M-series), December 2, 2025.

### Test Data

| Dataset | GTFS Static | Trip Updates | Vehicle Positions | Alerts |
|---------|-------------|--------------|-------------------|--------|
| Sofia   | 15.0 MB     | 931 KB       | 73 KB             | 151 KB |
| Pleven  | 218 KB      | 2.2 KB       | -                 | -      |

### Sofia - Full Validation (GTFS Static + 3 RT Feeds)

Validates TripUpdates, VehiclePositions, and Alerts against a 15 MB static GTFS.

| Run | Time (s) | Memory (MB) |
|-----|----------|-------------|
| 1   | 1.10     | 272         |
| 2   | 0.63     | 272         |
| 3   | 0.62     | 268         |
| 4   | 0.61     | 269         |
| 5   | 0.61     | 265         |

**Average: 0.71s, ~269 MB peak memory**

### Sofia - Realtime-Only Validation (No Static GTFS)

Validates TripUpdates, VehiclePositions, and Alerts without static GTFS reference.

| Run | Time (ms) | Memory (MB) |
|-----|-----------|-------------|
| 1   | 10        | 21          |
| 2   | 10        | 21          |
| 3   | 10        | 21          |
| 4   | 10        | 21          |
| 5   | 10        | 21          |

**Average: 10ms, ~21 MB peak memory**

### Pleven - Full Validation (GTFS Static + TripUpdates)

Validates TripUpdates against a 218 KB static GTFS.

| Run | Time (ms) | Memory (MB) |
|-----|-----------|-------------|
| 1   | 20        | 17          |
| 2   | 10        | 17          |
| 3   | 10        | 16          |
| 4   | 10        | 16          |
| 5   | 10        | 17          |

**Average: 12ms, ~17 MB peak memory**

### Pleven - Realtime-Only Validation (No Static GTFS)

Validates TripUpdates without static GTFS reference.

| Run | Time (ms) | Memory (MB) |
|-----|-----------|-------------|
| 1   | 10        | 6           |
| 2   | <5        | 6           |
| 3   | <5        | 6           |
| 4   | <5        | 6           |
| 5   | <5        | 6           |

**Average: <5ms, ~6 MB peak memory**

### Performance Summary

| Test Case | Avg Time | Peak Memory | Speedup vs Full |
|-----------|----------|-------------|-----------------|
| Sofia Full (15MB static, 1MB RT) | 0.71s | 269 MB | - |
| Sofia RT-Only (1MB RT) | 10ms | 21 MB | 71x faster, 13x less memory |
| Pleven Full (218KB static, 2KB RT) | 12ms | 17 MB | - |
| Pleven RT-Only (2KB RT) | <5ms | 6 MB | ~3x faster, 3x less memory |

### Key Findings

1. **RT-only mode is significantly faster** - For Sofia (large dataset), RT-only validation is ~71x faster than full validation
2. **Memory scales with GTFS static size** - Most memory usage comes from parsing the static GTFS
3. **Small feeds validate nearly instantly** - Pleven RT-only completes in under 5ms
4. **First run includes warmup** - Run 1 of Sofia full validation includes file system cache warming (~1.1s vs ~0.6s subsequent runs)
