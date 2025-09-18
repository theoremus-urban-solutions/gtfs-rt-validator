package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type StopValidator struct{}

func (StopValidator) Validate(ctx context.Context, _ int64, meta *Metadata, feed, _, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	for _, e := range feed.GetEntity() {
		if tu := e.GetTripUpdate(); tu != nil {
			for _, stu := range tu.GetStopTimeUpdate() {
				sid := stu.GetStopId()
				if sid != "" {
					if _, ok := meta.StopIDs[sid]; !ok {
						out = append(out, el("E011", "stop_id "+sid))
					}
					if lt, ok := meta.StopToLocationType[sid]; ok && lt != 0 {
						out = append(out, el("E015", "stop_id "+sid))
					}
				}
			}
		}
		if vp := e.GetVehicle(); vp != nil {
			sid := vp.GetStopId()
			if sid != "" {
				if _, ok := meta.StopIDs[sid]; !ok {
					out = append(out, el("E011", "stop_id "+sid))
				}
				if lt, ok := meta.StopToLocationType[sid]; ok && lt != 0 {
					out = append(out, el("E015", "stop_id "+sid))
				}
			}
		}
	}
	return out
}
