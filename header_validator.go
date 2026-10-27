package validator

import (
	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type HeaderValidator struct{}

func (HeaderValidator) Validate(c *Check) {
	feed := c.Feed
	h := feed.GetHeader()
	version := h.GetGtfsRealtimeVersion()
	if !IsValidVersion(h) {
		c.add("E038", "header", "", version, map[string]string{
			"gtfs_realtime_version": version,
		})
	}
	if IsV2OrHigher(h) && h.GetIncrementality().String() == "" {
		c.add("E049", "header", "", "", map[string]string{
			"gtfs_realtime_version": version,
		})
	}
	if h.GetIncrementality() == rt.FeedHeader_FULL_DATASET {
		for _, e := range feed.GetEntity() {
			if e.GetIsDeleted() {
				c.add("E039", e.GetId(), e.GetId(), "", map[string]string{
					"entity_id":      e.GetId(),
					"incrementality": "FULL_DATASET",
				})
				break
			}
		}
	}
}
