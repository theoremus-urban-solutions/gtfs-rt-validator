# gtfs-rt-validator Library Enhancement Request

## Context

We are integrating the `gtfs-rt-validator` library into a high-throughput GTFS-RT proxy/conversion service. Currently, we already fetch and cache GTFS static data in memory for conversion purposes. To avoid fetching GTFS data twice (once for validation, once for conversion), we need the validator to accept raw GTFS zip data as bytes instead of only file paths.

## Current Limitation

The current `NewValidator()` function only accepts a file path:

```go
func NewValidator(gtfsPath string, opts Options) (*Validator, error)
```

This forces us to either:
- Write GTFS data to disk (unnecessary I/O overhead)
- Maintain two separate GTFS loading mechanisms (duplication)
- Skip full validation (unacceptable for data quality)

## Required Enhancement

### 1. Add New Constructor Function

**File**: `validators.go`

Add a new constructor that accepts raw GTFS zip bytes:

```go
// NewValidatorFromBytes creates a validator from raw GTFS zip data in memory.
// This allows integration with systems that already have GTFS data loaded,
// avoiding redundant disk I/O or duplicate fetching from remote sources.
//
// Parameters:
//   - zipData: Complete GTFS zip file as byte slice
//   - opts: Validator options (same as NewValidator)
//
// Returns:
//   - *Validator: Initialized validator instance
//   - error: Any error during GTFS parsing or validation setup
//
// Example:
//   gtfsBytes, _ := os.ReadFile("gtfs.zip")
//   validator, err := NewValidatorFromBytes(gtfsBytes, Options{EnableShapes: true})
//   if err != nil {
//       log.Fatal(err)
//   }
//   valid, result := validator.ValidateIteration(ctx, feed, nil)
func NewValidatorFromBytes(zipData []byte, opts Options) (*Validator, error) {
    md, err := LoadMetadataFromBytes(zipData, !opts.EnableShapes)
    if err != nil {
        return nil, err
    }
    return &Validator{opts: opts, meta: md, rtOnly: false}, nil
}
```

### 2. Add Bytes-Based Metadata Loading

**File**: `parser.go`

Add a new function to load GTFS metadata from raw bytes:

```go
import (
    "archive/zip"
    "bytes"
    "encoding/csv"
    "fmt"
    "io"
    "strings"
)

// LoadMetadataFromBytes parses GTFS metadata from raw zip file bytes.
// This is used by NewValidatorFromBytes to load GTFS data from memory.
//
// Parameters:
//   - zipData: Complete GTFS zip file as byte slice
//   - ignoreShapes: If true, skip parsing shapes.txt (faster, less memory)
//
// Returns:
//   - *Metadata: Parsed GTFS metadata structure
//   - error: Any error during zip extraction or GTFS parsing
//
// The function:
//   1. Opens the zip archive from memory using bytes.Reader
//   2. Extracts all GTFS .txt files into CSV readers
//   3. Delegates to shared parsing logic (buildMetadataFromReaders)
//
// Supported files (same as LoadMetadata):
//   - agency.txt, routes.txt, trips.txt, stops.txt, stop_times.txt
//   - shapes.txt (optional), frequencies.txt (optional)
func LoadMetadataFromBytes(zipData []byte, ignoreShapes bool) (*Metadata, error) {
    // Create a reader from the byte slice
    r := bytes.NewReader(zipData)

    // Open as zip archive
    zipReader, err := zip.NewReader(r, int64(len(zipData)))
    if err != nil {
        return nil, fmt.Errorf("failed to open GTFS zip from bytes: %w", err)
    }

    // Extract all .txt files into CSV readers
    readers := make(map[string]*csv.Reader)
    var closers []io.Closer

    for _, f := range zipReader.File {
        if !strings.HasSuffix(strings.ToLower(f.Name), ".txt") {
            continue
        }

        rc, err := f.Open()
        if err != nil {
            // Close any already-opened files
            for _, c := range closers {
                c.Close()
            }
            return nil, fmt.Errorf("failed to open %s: %w", f.Name, err)
        }

        closers = append(closers, rc)
        readers[strings.ToLower(f.Name)] = csv.NewReader(rc)
    }

    // Ensure cleanup of all file handles
    defer func() {
        for _, c := range closers {
            c.Close()
        }
    }()

    // Use shared parsing logic
    return buildMetadataFromReaders(readers, ignoreShapes)
}
```

### 3. Refactor Existing LoadMetadata

**File**: `parser.go`

Refactor the existing `LoadMetadata()` function to share parsing logic:

```go
// LoadMetadata parses GTFS metadata from a file path (existing function).
// Now delegates to buildMetadataFromReaders for shared parsing logic.
//
// Parameters:
//   - path: File system path to GTFS zip or directory
//   - ignoreShapes: If true, skip parsing shapes.txt
//
// Returns:
//   - *Metadata: Parsed GTFS metadata structure
//   - error: Any error during file access or GTFS parsing
func LoadMetadata(path string, ignoreShapes bool) (*Metadata, error) {
    readers, cleanup, err := ZipOrDirReaders(path)
    if err != nil {
        return nil, err
    }
    defer cleanup()

    return buildMetadataFromReaders(readers, ignoreShapes)
}

// buildMetadataFromReaders contains the shared GTFS parsing logic.
// This function is called by both LoadMetadata (file path) and
// LoadMetadataFromBytes (raw bytes).
//
// Parameters:
//   - readers: Map of filename -> CSV reader for each GTFS .txt file
//   - ignoreShapes: If true, skip parsing shapes.txt
//
// Returns:
//   - *Metadata: Parsed GTFS metadata structure
//   - error: Any error during GTFS parsing or validation
//
// Implementation note:
//   This is the existing LoadMetadata body (lines 20+) extracted into
//   a separate function to avoid code duplication.
func buildMetadataFromReaders(readers map[string]*csv.Reader, ignoreShapes bool) (*Metadata, error) {
    // Move the existing LoadMetadata implementation here
    // This includes:
    //   - md := &Metadata{...} initialization
    //   - parseAgency(), parseRoutes(), parseTrips(), etc.
    //   - Building indexes and bounding boxes
    //   - Timezone parsing
    //   - All existing validation logic

    md := &Metadata{
        AgencyIDs:             make(map[string]struct{}),
        RouteIDs:              make(map[string]struct{}),
        TripsByTripID:         make(map[string]Trip),
        StopTimesByTripID:     make(map[string][]StopTime),
        StopIDs:               make(map[string]struct{}),
        StopToLocationType:    make(map[string]int),
        ExactTimesZeroTripIDs: make(map[string]struct{}),
        ExactTimesOneByTripID: make(map[string][]Frequency),
        ShapePointsByShapeID:  make(map[string][]ShapePoint),
        TripShapes:            make(map[string][][]float64),
        TripShapesBuffered:    make(map[string][][]float64),
        TripsWithMultiStops:   make(map[string][]string),
    }

    // Parse agency.txt (if exists)
    if r := readers["agency.txt"]; r != nil {
        if err := parseAgency(r, md); err != nil {
            return nil, err
        }
    }

    // Parse routes.txt (if exists)
    if r := readers["routes.txt"]; r != nil {
        if err := parseRoutes(r, md); err != nil {
            return nil, err
        }
    }

    // ... rest of existing parsing logic ...
    // (This is just the CURRENT LoadMetadata body starting from line ~20)

    return md, nil
}
```

## Expected Behavior

### Functional Requirements

1. **NewValidatorFromBytes() must produce identical validation results to NewValidator()** when given the same GTFS data
2. **All validation rules must work** (E001-E050, W001-W050, etc.)
3. **Shape validation (W024) must work** when `Options{EnableShapes: true}`
4. **Memory usage should be comparable** to file-based loading
5. **Error handling must be robust** - proper cleanup of resources on failure

### Performance Requirements

1. **Loading time should be similar** to file-based approach (within 10%)
2. **Memory footprint should be identical** to file-based approach
3. **No memory leaks** - all file handles and buffers properly closed

### API Requirements

1. **Backward compatibility** - existing NewValidator() must continue to work
2. **Consistent API** - both constructors use same Options struct
3. **Clear documentation** - docstrings explain use cases and parameters
4. **Same return types** - both return *Validator, error

## Testing Requirements

### Unit Tests Required

**File**: `validators_test.go`

```go
func TestNewValidatorFromBytes_BasicValidation(t *testing.T) {
    // Load GTFS zip as bytes
    gtfsBytes, err := os.ReadFile("testdata/sample-gtfs.zip")
    if err != nil {
        t.Fatal(err)
    }

    // Create validator from bytes
    v, err := NewValidatorFromBytes(gtfsBytes, Options{})
    if err != nil {
        t.Fatalf("NewValidatorFromBytes failed: %v", err)
    }

    // Create a simple valid GTFS-RT feed
    feed := createTestFeed(t)

    // Validate
    valid, result := v.ValidateIteration(context.Background(), feed, nil)

    if !valid {
        t.Errorf("Expected valid feed, got errors: %v", result.Errors)
    }
}

func TestNewValidatorFromBytes_CompareWithFilePath(t *testing.T) {
    gtfsPath := "testdata/sample-gtfs.zip"

    // Load same GTFS via file path
    v1, err := NewValidator(gtfsPath, Options{})
    if err != nil {
        t.Fatal(err)
    }

    // Load same GTFS via bytes
    gtfsBytes, _ := os.ReadFile(gtfsPath)
    v2, err := NewValidatorFromBytes(gtfsBytes, Options{})
    if err != nil {
        t.Fatal(err)
    }

    // Validate same feed with both validators
    feed := createTestFeed(t)
    ctx := context.Background()

    valid1, result1 := v1.ValidateIteration(ctx, feed, nil)
    valid2, result2 := v2.ValidateIteration(ctx, feed, nil)

    // Results should be identical
    if valid1 != valid2 {
        t.Errorf("Validation results differ: file=%v, bytes=%v", valid1, valid2)
    }

    if len(result1.Errors) != len(result2.Errors) {
        t.Errorf("Error count differs: file=%d, bytes=%d",
            len(result1.Errors), len(result2.Errors))
    }

    if len(result1.Warnings) != len(result2.Warnings) {
        t.Errorf("Warning count differs: file=%d, bytes=%d",
            len(result1.Warnings), len(result2.Warnings))
    }
}

func TestNewValidatorFromBytes_WithShapes(t *testing.T) {
    gtfsBytes, _ := os.ReadFile("testdata/gtfs-with-shapes.zip")

    // Create validator with shapes enabled
    v, err := NewValidatorFromBytes(gtfsBytes, Options{EnableShapes: true})
    if err != nil {
        t.Fatal(err)
    }

    // Validate feed with shape data
    feed := createTestFeedWithShapes(t)
    valid, result := v.ValidateIteration(context.Background(), feed, nil)

    // Should detect W024 warnings if shapes don't match
    hasW024 := false
    for _, list := range result.Lists {
        if list.Rule.ID == "W024" {
            hasW024 = true
            break
        }
    }

    // Verify shape validation ran (may or may not have warnings)
    t.Logf("W024 shape validation executed: %v", hasW024)
}

func TestNewValidatorFromBytes_InvalidZip(t *testing.T) {
    invalidData := []byte("not a zip file")

    _, err := NewValidatorFromBytes(invalidData, Options{})
    if err == nil {
        t.Error("Expected error for invalid zip data")
    }

    if !strings.Contains(err.Error(), "failed to open") {
        t.Errorf("Expected 'failed to open' error, got: %v", err)
    }
}

func TestNewValidatorFromBytes_EmptyZip(t *testing.T) {
    // Create empty zip
    buf := new(bytes.Buffer)
    zw := zip.NewWriter(buf)
    zw.Close()

    _, err := NewValidatorFromBytes(buf.Bytes(), Options{})
    // Should handle gracefully (empty Metadata, no errors)
    if err != nil {
        t.Errorf("Unexpected error for empty zip: %v", err)
    }
}
```

### Integration Tests Required

```go
func TestLoadMetadataFromBytes_AllFiles(t *testing.T) {
    // Test that all GTFS files are parsed correctly
    gtfsBytes, _ := os.ReadFile("testdata/complete-gtfs.zip")

    md, err := LoadMetadataFromBytes(gtfsBytes, false)
    if err != nil {
        t.Fatal(err)
    }

    // Verify all expected data was loaded
    if len(md.AgencyIDs) == 0 {
        t.Error("No agencies loaded")
    }
    if len(md.RouteIDs) == 0 {
        t.Error("No routes loaded")
    }
    if len(md.TripsByTripID) == 0 {
        t.Error("No trips loaded")
    }
    if len(md.StopTimesByTripID) == 0 {
        t.Error("No stop times loaded")
    }
    if len(md.StopIDs) == 0 {
        t.Error("No stops loaded")
    }
}
```

## Usage Examples

### Example 1: Basic Usage

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
    // Load GTFS static from disk
    gtfsBytes, err := os.ReadFile("schedule.zip")
    if err != nil {
        panic(err)
    }

    // Create validator from bytes
    v, err := validator.NewValidatorFromBytes(gtfsBytes, validator.Options{
        EnableShapes: true,
    })
    if err != nil {
        panic(err)
    }

    // Load GTFS-RT feed
    rtBytes, _ := os.ReadFile("tripupdates.pb")
    feed := &rt.FeedMessage{}
    validator.UnmarshalProtoTextOrBinary(rtBytes, feed)

    // Validate
    valid, result := v.ValidateIteration(context.Background(), feed, nil)

    fmt.Printf("Valid: %v\n", valid)
    fmt.Printf("Errors: %d, Warnings: %d\n", len(result.Errors), len(result.Warnings))
}
```

### Example 2: Fetch from HTTP and Validate

```go
func validateRemoteGTFS(gtfsURL string, rtFeedURL string) error {
    // Fetch GTFS static from URL
    resp, err := http.Get(gtfsURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    gtfsBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    // Create validator from downloaded bytes
    v, err := validator.NewValidatorFromBytes(gtfsBytes, validator.Options{})
    if err != nil {
        return err
    }

    // Fetch and validate GTFS-RT feed
    rtResp, _ := http.Get(rtFeedURL)
    defer rtResp.Body.Close()

    rtBytes, _ := io.ReadAll(rtResp.Body)
    feed := &rt.FeedMessage{}
    validator.UnmarshalProtoTextOrBinary(rtBytes, feed)

    valid, result := v.ValidateIteration(context.Background(), feed, nil)

    if !valid {
        return fmt.Errorf("validation failed: %d errors", len(result.Errors))
    }

    return nil
}
```

### Example 3: Reuse with Multiple Feeds

```go
func validateMultipleFeeds(gtfsBytes []byte, rtFeeds [][]byte) {
    // Create validator once
    v, _ := validator.NewValidatorFromBytes(gtfsBytes, validator.Options{})

    // Reuse for multiple feed validations
    for i, rtBytes := range rtFeeds {
        feed := &rt.FeedMessage{}
        validator.UnmarshalProtoTextOrBinary(rtBytes, feed)

        valid, result := v.ValidateIteration(context.Background(), feed, nil)

        fmt.Printf("Feed %d: valid=%v, errors=%d, warnings=%d\n",
            i, valid, len(result.Errors), len(result.Warnings))
    }
}
```

## Documentation Requirements

### README.md Updates

Add section after existing NewValidator() documentation:

```markdown
### Creating a Validator from In-Memory GTFS Data

If you already have GTFS static data in memory (e.g., fetched from a database or HTTP endpoint), you can avoid disk I/O by using `NewValidatorFromBytes()`:

\`\`\`go
// Load GTFS from any source
gtfsBytes := fetchGTFSFromDatabase()

// Create validator directly from bytes
validator, err := NewValidatorFromBytes(gtfsBytes, Options{
    EnableShapes: true,
})
if err != nil {
    log.Fatal(err)
}

// Use the validator normally
valid, result := validator.ValidateIteration(ctx, feed, nil)
\`\`\`

This is particularly useful when:
- Fetching GTFS from HTTP APIs or S3
- Using the same GTFS data for both validation and other processing
- Working in memory-constrained environments where disk I/O should be minimized
- Building high-throughput validation services
```

### API Documentation (godoc)

Ensure all new functions have complete godoc comments with:
- Clear description of purpose
- Parameter documentation
- Return value documentation
- Usage examples
- Notes on behavior and edge cases

## Deliverables

1. ✅ **Code changes** in `validators.go` and `parser.go`
2. ✅ **Unit tests** with >90% coverage of new code
3. ✅ **Integration tests** comparing file vs bytes approaches
4. ✅ **README updates** with examples
5. ✅ **Godoc comments** on all new exported functions
6. ✅ **Version bump** to v0.2.0 (semantic versioning - new feature)
7. ✅ **Git tag** v0.2.0 after merge

## Success Criteria

- [ ] All existing tests pass
- [ ] New tests pass with >90% coverage
- [ ] NewValidatorFromBytes() produces identical results to NewValidator()
- [ ] No memory leaks (run with `go test -race`)
- [ ] Documentation is clear and complete
- [ ] Backward compatible (no breaking changes)
- [ ] Performance is comparable (benchmark if needed)

## Questions or Clarifications

If anything is unclear, please ask:
- Repository: https://github.com/theoremus-urban-solutions/gtfs-rt-validator
- Use case: High-throughput transit data proxy/conversion service
- Goal: Avoid fetching GTFS data twice (once for validation, once for conversion)

Thank you for implementing this enhancement!
