package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"your.org/gtfsrt/validator/notice"
	"your.org/gtfsrt/validator/schema"
	"your.org/gtfsrt/validator/util"
)

type TimestampValidator struct{}

func (TimestampValidator) Validate(ctx context.Context, now int64, _ *schema.Metadata, feed, prev, _ *rt.FeedMessage) []notice.ErrorList {
	var out []notice.ErrorList
	h := feed.GetHeader()
	ts := int64(h.GetTimestamp())
	if util.IsV2OrHigher(h) && ts == 0 {
		out = append(out, notice.ErrorList{Rule: notice.GetRule("E048"), Occurrences: []notice.Occurrence{{Prefix: "header.timestamp"}}})
	} else if ts == 0 {
		out = append(out, notice.ErrorList{Rule: notice.GetRule("W001"), Occurrences: []notice.Occurrence{{Prefix: "header.timestamp"}}})
	} else {
		if !util.IsPosix(ts) {
			out = append(out, notice.ErrorList{Rule: notice.GetRule("E001"), Occurrences: []notice.Occurrence{{Prefix: "header.timestamp"}}})
		}
		// W008 age > 65s
		if util.GetAge(now, ts) > 65 {
			out = append(out, notice.ErrorList{Rule: notice.GetRule("W008"), Occurrences: []notice.Occurrence{{Prefix: "header.timestamp"}}})
		}
		// E050 future > 60s
		if util.IsInFuture(now, ts, 60) {
			out = append(out, notice.ErrorList{Rule: notice.GetRule("E050"), Occurrences: []notice.Occurrence{{Prefix: "header.timestamp"}}})
		}
		// E017/E018/W007 compare with previous header if present
		if prev != nil {
			pts := int64(prev.GetHeader().GetTimestamp())
			if pts == ts && prevSize(prev) != prevSize(feed) {
				out = append(out, notice.ErrorList{Rule: notice.GetRule("E017"), Occurrences: []notice.Occurrence{{Prefix: "header.timestamp"}}})
			} else if ts < pts {
				out = append(out, notice.ErrorList{Rule: notice.GetRule("E018"), Occurrences: []notice.Occurrence{{Prefix: "header.timestamp"}}})
			} else if ts-pts > 35 {
				out = append(out, notice.ErrorList{Rule: notice.GetRule("W007"), Occurrences: []notice.Occurrence{{Prefix: "header.timestamp"}}})
			}
		}
	}
	// Entities: W001, E012, E001, E050
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
			if !util.IsPosix(ets) {
				out = append(out, el("E001", "trip_update.timestamp"))
			}
			if util.IsInFuture(now, ets, 60) {
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
			if !util.IsPosix(ets) {
				out = append(out, el("E001", "vehicle.timestamp"))
			}
			if util.IsInFuture(now, ets, 60) {
				out = append(out, el("E050", "vehicle.timestamp"))
			}
		}
	}
	return out
}

func el(id, prefix string) notice.ErrorList {
	return notice.ErrorList{Rule: notice.GetRule(notice.RuleID(id)), Occurrences: []notice.Occurrence{{Prefix: prefix}}}
}

func prevSize(m *rt.FeedMessage) int {
	if m == nil {
		return 0
	}
	return len(m.GetEntity())
}
