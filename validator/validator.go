package validator

import (
	"context"
	"time"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"your.org/gtfsrt/validator/notice"
	"your.org/gtfsrt/validator/parser"
	"your.org/gtfsrt/validator/schema"
)

type Options struct {
	EnableShapes bool
}

type Validator struct {
	opts Options
	meta *schema.Metadata
	prev *rt.FeedMessage
}

type Result = notice.Result

func NewValidator(gtfsZipPath string, opts Options) (*Validator, error) {
	md, err := parser.LoadMetadata(gtfsZipPath, !opts.EnableShapes)
	if err != nil {
		return nil, err
	}
	return &Validator{opts: opts, meta: md}, nil
}

func (v *Validator) ValidateIteration(ctx context.Context, feed *rt.FeedMessage, combined *rt.FeedMessage) (bool, Result) {
	now := time.Now().UnixMilli()
	var lists []notice.ErrorList
	for _, rule := range DefaultValidators() {
		lists = append(lists, rule.Validate(ctx, now, v.meta, feed, v.prev, combined)...)
	}
	v.prev = feed
	var errorsFlat, warningsFlat []notice.Occurrence
	valid := true
	for _, l := range lists {
		if l.Rule.Severity == notice.Error {
			if len(l.Occurrences) > 0 {
				valid = false
			}
			errorsFlat = append(errorsFlat, l.Occurrences...)
		} else if l.Rule.Severity == notice.Warning {
			warningsFlat = append(warningsFlat, l.Occurrences...)
		}
	}
	res := notice.Result{Lists: lists, Errors: errorsFlat, Warnings: warningsFlat}
	return valid, res
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
