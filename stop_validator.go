package validator

import (
	"context"
	"fmt"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type StopValidator struct{}

func (StopValidator) Validate(ctx context.Context, _ int64, meta *Metadata, feed, _, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	for _, e := range feed.GetEntity() {
		eid := e.GetId()
		if tu := e.GetTripUpdate(); tu != nil {
			tripID := tu.GetTrip().GetTripId()
			for _, stu := range tu.GetStopTimeUpdate() {
				sid := stu.GetStopId()
				if sid != "" {
					if _, ok := meta.StopIDs[sid]; !ok {
						out = append(out, errorListDetailed("E011", "stop_id "+sid, eid, sid, map[string]string{
							"stop_id":       sid,
							"trip_id":       tripID,
							"stop_sequence": fmt.Sprintf("%d", stu.GetStopSequence()),
						}))
					}
					if lt, ok := meta.StopToLocationType[sid]; ok && lt != 0 {
						out = append(out, errorListDetailed("E015", "stop_id "+sid, eid, sid, map[string]string{
							"stop_id":       sid,
							"trip_id":       tripID,
							"location_type": fmt.Sprintf("%d", lt),
						}))
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
				if _, ok := meta.StopIDs[sid]; !ok {
					out = append(out, errorListDetailed("E011", "stop_id "+sid, eid, sid, map[string]string{
						"stop_id":    sid,
						"vehicle_id": vid,
					}))
				}
				if lt, ok := meta.StopToLocationType[sid]; ok && lt != 0 {
					out = append(out, errorListDetailed("E015", "stop_id "+sid, eid, sid, map[string]string{
						"stop_id":       sid,
						"vehicle_id":    vid,
						"location_type": fmt.Sprintf("%d", lt),
					}))
				}
			}
		}
	}
	return out
}
