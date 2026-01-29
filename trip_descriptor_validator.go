package validator

import (
	"context"
	"fmt"
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
		eid := e.GetId()
		if tu := e.GetTripUpdate(); tu != nil {
			td := tu.GetTrip()
			tripID := td.GetTripId()
			routeID := td.GetRouteId()
			if td != nil && td.ScheduleRelationship == nil {
				out = append(out, errorListDetailed("W009", "trip_update.trip", eid, "", map[string]string{
					"trip_id": tripID,
				}))
			}
			if tripID == "" {
				out = append(out, errorListDetailed("W006", "trip_update.trip missing trip_id", eid, "", map[string]string{
					"route_id": routeID,
				}))
			} else {
				trip, ok := meta.TripsByTripID[tripID]
				if !ok {
					if td.GetScheduleRelationship() != rt.TripDescriptor_ADDED {
						out = append(out, errorListDetailed("E003", "trip_id "+tripID, eid, tripID, map[string]string{
							"trip_id": tripID,
						}))
					}
				} else {
					if td.GetScheduleRelationship() == rt.TripDescriptor_ADDED {
						out = append(out, errorListDetailed("E016", "trip_id "+tripID, eid, tripID, map[string]string{
							"trip_id":               tripID,
							"schedule_relationship": "ADDED",
						}))
					}
					if td.GetStartTime() != "" {
						if sts := meta.StopTimesByTripID[tripID]; len(sts) > 0 {
							if sts[0].ArrivalTime != "" && td.GetStartTime() != sts[0].ArrivalTime {
								out = append(out, errorListDetailed("E023", "trip_id "+tripID, eid, td.GetStartTime(), map[string]string{
									"trip_id":  tripID,
									"expected": sts[0].ArrivalTime,
									"actual":   td.GetStartTime(),
								}))
							}
						}
					}
					if td.DirectionId != nil && trip.DirectionID != nil {
						if td.GetDirectionId() != uint32(*trip.DirectionID) {
							out = append(out, errorListDetailed("E024", "trip_id "+tripID, eid, fmt.Sprintf("%d", td.GetDirectionId()), map[string]string{
								"trip_id":  tripID,
								"expected": fmt.Sprintf("%d", *trip.DirectionID),
								"actual":   fmt.Sprintf("%d", td.GetDirectionId()),
							}))
						}
					}
				}
				if routeID != "" {
					if _, ok := meta.RouteIDs[routeID]; !ok {
						out = append(out, errorListDetailed("E004", "route_id "+routeID, eid, routeID, map[string]string{
							"trip_id":  tripID,
							"route_id": routeID,
						}))
					}
				}
				if s := td.GetStartTime(); s != "" && !reStartTime.MatchString(s) {
					out = append(out, errorListDetailed("E020", "start_time "+s, eid, s, map[string]string{
						"trip_id":    tripID,
						"start_time": s,
					}))
				}
				if d := td.GetStartDate(); d != "" && !reStartDate.MatchString(d) {
					out = append(out, errorListDetailed("E021", "start_date "+d, eid, d, map[string]string{
						"trip_id":    tripID,
						"start_date": d,
					}))
				}
			}
		}
		if al := e.GetAlert(); al != nil {
			ies := al.GetInformedEntity()
			if len(ies) == 0 {
				out = append(out, errorListDetailed("E032", "alert", eid, "", nil))
				continue
			}
			anySpecified := false
			for _, ie := range ies {
				if ie.GetRouteId() != "" || (ie.Trip != nil && ie.Trip.GetTripId() != "") || ie.GetStopId() != "" || ie.GetAgencyId() != "" {
					anySpecified = true
				}
				if aid := ie.GetAgencyId(); aid != "" {
					if _, ok := meta.AgencyIDs[aid]; !ok {
						out = append(out, errorListDetailed("E034", "agency_id "+aid, eid, aid, map[string]string{
							"agency_id": aid,
						}))
					}
				}
				if ie.GetTrip() != nil {
					tid := ie.GetTrip().GetTripId()
					if trip, ok := meta.TripsByTripID[tid]; ok {
						if ie.GetRouteId() != "" && trip.RouteID != ie.GetRouteId() {
							out = append(out, errorListDetailed("E031", "trip_id "+tid, eid, ie.GetRouteId(), map[string]string{
								"trip_id":                  tid,
								"informed_entity_route_id": ie.GetRouteId(),
								"trip_route_id":            trip.RouteID,
							}))
						}
						if td := ie.GetTrip(); td != nil {
							if td.GetRouteId() != "" && td.GetRouteId() != trip.RouteID {
								out = append(out, errorListDetailed("E035", "trip_id "+tid, eid, td.GetRouteId(), map[string]string{
									"trip_id":                tid,
									"trip_descriptor_route":  td.GetRouteId(),
									"expected_trip_route_id": trip.RouteID,
								}))
							}
							if ie.GetRouteId() != "" && td.GetRouteId() != "" && td.GetRouteId() != ie.GetRouteId() {
								out = append(out, errorListDetailed("E030", "trip_id "+tid, eid, "", map[string]string{
									"trip_id":                  tid,
									"trip_descriptor_route":    td.GetRouteId(),
									"informed_entity_route_id": ie.GetRouteId(),
								}))
							}
						}
					}
				}
			}
			if !anySpecified {
				out = append(out, errorListDetailed("E033", "alert.informed_entity", eid, "", nil))
			}
		}
	}
	_ = time.Now
	return out
}
