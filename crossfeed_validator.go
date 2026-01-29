package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type CrossFeedValidator struct{}

func (CrossFeedValidator) Validate(ctx context.Context, _ int64, _ *Metadata, feed, _ *rt.FeedMessage, combined *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	if combined == nil {
		return out
	}
	tuTripToVeh := map[string]string{}
	tuVehToTrip := map[string]string{}
	vpVehToTrip := map[string]string{}
	vpTripToVeh := map[string]string{}
	tuTripToEntity := map[string]string{}
	vpVehToEntity := map[string]string{}

	var hasTU, hasVP bool
	for _, e := range combined.GetEntity() {
		eid := e.GetId()
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
				tuTripToEntity[tid] = eid
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
				vpVehToEntity[vid] = eid
			}
		}
	}
	if !hasTU || !hasVP {
		return out
	}

	for tid, vid := range tuTripToVeh {
		if _, ok := vpTripToVeh[tid]; !ok {
			out = append(out, errorListDetailed("W003", "trip_id "+tid, tuTripToEntity[tid], tid, map[string]string{
				"trip_id":         tid,
				"vehicle_id":      vid,
				"missing_in_feed": "VehiclePosition",
			}))
		}
	}
	for vid, tid := range vpVehToTrip {
		if _, ok := tuVehToTrip[vid]; !ok {
			out = append(out, errorListDetailed("W003", "vehicle_id "+vid, vpVehToEntity[vid], vid, map[string]string{
				"vehicle_id":      vid,
				"trip_id":         tid,
				"missing_in_feed": "TripUpdate",
			}))
		}
	}

	for tid, vid := range tuTripToVeh {
		if v2, ok := vpTripToVeh[tid]; ok && v2 != vid {
			out = append(out, errorListDetailed("E047", "trip_id "+tid, tuTripToEntity[tid], "", map[string]string{
				"trip_id":               tid,
				"trip_update_vehicle":   vid,
				"vehicle_position_vehicle": v2,
			}))
		}
	}
	return out
}
