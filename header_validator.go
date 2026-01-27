package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type HeaderValidator struct{}

func (HeaderValidator) Validate(ctx context.Context, now int64, _ *Metadata, feed, prev, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	h := feed.GetHeader()
	version := h.GetGtfsRealtimeVersion()
	if !IsValidVersion(h) {
		out = append(out, errorListDetailed("E038", "header", "", version, map[string]string{
			"gtfs_realtime_version": version,
		}))
	}
	if IsV2OrHigher(h) && h.GetIncrementality().String() == "" {
		out = append(out, errorListDetailed("E049", "header", "", "", map[string]string{
			"gtfs_realtime_version": version,
		}))
	}
	if h.GetIncrementality() == rt.FeedHeader_FULL_DATASET {
		for _, e := range feed.GetEntity() {
			if e.GetIsDeleted() {
				out = append(out, errorListDetailed("E039", e.GetId(), e.GetId(), "", map[string]string{
					"entity_id":      e.GetId(),
					"incrementality": "FULL_DATASET",
				}))
				break
			}
		}
	}
	return out
}
