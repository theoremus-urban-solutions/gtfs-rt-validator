package validator

import (
	"context"
	"time"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type FeedEntityValidator interface {
	Validate(ctx context.Context, now int64, meta *Metadata, feed, prev, combined *rt.FeedMessage) []ErrorList
}

type Validator struct {
	opts   Options
	meta   *Metadata
	prev   *rt.FeedMessage
	rtOnly bool
}

type Options struct {
	EnableShapes bool
}

// NewValidator creates a validator that requires static GTFS for full validation.
func NewValidator(gtfsZipPath string, opts Options) (*Validator, error) {
	md, err := LoadMetadata(gtfsZipPath, !opts.EnableShapes)
	if err != nil {
		return nil, err
	}
	return &Validator{opts: opts, meta: md, rtOnly: false}, nil
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
	return &Validator{opts: opts, meta: md, rtOnly: false}, nil
}

// NewRTOnlyValidator creates a validator that only checks GTFS-RT spec compliance
// without requiring static GTFS data. Use this for fast fail-fast validation
// before loading expensive static GTFS metadata.
func NewRTOnlyValidator(opts Options) *Validator {
	return &Validator{opts: opts, meta: nil, rtOnly: true}
}

func (v *Validator) ValidateIteration(ctx context.Context, feed *rt.FeedMessage, combined *rt.FeedMessage) (bool, Result) {
	now := time.Now().UnixMilli()
	var lists []ErrorList
	validators := DefaultValidators()
	if v.rtOnly {
		validators = RTOnlyValidators()
	}
	for _, rule := range validators {
		lists = append(lists, rule.Validate(ctx, now, v.meta, feed, v.prev, combined)...)
	}
	v.prev = feed
	var errorsFlat, warningsFlat []Occurrence
	valid := true
	for _, l := range lists {
		if l.Rule.Severity == Error {
			if len(l.Occurrences) > 0 {
				valid = false
			}
			errorsFlat = append(errorsFlat, l.Occurrences...)
		} else if l.Rule.Severity == Warning {
			warningsFlat = append(warningsFlat, l.Occurrences...)
		}
	}
	res := Result{Lists: lists, Errors: errorsFlat, Warnings: warningsFlat}
	return valid, res
}

// note: for deterministic tests, wrap time.Now in tests if needed

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
