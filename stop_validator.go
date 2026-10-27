package validator

import (
	"fmt"
)

type StopValidator struct{}

func (StopValidator) Validate(c *Check) {
	for _, e := range c.Feed.GetEntity() {
		eid := e.GetId()
		if tu := e.GetTripUpdate(); tu != nil {
			tripID := tu.GetTrip().GetTripId()
			for _, stu := range tu.GetStopTimeUpdate() {
				sid := stu.GetStopId()
				if sid != "" {
					lt, p := c.Static.Stop(sid, tu.GetTrip().GetStartDate())
					if p == Absent {
						c.add("E011", "stop_id "+sid, eid, sid, map[string]string{
							"stop_id":       sid,
							"trip_id":       tripID,
							"stop_sequence": fmt.Sprintf("%d", stu.GetStopSequence()),
						})
					}
					if p == Present && lt != 0 {
						c.add("E015", "stop_id "+sid, eid, sid, map[string]string{
							"stop_id":       sid,
							"trip_id":       tripID,
							"location_type": fmt.Sprintf("%d", lt),
						})
					}
				}
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			sid := vp.GetStopId()
			vid := ""
			if vp.GetVehicle() != nil {
				vid = vp.GetVehicle().GetId()
			}
			if sid != "" {
				lt, p := c.Static.Stop(sid, vp.GetTrip().GetStartDate())
				if p == Absent {
					c.add("E011", "stop_id "+sid, eid, sid, map[string]string{
						"stop_id":    sid,
						"vehicle_id": vid,
					})
				}
				if p == Present && lt != 0 {
					c.add("E015", "stop_id "+sid, eid, sid, map[string]string{
						"stop_id":       sid,
						"vehicle_id":    vid,
						"location_type": fmt.Sprintf("%d", lt),
					})
				}
			}
		}
	}
}
