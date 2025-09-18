package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type TimestampValidator struct{}

func (TimestampValidator) Validate(ctx context.Context, now int64, _ *Metadata, feed, prev, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	h := feed.GetHeader()
	ts := int64(h.GetTimestamp())
	if IsV2OrHigher(h) && ts == 0 {
		out = append(out, ErrorList{Rule: GetRule("E048"), Occurrences: []Occurrence{{Prefix: "header.timestamp"}}})
	} else if ts == 0 {
		out = append(out, ErrorList{Rule: GetRule("W001"), Occurrences: []Occurrence{{Prefix: "header.timestamp"}}})
	} else {
		if !IsPosix(ts) {
			out = append(out, ErrorList{Rule: GetRule("E001"), Occurrences: []Occurrence{{Prefix: "header.timestamp"}}})
		}
		if GetAge(now, ts) > 65 {
			out = append(out, ErrorList{Rule: GetRule("W008"), Occurrences: []Occurrence{{Prefix: "header.timestamp"}}})
		}
		if IsInFuture(now, ts, 60) {
			out = append(out, ErrorList{Rule: GetRule("E050"), Occurrences: []Occurrence{{Prefix: "header.timestamp"}}})
		}
		if prev != nil {
			pts := int64(prev.GetHeader().GetTimestamp())
			if pts == ts && prevSize(prev) != prevSize(feed) {
				out = append(out, ErrorList{Rule: GetRule("E017"), Occurrences: []Occurrence{{Prefix: "header.timestamp"}}})
			} else if ts < pts {
				out = append(out, ErrorList{Rule: GetRule("E018"), Occurrences: []Occurrence{{Prefix: "header.timestamp"}}})
			} else if ts-pts > 35 {
				out = append(out, ErrorList{Rule: GetRule("W007"), Occurrences: []Occurrence{{Prefix: "header.timestamp"}}})
			}
		}
	}
	for _, e := range feed.GetEntity() {
		if tu := e.GetTripUpdate(); tu != nil {
			ets := int64(tu.GetTimestamp())
			if ets == 0 {
				out = append(out, el("W001", "trip_update.timestamp"))
				continue
			}
			if ets > ts {
				out = append(out, el("E012", "trip_update.timestamp"))
			}
			if !IsPosix(ets) {
				out = append(out, el("E001", "trip_update.timestamp"))
			}
			if IsInFuture(now, ets, 60) {
				out = append(out, el("E050", "trip_update.timestamp"))
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			ets := int64(vp.GetTimestamp())
			if ets == 0 {
				out = append(out, el("W001", "vehicle.timestamp"))
				continue
			}
			if ets > ts {
				out = append(out, el("E012", "vehicle.timestamp"))
			}
			if !IsPosix(ets) {
				out = append(out, el("E001", "vehicle.timestamp"))
			}
			if IsInFuture(now, ets, 60) {
				out = append(out, el("E050", "vehicle.timestamp"))
			}
		}
	}
	return out
}

func el(id, prefix string) ErrorList {
	return ErrorList{Rule: GetRule(RuleID(id)), Occurrences: []Occurrence{{Prefix: prefix}}}
}

func prevSize(m *rt.FeedMessage) int {
	if m == nil {
		return 0
	}
	return len(m.GetEntity())
}
