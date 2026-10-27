package validator

import (
	"strconv"
)

type TimestampValidator struct{}

func (TimestampValidator) Validate(c *Check) {
	feed, now := c.Feed, c.NowMs
	h := feed.GetHeader()
	ts := int64(h.GetTimestamp())
	tsStr := func() string { return strconv.FormatInt(ts, 10) }
	if IsV2OrHigher(h) && ts == 0 {
		c.add("E048", "header.timestamp", "", "", nil)
	} else if ts == 0 {
		c.add("W001", "header.timestamp", "", "", nil)
	} else {
		if !IsPosix(ts) {
			c.add("E001", "header.timestamp", "", tsStr(), nil)
		}
		if age := GetAge(now, ts); age > 65 {
			c.add("W008", "header.timestamp", "", tsStr(), map[string]string{
				"age_seconds": strconv.FormatInt(age, 10),
			})
		}
		if IsInFuture(now, ts, 60) {
			c.add("E050", "header.timestamp", "", tsStr(), map[string]string{
				"now": strconv.FormatInt(now, 10),
			})
		}
		if prev := c.Prev; prev != nil {
			pts := int64(prev.Timestamp)
			n := len(feed.GetEntity())
			if pts == ts && prev.EntityCount != n {
				c.add("E017", "header.timestamp", "", tsStr(), map[string]string{
					"previous_timestamp":    strconv.FormatInt(pts, 10),
					"previous_entity_count": strconv.Itoa(prev.EntityCount),
					"current_entity_count":  strconv.Itoa(n),
				})
			} else if ts < pts {
				c.add("E018", "header.timestamp", "", tsStr(), map[string]string{
					"previous_timestamp": strconv.FormatInt(pts, 10),
				})
			} else if ts-pts > 35 {
				c.add("W007", "header.timestamp", "", tsStr(), map[string]string{
					"previous_timestamp": strconv.FormatInt(pts, 10),
					"interval_seconds":   strconv.FormatInt(ts-pts, 10),
				})
			}
		}
	}
	for _, e := range feed.GetEntity() {
		if tu := e.GetTripUpdate(); tu != nil {
			ets := int64(tu.GetTimestamp())
			eid := e.GetId()
			tripID := tu.GetTrip().GetTripId()
			if ets == 0 {
				c.add("W001", "trip_update.timestamp", eid, "", map[string]string{
					"trip_id": tripID,
				})
				continue
			}
			if ets > ts {
				c.add("E012", "trip_update.timestamp", eid, strconv.FormatInt(ets, 10), map[string]string{
					"trip_id":          tripID,
					"header_timestamp": tsStr(),
				})
			}
			if !IsPosix(ets) {
				c.add("E001", "trip_update.timestamp", eid, strconv.FormatInt(ets, 10), map[string]string{
					"trip_id": tripID,
				})
			}
			if IsInFuture(now, ets, 60) {
				c.add("E050", "trip_update.timestamp", eid, strconv.FormatInt(ets, 10), map[string]string{
					"trip_id": tripID,
					"now":     strconv.FormatInt(now, 10),
				})
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			ets := int64(vp.GetTimestamp())
			eid := e.GetId()
			vid := vp.GetVehicle().GetId()
			if ets == 0 {
				c.add("W001", "vehicle.timestamp", eid, "", map[string]string{
					"vehicle_id": vid,
				})
				continue
			}
			if ets > ts {
				c.add("E012", "vehicle.timestamp", eid, strconv.FormatInt(ets, 10), map[string]string{
					"vehicle_id":       vid,
					"header_timestamp": tsStr(),
				})
			}
			if !IsPosix(ets) {
				c.add("E001", "vehicle.timestamp", eid, strconv.FormatInt(ets, 10), map[string]string{
					"vehicle_id": vid,
				})
			}
			if IsInFuture(now, ets, 60) {
				c.add("E050", "vehicle.timestamp", eid, strconv.FormatInt(ets, 10), map[string]string{
					"vehicle_id": vid,
					"now":        strconv.FormatInt(now, 10),
				})
			}
		}
	}
}
