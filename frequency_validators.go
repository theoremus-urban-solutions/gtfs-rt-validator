package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type FreqZeroValidator struct{}
type FreqOneValidator struct{}

func (FreqZeroValidator) Validate(ctx context.Context, _ int64, meta *Metadata, feed, _, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	for _, e := range feed.GetEntity() {
		eid := e.GetId()
		if tu := e.GetTripUpdate(); tu != nil {
			tid := tu.GetTrip().GetTripId()
			if _, ok := meta.ExactTimesZeroTripIDs[tid]; ok {
				vid := ""
				if tu.GetVehicle() != nil {
					vid = tu.GetVehicle().GetId()
				}
				if tu.GetTrip().GetStartDate() == "" {
					out = append(out, errorListDetailed("E006", "trip_id "+tid+" missing start_date", eid, "", map[string]string{
						"trip_id":       tid,
						"missing_field": "start_date",
					}))
				}
				if tu.GetTrip().GetStartTime() == "" {
					out = append(out, errorListDetailed("E006", "trip_id "+tid+" missing start_time", eid, "", map[string]string{
						"trip_id":       tid,
						"missing_field": "start_time",
					}))
				}
				if tu.GetTrip().GetTripId() == "" {
					out = append(out, errorListDetailed("E006", "missing trip_id", eid, "", map[string]string{
						"missing_field": "trip_id",
					}))
				}
				if tu.GetTrip().GetScheduleRelationship() != rt.TripDescriptor_UNSCHEDULED {
					out = append(out, errorListDetailed("E013", "trip_id "+tid, eid, tu.GetTrip().GetScheduleRelationship().String(), map[string]string{
						"trip_id":               tid,
						"schedule_relationship": tu.GetTrip().GetScheduleRelationship().String(),
						"expected":              "UNSCHEDULED",
					}))
				}
				if vid == "" {
					out = append(out, errorListDetailed("W005", "trip_id "+tid, eid, "", map[string]string{
						"trip_id": tid,
					}))
				}
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			if vp.GetTrip() != nil {
				tid := vp.GetTrip().GetTripId()
				if _, ok := meta.ExactTimesZeroTripIDs[tid]; ok {
					vid := ""
					if vp.GetVehicle() != nil {
						vid = vp.GetVehicle().GetId()
					}
					if vid == "" {
						out = append(out, errorListDetailed("W005", "trip_id "+tid, eid, "", map[string]string{
							"trip_id": tid,
						}))
					}
				}
			}
		}
	}
	return out
}

func (FreqOneValidator) Validate(ctx context.Context, _ int64, meta *Metadata, feed, _, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	for _, e := range feed.GetEntity() {
		eid := e.GetId()
		if tu := e.GetTripUpdate(); tu != nil {
			tid := tu.GetTrip().GetTripId()
			freqs := meta.ExactTimesOneByTripID[tid]
			if len(freqs) == 0 {
				continue
			}
			st := tu.GetTrip().GetStartTime()
			if st == "" {
				continue
			}
			match := false
			for range freqs {
				match = true
				break
			}
			if !match {
				out = append(out, errorListDetailed("E019", "trip_id "+tid, eid, st, map[string]string{
					"trip_id":    tid,
					"start_time": st,
				}))
			}
		}
	}
	return out
}
