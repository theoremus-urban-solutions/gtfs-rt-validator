# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

```bash
# Build the CLI
go build -o gtfsrt-validator ./cmd/gtfsrt-validator

# Run linting
golangci-lint run

# Run tests (if tests exist)
go test ./...

# Validate a single GTFS-RT file
./gtfsrt-validator validate -gtfs <schedule.zip> -rt <realtime.pb>

# Validate with multiple RT feeds (combined checks)
./gtfsrt-validator validate -gtfs <schedule.zip> -rt-tu <tripupdates.pb> -rt-vp <vehiclepositions.pb>

# Batch validate a directory of .pb files
./gtfsrt-validator validate -gtfs <schedule.zip> -rt-path <dir> -format summary
```

## Architecture

This is a GTFS-Realtime validator written in Go. It validates real-time transit feeds against their static GTFS schedule data.

### Core Components

- **validator package** (root): Contains all validation logic and data structures
- **cmd/gtfsrt-validator/main.go**: CLI entry point with flag parsing and output formatting

### Validation Flow

1. **Metadata Loading** (`parser.go`): Parses static GTFS files (trips.txt, stops.txt, etc.) into a `Metadata` struct that caches IDs, stop times, frequencies, and bounding boxes
2. **Validator Registry** (`validator_registry.go`): `DefaultValidators()` returns all active validators
3. **Validation Execution** (`validators.go`): `ValidateIteration()` runs each validator and collects `ErrorList` results
4. **Output** (`report.go`): Results rendered as console, JSON, summary, or Java-compatible JSON format

### Validator Interface

All validators implement `FeedEntityValidator`:
```go
type FeedEntityValidator interface {
    Validate(ctx context.Context, now int64, meta *Metadata, feed, prev, combined *rt.FeedMessage) []ErrorList
}
```

- `now`: Current timestamp in milliseconds
- `meta`: Parsed static GTFS data
- `feed`: Current GTFS-RT feed being validated
- `prev`: Previous feed iteration (for sequential checks like E017, E018)
- `combined`: Merged feed when validating multiple sources (for cross-feed checks like E047)

### Rule Definitions

Rules are defined in `notice.go` with IDs like E001, W001. The prefix indicates severity:
- `E` = Error (causes validation failure)
- `W` = Warning (informational)

### Protobuf Handling

Uses `github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs` for GTFS-RT protobuf types. The `protobuf.go` file handles parsing both binary and text-format protobufs.
