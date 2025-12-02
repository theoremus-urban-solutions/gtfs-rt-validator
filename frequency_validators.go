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
		if tu := e.GetTripUpdate(); tu != nil {
			tid := tu.GetTrip().GetTripId()
			if _, ok := meta.ExactTimesZeroTripIDs[tid]; ok {
				if tu.GetTrip().GetStartDate() == "" {
					out = append(out, el("E006", "trip_id "+tid+" missing start_date"))
				}
				if tu.GetTrip().GetStartTime() == "" {
					out = append(out, el("E006", "trip_id "+tid+" missing start_time"))
				}
				if tu.GetTrip().GetTripId() == "" {
					out = append(out, el("E006", "missing trip_id"))
				}
				if tu.GetTrip().GetScheduleRelationship() != rt.TripDescriptor_UNSCHEDULED {
					out = append(out, el("W019", "trip_id "+tid))
				}
				if tu.GetVehicle() == nil || tu.GetVehicle().GetId() == "" {
					out = append(out, el("W005", "trip_id "+tid))
				}
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			if vp.GetTrip() != nil {
				tid := vp.GetTrip().GetTripId()
				if _, ok := meta.ExactTimesZeroTripIDs[tid]; ok {
					if vp.GetVehicle() == nil || vp.GetVehicle().GetId() == "" {
						out = append(out, el("W005", "trip_id "+tid))
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
				out = append(out, el("E019", "trip_id "+tid))
			}
		}
	}
	return out
}
