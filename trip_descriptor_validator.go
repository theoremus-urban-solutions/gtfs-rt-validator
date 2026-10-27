package validator

import (
	"fmt"
	"regexp"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type TripDescriptorValidator struct{}

var (
	reStartTime = regexp.MustCompile(`^([0-1]?\d|2\d|29):[0-5]\d:[0-5]\d$`)
	reStartDate = regexp.MustCompile(`^\d{8}$`)
)

func (TripDescriptorValidator) Validate(c *Check) {
	for _, e := range c.Feed.GetEntity() {
		eid := e.GetId()
		if tu := e.GetTripUpdate(); tu != nil {
			td := tu.GetTrip()
			tripID := td.GetTripId()
			routeID := td.GetRouteId()
			if td != nil && td.ScheduleRelationship == nil {
				c.add("W009", "trip_update.trip", eid, "", map[string]string{
					"trip_id": tripID,
				})
			}
			if tripID == "" {
				c.add("W006", "trip_update.trip missing trip_id", eid, "", map[string]string{
					"route_id": routeID,
				})
			} else {
				trip, p := c.Static.Trip(tripID, td.GetStartDate())
				switch p {
				case Absent:
					if td.GetScheduleRelationship() != rt.TripDescriptor_ADDED {
						c.add("E003", "trip_id "+tripID, eid, tripID, map[string]string{
							"trip_id": tripID,
						})
					}
				case Present:
					if td.GetScheduleRelationship() == rt.TripDescriptor_ADDED {
						c.add("E016", "trip_id "+tripID, eid, tripID, map[string]string{
							"trip_id":               tripID,
							"schedule_relationship": "ADDED",
						})
					}
					if s := td.GetStartTime(); s != "" && trip.FirstArrival >= 0 {
						if t, ok := parseClock(s); !ok || t != trip.FirstArrival {
							c.add("E023", "trip_id "+tripID, eid, s, map[string]string{
								"trip_id":  tripID,
								"expected": formatClock(trip.FirstArrival),
								"actual":   s,
							})
						}
					}
					if td.DirectionId != nil && trip.DirectionID >= 0 {
						if td.GetDirectionId() != uint32(trip.DirectionID) {
							c.add("E024", "trip_id "+tripID, eid, fmt.Sprintf("%d", td.GetDirectionId()), map[string]string{
								"trip_id":  tripID,
								"expected": fmt.Sprintf("%d", trip.DirectionID),
								"actual":   fmt.Sprintf("%d", td.GetDirectionId()),
							})
						}
					}
				}
				if routeID != "" && c.Static.Route(routeID) == Absent {
					c.add("E004", "route_id "+routeID, eid, routeID, map[string]string{
						"trip_id":  tripID,
						"route_id": routeID,
					})
				}
				if s := td.GetStartTime(); s != "" && !reStartTime.MatchString(s) {
					c.add("E020", "start_time "+s, eid, s, map[string]string{
						"trip_id":    tripID,
						"start_time": s,
					})
				}
				if d := td.GetStartDate(); d != "" && !reStartDate.MatchString(d) {
					c.add("E021", "start_date "+d, eid, d, map[string]string{
						"trip_id":    tripID,
						"start_date": d,
					})
				}
			}
		}
		if al := e.GetAlert(); al != nil {
			ies := al.GetInformedEntity()
			if len(ies) == 0 {
				c.add("E032", "alert", eid, "", nil)
				continue
			}
			for _, ie := range ies {
				if !selectsSomething(ie) {
					c.add("E033", "alert.informed_entity", eid, "", nil)
				}
				if aid := ie.GetAgencyId(); aid != "" && c.Static.Agency(aid) == Absent {
					c.add("E034", "agency_id "+aid, eid, aid, map[string]string{
						"agency_id": aid,
					})
				}
				if td := ie.GetTrip(); td != nil {
					tid := td.GetTripId()
					if trip, p := c.Static.Trip(tid, td.GetStartDate()); p == Present {
						if ie.GetRouteId() != "" && trip.RouteID != ie.GetRouteId() {
							c.add("E031", "trip_id "+tid, eid, ie.GetRouteId(), map[string]string{
								"trip_id":                  tid,
								"informed_entity_route_id": ie.GetRouteId(),
								"trip_route_id":            trip.RouteID,
							})
						}
						if td.GetRouteId() != "" && td.GetRouteId() != trip.RouteID {
							c.add("E035", "trip_id "+tid, eid, td.GetRouteId(), map[string]string{
								"trip_id":                tid,
								"trip_descriptor_route":  td.GetRouteId(),
								"expected_trip_route_id": trip.RouteID,
							})
						}
						if ie.GetRouteId() != "" && td.GetRouteId() != "" && td.GetRouteId() != ie.GetRouteId() {
							c.add("E030", "trip_id "+tid, eid, "", map[string]string{
								"trip_id":                  tid,
								"trip_descriptor_route":    td.GetRouteId(),
								"informed_entity_route_id": ie.GetRouteId(),
							})
						}
					}
				}
			}
		}
	}
}

// selectsSomething reports whether a selector names an agency, route, route type, stop, trip or trip route.
func selectsSomething(ie *rt.EntitySelector) bool {
	if ie.AgencyId != nil || ie.RouteId != nil || ie.RouteType != nil || ie.StopId != nil {
		return true
	}
	td := ie.GetTrip()
	return td != nil && (td.TripId != nil || td.RouteId != nil)
}
