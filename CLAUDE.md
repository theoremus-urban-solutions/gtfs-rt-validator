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

1. **Static loading**: `LoadStaticIndex` / `LoadStaticIndexPath` (`static_index.go`) build the compact `*StaticIndex` the constructors use; `LoadMetadata` (`parser.go`) builds the older map-based `*Metadata`. Both implement `Static` (`static.go`) and both loads are strict: a missing required file or column, or a CSV error, fails the load.
2. **Validator Registry** (`validator_registry.go`): `DefaultValidators()` returns all active validators.
3. **Validation Execution** (`check.go`): `ValidateFeed(ctx, static, feed, prev, opts)` is stateless and safe for concurrent feeds; the caller keeps one `PrevHeader` per feed. `Validator.ValidateIteration` (`validators.go`) wraps it for the CLI and remembers its own `PrevHeader`.
4. **Output** (`report.go`): Results rendered as console, JSON, summary, or Java-compatible JSON format.

### Validator Interface

All validators implement `FeedEntityValidator`:
```go
type FeedEntityValidator interface {
    Validate(c *Check)
}
```

`Check` (`check.go`) carries `Ctx`, `NowMs` (Unix milliseconds), `Static` (nil when only GTFS-RT rules run), `Feed`, `Prev` (the previous header of this same feed, for E017 and E018) and `Combined` (the merged feed for cross-feed rules such as E047). A rule reports with `c.add(rule, prefix, entityID, value, details)`.

`Static` answers `Trip`, `Stop`, `Route` and `Agency` with a `Presence`: `Present`, `Absent`, or `NotCovered` when the release cannot judge that id, in which case the rule skips its static check.

### Behaviour matched against MobilityData's validator

- `canonical_test.go` holds one case per reviewed verdict, each with MobilityData's result, and runs every case on both `*Metadata` and `*StaticIndex`.
- Test whether a proto field is set (`StopSequence != nil`, `Delay == nil`, `Incrementality == nil`), never its getter, which returns zero for an unset field.
- `static_index_test.go` checks `*StaticIndex` against `*Metadata` trip by trip; with `GTFS_RT_SOFIA_ZIP` set to a real release it also runs the Sofia parity, load budget and benchmarks.

### Rule Definitions

Rules are defined in `notice.go` with IDs like E001, W001. The prefix indicates severity:
- `E` = Error (causes validation failure)
- `W` = Warning (informational)

### Protobuf Handling

Uses `github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs` for GTFS-RT protobuf types. The `protobuf.go` file handles parsing both binary and text-format protobufs.
