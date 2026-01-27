package validator

import (
	"context"
	"fmt"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type TimestampValidator struct{}

func (TimestampValidator) Validate(ctx context.Context, now int64, _ *Metadata, feed, prev, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	h := feed.GetHeader()
	ts := int64(h.GetTimestamp())
	tsStr := fmt.Sprintf("%d", ts)
	if IsV2OrHigher(h) && ts == 0 {
		out = append(out, errorListDetailed("E048", "header.timestamp", "", "", nil))
	} else if ts == 0 {
		out = append(out, errorListDetailed("W001", "header.timestamp", "", "", nil))
	} else {
		if !IsPosix(ts) {
			out = append(out, errorListDetailed("E001", "header.timestamp", "", tsStr, nil))
		}
		if age := GetAge(now, ts); age > 65 {
			out = append(out, errorListDetailed("W008", "header.timestamp", "", tsStr, map[string]string{
				"age_seconds": fmt.Sprintf("%d", age),
			}))
		}
		if IsInFuture(now, ts, 60) {
			out = append(out, errorListDetailed("E050", "header.timestamp", "", tsStr, map[string]string{
				"now": fmt.Sprintf("%d", now),
			}))
		}
		if prev != nil {
			pts := int64(prev.GetHeader().GetTimestamp())
			if pts == ts && prevSize(prev) != prevSize(feed) {
				out = append(out, errorListDetailed("E017", "header.timestamp", "", tsStr, map[string]string{
					"previous_timestamp":    fmt.Sprintf("%d", pts),
					"previous_entity_count": fmt.Sprintf("%d", prevSize(prev)),
					"current_entity_count":  fmt.Sprintf("%d", prevSize(feed)),
				}))
			} else if ts < pts {
				out = append(out, errorListDetailed("E018", "header.timestamp", "", tsStr, map[string]string{
					"previous_timestamp": fmt.Sprintf("%d", pts),
				}))
			} else if ts-pts > 35 {
				out = append(out, errorListDetailed("W007", "header.timestamp", "", tsStr, map[string]string{
					"previous_timestamp": fmt.Sprintf("%d", pts),
					"interval_seconds":   fmt.Sprintf("%d", ts-pts),
				}))
			}
		}
	}
	for _, e := range feed.GetEntity() {
		if tu := e.GetTripUpdate(); tu != nil {
			ets := int64(tu.GetTimestamp())
			eid := e.GetId()
			tripID := tu.GetTrip().GetTripId()
			if ets == 0 {
				out = append(out, errorListDetailed("W001", "trip_update.timestamp", eid, "", map[string]string{
					"trip_id": tripID,
				}))
				continue
			}
			etsStr := fmt.Sprintf("%d", ets)
			if ets > ts {
				out = append(out, errorListDetailed("E012", "trip_update.timestamp", eid, etsStr, map[string]string{
					"trip_id":          tripID,
					"header_timestamp": tsStr,
				}))
			}
			if !IsPosix(ets) {
				out = append(out, errorListDetailed("E001", "trip_update.timestamp", eid, etsStr, map[string]string{
					"trip_id": tripID,
				}))
			}
			if IsInFuture(now, ets, 60) {
				out = append(out, errorListDetailed("E050", "trip_update.timestamp", eid, etsStr, map[string]string{
					"trip_id": tripID,
					"now":     fmt.Sprintf("%d", now),
				}))
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			ets := int64(vp.GetTimestamp())
			eid := e.GetId()
			vid := ""
			if vp.GetVehicle() != nil {
				vid = vp.GetVehicle().GetId()
			}
			if ets == 0 {
				out = append(out, errorListDetailed("W001", "vehicle.timestamp", eid, "", map[string]string{
					"vehicle_id": vid,
				}))
				continue
			}
			etsStr := fmt.Sprintf("%d", ets)
			if ets > ts {
				out = append(out, errorListDetailed("E012", "vehicle.timestamp", eid, etsStr, map[string]string{
					"vehicle_id":       vid,
					"header_timestamp": tsStr,
				}))
			}
			if !IsPosix(ets) {
				out = append(out, errorListDetailed("E001", "vehicle.timestamp", eid, etsStr, map[string]string{
					"vehicle_id": vid,
				}))
			}
			if IsInFuture(now, ets, 60) {
				out = append(out, errorListDetailed("E050", "vehicle.timestamp", eid, etsStr, map[string]string{
					"vehicle_id": vid,
					"now":        fmt.Sprintf("%d", now),
				}))
			}
		}
	}
	return out
}

func prevSize(m *rt.FeedMessage) int {
	if m == nil {
		return 0
	}
	return len(m.GetEntity())
}
