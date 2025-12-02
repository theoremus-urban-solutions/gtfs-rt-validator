package validator

import (
	"context"
	"regexp"
	"time"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type TripDescriptorValidator struct{}

var (
	reStartTime = regexp.MustCompile(`^([0-1]?\d|2\d|29):[0-5]\d:[0-5]\d$`)
	reStartDate = regexp.MustCompile(`^\d{8}$`)
)

func (TripDescriptorValidator) Validate(ctx context.Context, _ int64, meta *Metadata, feed, _, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	for _, e := range feed.GetEntity() {
		if tu := e.GetTripUpdate(); tu != nil {
			td := tu.GetTrip()
			tripID := td.GetTripId()
			if tripID == "" {
				out = append(out, el("W006", "trip_update.trip missing trip_id"))
			} else {
				trip, ok := meta.TripsByTripID[tripID]
				if !ok {
					if td.GetScheduleRelationship() != rt.TripDescriptor_ADDED {
						out = append(out, el("W015", "trip_id "+tripID))
					}
				} else {
					if td.GetScheduleRelationship() == rt.TripDescriptor_ADDED {
						out = append(out, el("W021", "trip_id "+tripID))
					}
					if td.GetStartTime() != "" {
						if sts := meta.StopTimesByTripID[tripID]; len(sts) > 0 {
							if sts[0].ArrivalTime != "" && td.GetStartTime() != sts[0].ArrivalTime {
								out = append(out, el("W022", "trip_id "+tripID))
							}
						}
					}
					if td.DirectionId != nil && trip.DirectionID != nil {
						if td.GetDirectionId() != uint32(*trip.DirectionID) {
							out = append(out, el("W023", "trip_id "+tripID))
						}
					}
				}
				if td.GetRouteId() != "" {
					if _, ok := meta.RouteIDs[td.GetRouteId()]; !ok {
						out = append(out, el("W016", "route_id "+td.GetRouteId()))
					}
				}
				if s := td.GetStartTime(); s != "" && !reStartTime.MatchString(s) {
					out = append(out, el("E020", "start_time "+s))
				}
				if d := td.GetStartDate(); d != "" && !reStartDate.MatchString(d) {
					out = append(out, el("E021", "start_date "+d))
				}
			}
		}
		if al := e.GetAlert(); al != nil {
			ies := al.GetInformedEntity()
			if len(ies) == 0 {
				out = append(out, el("E032", "alert"))
				continue
			}
			anySpecified := false
			for _, ie := range ies {
				if ie.GetRouteId() != "" || (ie.Trip != nil && ie.Trip.GetTripId() != "") || ie.GetStopId() != "" || ie.GetAgencyId() != "" {
					anySpecified = true
				}
				if aid := ie.GetAgencyId(); aid != "" {
					if _, ok := meta.AgencyIDs[aid]; !ok {
						out = append(out, el("W027", "agency_id "+aid))
					}
				}
				if ie.GetTrip() != nil {
					tid := ie.GetTrip().GetTripId()
					if trip, ok := meta.TripsByTripID[tid]; ok {
						if ie.GetRouteId() != "" && trip.RouteID != ie.GetRouteId() {
							out = append(out, el("W026", "trip_id "+tid))
						}
						if td := ie.GetTrip(); td != nil {
							if td.GetRouteId() != "" && td.GetRouteId() != trip.RouteID {
								out = append(out, el("W028", "trip_id "+tid))
							}
							if ie.GetRouteId() != "" && td.GetRouteId() != "" && td.GetRouteId() != ie.GetRouteId() {
								out = append(out, el("W025", "trip_id "+tid))
							}
						}
					}
				}
			}
			if !anySpecified {
				out = append(out, el("E033", "alert.informed_entity"))
			}
		}
	}
	_ = time.Now
	return out
}
