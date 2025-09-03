package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"your.org/gtfsrt/validator/notice"
	"your.org/gtfsrt/validator/schema"
	"your.org/gtfsrt/validator/util"
)

type VehicleValidator struct{}

func (VehicleValidator) Validate(ctx context.Context, _ int64, meta *schema.Metadata, feed, _, _ *rt.FeedMessage) []notice.ErrorList {
	var out []notice.ErrorList
	seen := make(map[string]struct{})

	for _, e := range feed.GetEntity() {
		if tu := e.GetTripUpdate(); tu != nil {
			// W002 for TripUpdate vehicle descriptor
			if tu.GetVehicle() == nil || tu.GetVehicle().GetId() == "" {
				out = append(out, el("W002", "trip_update"))
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			// W002 and E052
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

			// Position checks
			pos := vp.GetPosition()
			if pos == nil || !util.IsPositionValid(float64(pos.GetLatitude()), float64(pos.GetLongitude())) {
				out = append(out, el("E026", "vehicle.position"))
			} else {
				// E027 bearing
				if !util.IsBearingValid(true, pos.GetBearing()) {
					out = append(out, el("E027", "vehicle.position.bearing"))
				}
				// W004 unrealistic speed
				if s := pos.GetSpeed(); s < 0 || s > 26 {
					out = append(out, el("W004", "vehicle.position.speed"))
				}
				// E028 coverage: within shape bbox with buffer if present else stop bbox with buffer
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
				// E029 far from trip shape: deferred until trip buffered shapes are implemented
			}
		}
	}
	return out
}
