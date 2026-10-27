package validator

import (
	"fmt"
)

type VehicleValidator struct {
	RTOnly bool // When true, skip checks that require static GTFS (E028 bbox)
}

func (v VehicleValidator) Validate(c *Check) {
	feed := c.Feed
	var seen map[string]struct{}

	for _, e := range feed.GetEntity() {
		eid := e.GetId()
		if tu := e.GetTripUpdate(); tu != nil {
			if tu.GetVehicle() == nil || tu.GetVehicle().GetId() == "" {
				tripID := tu.GetTrip().GetTripId()
				c.add("W002", "trip_update", eid, "", map[string]string{
					"trip_id": tripID,
				})
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			vid := ""
			if vp.GetVehicle() != nil {
				vid = vp.GetVehicle().GetId()
			}
			if vid == "" {
				c.add("W002", "vehicle", eid, "", nil)
			} else {
				if _, dup := seen[vid]; dup {
					c.add("E052", "vehicle.id "+vid, eid, vid, map[string]string{
						"vehicle_id": vid,
					})
				} else {
					if seen == nil {
						seen = make(map[string]struct{})
					}
					seen[vid] = struct{}{}
				}
			}

			pos := vp.GetPosition()
			// position is optional, so E026 judges only a position that was sent.
			if pos != nil && !IsPositionValid(float64(pos.GetLatitude()), float64(pos.GetLongitude())) {
				c.add("E026", "vehicle.position", eid, "", map[string]string{
					"vehicle_id": vid,
					"latitude":   fmt.Sprintf("%f", pos.GetLatitude()),
					"longitude":  fmt.Sprintf("%f", pos.GetLongitude()),
				})
			} else if pos != nil {
				lat := float64(pos.GetLatitude())
				lon := float64(pos.GetLongitude())
				if !IsBearingValid(true, pos.GetBearing()) {
					c.add("E027", "vehicle.position.bearing", eid, fmt.Sprintf("%.2f", pos.GetBearing()), map[string]string{
						"vehicle_id": vid,
						"latitude":   fmt.Sprintf("%f", lat),
						"longitude":  fmt.Sprintf("%f", lon),
					})
				}
				if s := pos.GetSpeed(); s < 0 || s > 26 {
					c.add("W004", "vehicle.position.speed", eid, fmt.Sprintf("%.2f", s), map[string]string{
						"vehicle_id": vid,
						"latitude":   fmt.Sprintf("%f", lat),
						"longitude":  fmt.Sprintf("%f", lon),
					})
				}
				// E028: Skip bbox check in RT-only mode (requires static GTFS)
				if !v.RTOnly && c.Static != nil {
					if area, ok := c.Static.ServiceArea(); ok && !area.Contains(lat, lon) {
						c.add("E028", "vehicle.position", eid, "", map[string]string{
							"vehicle_id": vid,
							"latitude":   fmt.Sprintf("%f", lat),
							"longitude":  fmt.Sprintf("%f", lon),
						})
					}
				}
			}
		}
	}
}
