# GTFS-Realtime Validator

A high-performance GTFS-Realtime validator written in Go that validates real-time transit feeds against their static GTFS schedule data.

## Features

- Validates GTFS-RT feeds (TripUpdates, VehiclePositions, Alerts) against GTFS static schedules
- Supports realtime-only validation mode (without static GTFS)
- Multiple output formats: console, JSON, summary, Java-compatible JSON
- Batch validation of multiple .pb files

## Installation

```bash
go get github.com/theoremus-urban-solutions/gtfs-rt-validator
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
    validator "github.com/theoremus-urban-solutions/gtfs-rt-validator"
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

### Creating a Validator from In-Memory GTFS Data

If you already have GTFS static data in memory (e.g., fetched from a database or HTTP endpoint), you can avoid disk I/O by using `NewValidatorFromBytes()`:

```go
// Load GTFS from any source (HTTP, database, S3, etc.)
gtfsBytes, _ := fetchGTFSFromRemote()

// Create validator directly from bytes
v, err := validator.NewValidatorFromBytes(gtfsBytes, validator.Options{
    EnableShapes: true,
})
if err != nil {
    log.Fatal(err)
}

// Use the validator normally
valid, result := v.ValidateIteration(ctx, feed, nil)
```

This is particularly useful when:
- Fetching GTFS from HTTP APIs or S3
- Using the same GTFS data for both validation and other processing
- Working in memory-constrained environments where disk I/O should be minimized
- Building high-throughput validation services

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

Sofia, 29 October 2026. One saved snapshot of the static GTFS and the three realtime feeds was checked by v0.6.0 and by v0.7.0, on a Linux VM with 8 vCPUs and Go 1.24.2. Each feed is checked on its own, with `NewValidator(zip, Options{EnableShapes: true})` and `ValidateIteration(ctx, feed, nil)`.

| Input | Size | Content |
|---|---|---|
| Static GTFS | 19.0 MB zip | 29,199 trips, 677,289 stop_times, 823,645 shape points |
| TripUpdates | 903 KB | 1,278 entities |
| VehiclePositions | 66 KB | 573 entities |
| Alerts | 104 KB | 21 entities |

| Measure | v0.6.0 | v0.7.0 |
|---|---|---|
| Static load, median of 5 | 2.65 s | 0.79 s |
| Memory allocated while loading | 567 MiB | 21 MiB |
| Heap kept after loading | 201 MiB | 8.4 MiB |
| Peak RSS, load and all 3 feeds | 337 MiB | 44 MiB |
| TripUpdates check, median of 30 | 10.8 ms | 8.4 ms |
| VehiclePositions check, median of 30 | 0.57 ms | 0.65 ms |
| Alerts check, median of 30 | 3 µs | 3 µs |

The saving comes from `StaticIndex`, which keeps only the static columns the rules read.

Both versions report the same errors and warnings, on this snapshot and on five more taken 30 s apart:

- E022 46–54 times per TripUpdates feed;
- W002 on every TripUpdate;
- W004 on 158–183 vehicles;
- W008 whenever a header is over 65 s old.

None of the rules v0.7.0 corrected fire on Sofia's data.

Times are wall clock on a shared VM. Peak RSS is the process `VmHWM`. Protobuf decoding is not included.
