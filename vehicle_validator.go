package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type VehicleValidator struct{}

func (VehicleValidator) Validate(ctx context.Context, _ int64, meta *Metadata, feed, _, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	seen := make(map[string]struct{})

	for _, e := range feed.GetEntity() {
		if tu := e.GetTripUpdate(); tu != nil {
			if tu.GetVehicle() == nil || tu.GetVehicle().GetId() == "" {
				out = append(out, el("W002", "trip_update"))
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			vid := ""
			if vp.GetVehicle() != nil {
				vid = vp.GetVehicle().GetId()
			}
			if vid == "" {
				out = append(out, el("W002", "vehicle"))
			} else {
				if _, dup := seen[vid]; dup {
					out = append(out, el("E052", "vehicle.id "+vid))
				} else {
					seen[vid] = struct{}{}
				}
			}

			pos := vp.GetPosition()
			if pos == nil || !IsPositionValid(float64(pos.GetLatitude()), float64(pos.GetLongitude())) {
				out = append(out, el("E026", "vehicle.position"))
			} else {
				if !IsBearingValid(true, pos.GetBearing()) {
					out = append(out, el("E027", "vehicle.position.bearing"))
				}
				if s := pos.GetSpeed(); s < 0 || s > 26 {
					out = append(out, el("W004", "vehicle.position.speed"))
				}
				lat := float64(pos.GetLatitude())
				lon := float64(pos.GetLongitude())
				if meta.ShapeBBoxBuffered != nil {
					if !meta.ShapeBBoxBuffered.Contains(lat, lon) {
						out = append(out, el("E028", "vehicle.position"))
					}
				} else {
					if !meta.StopBBoxBuffered.Contains(lat, lon) {
						out = append(out, el("E028", "vehicle.position"))
					}
				}
			}
		}
	}
	return out
}
