package validator

import (
	"context"
	"fmt"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type VehicleValidator struct {
	RTOnly bool // When true, skip checks that require static GTFS (E028 bbox)
}

func (v VehicleValidator) Validate(ctx context.Context, _ int64, meta *Metadata, feed, _, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	seen := make(map[string]struct{})

	for _, e := range feed.GetEntity() {
		eid := e.GetId()
		if tu := e.GetTripUpdate(); tu != nil {
			if tu.GetVehicle() == nil || tu.GetVehicle().GetId() == "" {
				tripID := tu.GetTrip().GetTripId()
				out = append(out, errorListDetailed("W002", "trip_update", eid, "", map[string]string{
					"trip_id": tripID,
				}))
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			vid := ""
			if vp.GetVehicle() != nil {
				vid = vp.GetVehicle().GetId()
			}
			if vid == "" {
				out = append(out, errorListDetailed("W002", "vehicle", eid, "", nil))
			} else {
				if _, dup := seen[vid]; dup {
					out = append(out, errorListDetailed("E052", "vehicle.id "+vid, eid, vid, map[string]string{
						"vehicle_id": vid,
					}))
				} else {
					seen[vid] = struct{}{}
				}
			}

			pos := vp.GetPosition()
			if pos == nil || !IsPositionValid(float64(pos.GetLatitude()), float64(pos.GetLongitude())) {
				var details map[string]string
				if pos != nil {
					details = map[string]string{
						"vehicle_id": vid,
						"latitude":   fmt.Sprintf("%f", pos.GetLatitude()),
						"longitude":  fmt.Sprintf("%f", pos.GetLongitude()),
					}
				} else {
					details = map[string]string{"vehicle_id": vid}
				}
				out = append(out, errorListDetailed("E026", "vehicle.position", eid, "", details))
			} else {
				lat := float64(pos.GetLatitude())
				lon := float64(pos.GetLongitude())
				if !IsBearingValid(true, pos.GetBearing()) {
					out = append(out, errorListDetailed("E027", "vehicle.position.bearing", eid, fmt.Sprintf("%.2f", pos.GetBearing()), map[string]string{
						"vehicle_id": vid,
						"latitude":   fmt.Sprintf("%f", lat),
						"longitude":  fmt.Sprintf("%f", lon),
					}))
				}
				if s := pos.GetSpeed(); s < 0 || s > 26 {
					out = append(out, errorListDetailed("W004", "vehicle.position.speed", eid, fmt.Sprintf("%.2f", s), map[string]string{
						"vehicle_id": vid,
						"latitude":   fmt.Sprintf("%f", lat),
						"longitude":  fmt.Sprintf("%f", lon),
					}))
				}
				// E028: Skip bbox check in RT-only mode (requires static GTFS)
				if !v.RTOnly && meta != nil {
					if meta.ShapeBBoxBuffered != nil {
						if !meta.ShapeBBoxBuffered.Contains(lat, lon) {
							out = append(out, errorListDetailed("E028", "vehicle.position", eid, "", map[string]string{
								"vehicle_id": vid,
								"latitude":   fmt.Sprintf("%f", lat),
								"longitude":  fmt.Sprintf("%f", lon),
							}))
						}
					} else {
						if !meta.StopBBoxBuffered.Contains(lat, lon) {
							out = append(out, errorListDetailed("E028", "vehicle.position", eid, "", map[string]string{
								"vehicle_id": vid,
								"latitude":   fmt.Sprintf("%f", lat),
								"longitude":  fmt.Sprintf("%f", lon),
							}))
						}
					}
				}
			}
		}
	}
	return out
}
