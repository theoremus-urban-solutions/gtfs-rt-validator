package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type HeaderValidator struct{}

func (HeaderValidator) Validate(ctx context.Context, now int64, _ *Metadata, feed, prev, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	h := feed.GetHeader()
	if !IsValidVersion(h) {
		out = append(out, ErrorList{Rule: GetRule("E038"), Occurrences: []Occurrence{{Prefix: "header"}}})
	}
	if IsV2OrHigher(h) && h.GetIncrementality().String() == "" {
		out = append(out, ErrorList{Rule: GetRule("E049"), Occurrences: []Occurrence{{Prefix: "header"}}})
	}
	if h.GetIncrementality() == rt.FeedHeader_FULL_DATASET {
		for _, e := range feed.GetEntity() {
			if e.GetIsDeleted() {
				out = append(out, ErrorList{Rule: GetRule("E039"), Occurrences: []Occurrence{{Prefix: e.GetId()}}})
				break
			}
		}
	}
	return out
}
