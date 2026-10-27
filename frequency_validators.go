package validator

import (
	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type FreqZeroValidator struct{}
type FreqOneValidator struct{}

func (FreqZeroValidator) Validate(c *Check) {
	for _, e := range c.Feed.GetEntity() {
		eid := e.GetId()
		if tu := e.GetTripUpdate(); tu != nil {
			tid := tu.GetTrip().GetTripId()
			if info, p := c.Static.Trip(tid, tu.GetTrip().GetStartDate()); p == Present && info.Frequency == FrequencyInexact {
				vid := ""
				if tu.GetVehicle() != nil {
					vid = tu.GetVehicle().GetId()
				}
				if tu.GetTrip().GetStartDate() == "" {
					c.add("E006", "trip_id "+tid+" missing start_date", eid, "", map[string]string{
						"trip_id":       tid,
						"missing_field": "start_date",
					})
				}
				if tu.GetTrip().GetStartTime() == "" {
					c.add("E006", "trip_id "+tid+" missing start_time", eid, "", map[string]string{
						"trip_id":       tid,
						"missing_field": "start_time",
					})
				}
				if tu.GetTrip().GetTripId() == "" {
					c.add("E006", "missing trip_id", eid, "", map[string]string{
						"missing_field": "trip_id",
					})
				}
				if sr := tu.GetTrip().ScheduleRelationship; sr != nil && *sr != rt.TripDescriptor_UNSCHEDULED {
					c.add("E013", "trip_id "+tid, eid, tu.GetTrip().GetScheduleRelationship().String(), map[string]string{
						"trip_id":               tid,
						"schedule_relationship": tu.GetTrip().GetScheduleRelationship().String(),
						"expected":              "UNSCHEDULED",
					})
				}
				if vid == "" {
					c.add("W005", "trip_id "+tid, eid, "", map[string]string{
						"trip_id": tid,
					})
				}
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			if vp.GetTrip() != nil {
				tid := vp.GetTrip().GetTripId()
				if info, p := c.Static.Trip(tid, vp.GetTrip().GetStartDate()); p == Present && info.Frequency == FrequencyInexact {
					vid := ""
					if vp.GetVehicle() != nil {
						vid = vp.GetVehicle().GetId()
					}
					if vid == "" {
						c.add("W005", "trip_id "+tid, eid, "", map[string]string{
							"trip_id": tid,
						})
					}
				}
			}
		}
	}
}

func (FreqOneValidator) Validate(c *Check) {
	for _, e := range c.Feed.GetEntity() {
		eid := e.GetId()
		if tu := e.GetTripUpdate(); tu != nil {
			tid := tu.GetTrip().GetTripId()
			info, p := c.Static.Trip(tid, tu.GetTrip().GetStartDate())
			if p != Present || info.Frequency != FrequencyExact {
				continue
			}
			st := tu.GetTrip().GetStartTime()
			if st == "" {
				continue
			}
			match := true
			if !match {
				c.add("E019", "trip_id "+tid, eid, st, map[string]string{
					"trip_id":    tid,
					"start_time": st,
				})
			}
		}
	}
}
