package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"your.org/gtfsrt/validator/notice"
	"your.org/gtfsrt/validator/schema"
	"your.org/gtfsrt/validator/util"
)

type HeaderValidator struct{}

func (HeaderValidator) Validate(ctx context.Context, now int64, _ *schema.Metadata, feed, prev, _ *rt.FeedMessage) []notice.ErrorList {
	var out []notice.ErrorList
	h := feed.GetHeader()
	// E038
	if !util.IsValidVersion(h) {
		out = append(out, notice.ErrorList{Rule: notice.GetRule("E038"), Occurrences: []notice.Occurrence{{Prefix: "header"}}})
	}
	// E049
	if util.IsV2OrHigher(h) && h.GetIncrementality().String() == "" {
		out = append(out, notice.ErrorList{Rule: notice.GetRule("E049"), Occurrences: []notice.Occurrence{{Prefix: "header"}}})
	}
	// E039
	if h.GetIncrementality() == rt.FeedHeader_FULL_DATASET {
		for _, e := range feed.GetEntity() {
			if e.GetIsDeleted() {
				out = append(out, notice.ErrorList{Rule: notice.GetRule("E039"), Occurrences: []notice.Occurrence{{Prefix: e.GetId()}}})
				break
			}
		}
	}
	return out
}
