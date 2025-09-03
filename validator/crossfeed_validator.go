package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"your.org/gtfsrt/validator/notice"
	"your.org/gtfsrt/validator/schema"
)

type CrossFeedValidator struct{}

func (CrossFeedValidator) Validate(ctx context.Context, _ int64, _ *schema.Metadata, feed, _ *rt.FeedMessage, combined *rt.FeedMessage) []notice.ErrorList {
	var out []notice.ErrorList
	if combined == nil {
		return out
	}
	// Build maps: TU trip->vehicle, vehicle->trip; VP vehicle->trip, trip->vehicle
	tuTripToVeh := map[string]string{}
	tuVehToTrip := map[string]string{}
	vpVehToTrip := map[string]string{}
	vpTripToVeh := map[string]string{}

	var hasTU, hasVP bool
	for _, e := range combined.GetEntity() {
		if tu := e.GetTripUpdate(); tu != nil {
			hasTU = true
			tid := tu.GetTrip().GetTripId()
			vid := ""
			if tu.GetVehicle() != nil {
				vid = tu.GetVehicle().GetId()
			}
			if tid != "" && vid != "" {
				tuTripToVeh[tid] = vid
				tuVehToTrip[vid] = tid
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			hasVP = true
			tid := vp.GetTrip().GetTripId()
			vid := ""
			if vp.GetVehicle() != nil {
				vid = vp.GetVehicle().GetId()
			}
			if tid != "" && vid != "" {
				vpVehToTrip[vid] = tid
				vpTripToVeh[tid] = vid
			}
		}
	}
	if !hasTU || !hasVP {
		return out
	}

	// W003: IDs missing across feeds
	for tid := range tuTripToVeh {
		if _, ok := vpTripToVeh[tid]; !ok {
			out = append(out, el("W003", "trip_id "+tid))
		}
	}
	for vid := range vpVehToTrip {
		if _, ok := tuVehToTrip[vid]; !ok {
			out = append(out, el("W003", "vehicle_id "+vid))
		}
	}

	// E047: mismatched pairings
	for tid, vid := range tuTripToVeh {
		if v2, ok := vpTripToVeh[tid]; ok && v2 != vid {
			out = append(out, el("E047", "trip_id "+tid))
		}
	}
	return out
}
