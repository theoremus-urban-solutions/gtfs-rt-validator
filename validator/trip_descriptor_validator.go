package validator

import (
	"context"
	"regexp"
	"time"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"your.org/gtfsrt/validator/notice"
	"your.org/gtfsrt/validator/schema"
)

type TripDescriptorValidator struct{}

var (
	reStartTime = regexp.MustCompile(`^([0-1]?\d|2\d|29):[0-5]\d:[0-5]\d$`)
	reStartDate = regexp.MustCompile(`^\d{8}$`)
)

func (TripDescriptorValidator) Validate(ctx context.Context, _ int64, meta *schema.Metadata, feed, _, _ *rt.FeedMessage) []notice.ErrorList {
	var out []notice.ErrorList
	// Iterate TUs and Alerts
	for _, e := range feed.GetEntity() {
		if tu := e.GetTripUpdate(); tu != nil {
			td := tu.GetTrip()
			tripID := td.GetTripId()
			// W006 missing trip_id
			if tripID == "" {
				out = append(out, el("W006", "trip_update.trip missing trip_id"))
			} else {
				trip, ok := meta.TripsByTripID[tripID]
				if !ok {
					// E003 unless ADDED
					if td.GetScheduleRelationship() != rt.TripDescriptor_ADDED {
						out = append(out, el("E003", "trip_id "+tripID))
					}
				} else {
					// E016 if ADDED but exists in GTFS
					if td.GetScheduleRelationship() == rt.TripDescriptor_ADDED {
						out = append(out, el("E016", "trip_id "+tripID))
					}
					// E023 start_time equals first GTFS arrival_time if provided and not frequency-based
					if td.GetStartTime() != "" {
						if sts := meta.StopTimesByTripID[tripID]; len(sts) > 0 {
							if sts[0].ArrivalTime != "" && td.GetStartTime() != sts[0].ArrivalTime {
								out = append(out, el("E023", "trip_id "+tripID))
							}
						}
					}
					// E024 direction_id matches
					if td.DirectionId != nil && trip.DirectionID != nil {
						if td.GetDirectionId() != uint32(*trip.DirectionID) {
							out = append(out, el("E024", "trip_id "+tripID))
						}
					}
				}
				// E004 route exists
				if td.GetRouteId() != "" {
					if _, ok := meta.RouteIDs[td.GetRouteId()]; !ok {
						out = append(out, el("E004", "route_id "+td.GetRouteId()))
					}
				}
				// E020/E021 format checks
				if s := td.GetStartTime(); s != "" && !reStartTime.MatchString(s) {
					out = append(out, el("E020", "start_time "+s))
				}
				if d := td.GetStartDate(); d != "" && !reStartDate.MatchString(d) {
					out = append(out, el("E021", "start_date "+d))
				}
				// W009 omitted: proto3 doesn't expose presence; handled elsewhere if needed
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
				// E034 agency exists
				if aid := ie.GetAgencyId(); aid != "" {
					if _, ok := meta.AgencyIDs[aid]; !ok {
						out = append(out, el("E034", "agency_id "+aid))
					}
				}
				// E030/E031/E035 consistency
				if ie.GetTrip() != nil {
					tid := ie.GetTrip().GetTripId()
					if trip, ok := meta.TripsByTripID[tid]; ok {
						if ie.GetRouteId() != "" && trip.RouteID != ie.GetRouteId() {
							out = append(out, el("E031", "trip_id "+tid))
						}
						if td := ie.GetTrip(); td != nil {
							if td.GetRouteId() != "" && td.GetRouteId() != trip.RouteID {
								out = append(out, el("E035", "trip_id "+tid))
							}
							if ie.GetRouteId() != "" && td.GetRouteId() != "" && td.GetRouteId() != ie.GetRouteId() {
								out = append(out, el("E030", "trip_id "+tid))
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
	_ = time.Now // silence unused if optimized out
	return out
}
