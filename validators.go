package validator

import (
	"context"
	"sync"
	"time"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type FeedEntityValidator interface {
	Validate(c *Check)
}

// Validator checks successive builds of one feed and remembers the previous header between calls.
type Validator struct {
	opts   Options
	static Static

	mu   sync.Mutex
	prev *PrevHeader
}

type Options struct {
	// EnableShapes makes LoadMetadata read shapes.txt.
	EnableShapes bool
	// Now is the check time; zero means time.Now().
	Now time.Time
	// Combined is the merged TripUpdates and VehiclePositions feed for the cross-feed rules.
	Combined *rt.FeedMessage
	// MaxSamplesPerRule caps the occurrences kept per rule; 0 means 5 and a negative value keeps all.
	MaxSamplesPerRule int
}

// NewValidator creates a validator that requires static GTFS for full validation.
func NewValidator(gtfsZipPath string, opts Options) (*Validator, error) {
	md, err := LoadMetadata(gtfsZipPath, !opts.EnableShapes)
	if err != nil {
		return nil, err
	}
	return &Validator{opts: opts, static: md}, nil
}

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
//
//	gtfsBytes, _ := os.ReadFile("gtfs.zip")
//	validator, err := NewValidatorFromBytes(gtfsBytes, Options{EnableShapes: true})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	valid, result := validator.ValidateIteration(ctx, feed, nil)
func NewValidatorFromBytes(zipData []byte, opts Options) (*Validator, error) {
	md, err := LoadMetadataFromBytes(zipData, !opts.EnableShapes)
	if err != nil {
		return nil, err
	}
	return &Validator{opts: opts, static: md}, nil
}

// NewValidatorFromInput creates a validator from a GTFS source that can be
// either a local file path, directory, or HTTP/HTTPS URL.
// It automatically detects the input type and fetches/loads accordingly.
func NewValidatorFromInput(gtfsInput string, opts Options) (*Validator, error) {
	if IsURL(gtfsInput) {
		data, err := FetchInput(gtfsInput)
		if err != nil {
			return nil, err
		}
		return NewValidatorFromBytes(data, opts)
	}
	return NewValidator(gtfsInput, opts)
}

// NewRTOnlyValidator creates a validator that only checks GTFS-RT spec compliance
// without requiring static GTFS data. Use this for fast fail-fast validation
// before loading expensive static GTFS metadata.
func NewRTOnlyValidator(opts Options) *Validator {
	return &Validator{opts: opts}
}

// NewValidatorWithStatic creates a validator over an already loaded release; nil runs only the GTFS-RT rules.
func NewValidatorWithStatic(static Static, opts Options) *Validator {
	return &Validator{opts: opts, static: static}
}

// ValidateIteration checks the next build of the same feed and keeps every occurrence.
func (v *Validator) ValidateIteration(ctx context.Context, feed *rt.FeedMessage, combined *rt.FeedMessage) (bool, Result) {
	opts := v.opts
	opts.Combined = combined
	if opts.MaxSamplesPerRule == 0 {
		opts.MaxSamplesPerRule = -1
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	res := ValidateFeed(ctx, v.static, feed, v.prev, opts)
	h := HeaderOf(feed)
	v.prev = &h
	return res.Valid, res
}

// BuildCombined chooses the header with the max timestamp and concatenates entities.
func BuildCombined(feeds ...*rt.FeedMessage) *rt.FeedMessage {
	var maxH *rt.FeedHeader
	var entities []*rt.FeedEntity
	for _, f := range feeds {
		if f == nil {
			continue
		}
		h := f.GetHeader()
		if maxH == nil || h.GetTimestamp() > maxH.GetTimestamp() {
			maxH = h
		}
		entities = append(entities, f.GetEntity()...)
	}
	if maxH == nil {
		return nil
	}
	b := &rt.FeedMessage{}
	b.Header = maxH
	b.Entity = entities
	return b
}
