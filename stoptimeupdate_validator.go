package validator

import (
	"fmt"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type StopTimeUpdateValidator struct {
	RTOnly bool // When true, skip checks that require static GTFS (E009, E045, E046, E051)
}

func (v StopTimeUpdateValidator) Validate(c *Check) {
	static := c.Static
	if v.RTOnly {
		static = nil
	}
	for _, e := range c.Feed.GetEntity() {
		tu := e.GetTripUpdate()
		if tu == nil {
			continue
		}
		eid := e.GetId()
		tripID := tu.GetTrip().GetTripId()
		if tripID == "" {
			continue
		}

		var trip TripInfo
		tripStatic := static
		if static != nil {
			var p Presence
			if trip, p = static.Trip(tripID, tu.GetTrip().GetStartDate()); p == NotCovered {
				tripStatic = nil
			}
		}
		var hadE051 bool
		var prevSeq int = -1
		var prevStopID string
		var prevArr int64

		for _, stu := range tu.GetStopTimeUpdate() {
			stopID := stu.GetStopId()
			stopSeq := stu.GetStopSequence()
			stopSeqStr := fmt.Sprintf("%d", stopSeq)

			if stu.ScheduleRelationship == nil {
				c.add("W009", "trip_id "+tripID, eid, "", map[string]string{
					"trip_id":       tripID,
					"stop_id":       stopID,
					"stop_sequence": stopSeqStr,
				})
			}

			if stopID == "" && stopSeq == 0 {
				c.add("E040", "trip_id "+tripID, eid, "", map[string]string{
					"trip_id": tripID,
				})
			}

			if stu.GetScheduleRelationship() == rt.TripUpdate_StopTimeUpdate_NO_DATA {
				if stu.GetArrival() != nil || stu.GetDeparture() != nil {
					c.add("E042", "trip_id "+tripID, eid, "", map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
					})
				}
				continue
			}

			if stu.GetScheduleRelationship() != rt.TripUpdate_StopTimeUpdate_SKIPPED && stu.GetScheduleRelationship() != rt.TripUpdate_StopTimeUpdate_NO_DATA {
				if stu.GetArrival() == nil && stu.GetDeparture() == nil {
					c.add("E043", "trip_id "+tripID, eid, "", map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
					})
				}
			}

			if ev := stu.GetArrival(); ev != nil {
				if ev.GetDelay() == 0 && ev.GetTime() == 0 {
					c.add("E044", "trip_id "+tripID, eid, "", map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
						"event_type":    "arrival",
					})
				}
			}
			if ev := stu.GetDeparture(); ev != nil {
				if ev.GetDelay() == 0 && ev.GetTime() == 0 {
					c.add("E044", "trip_id "+tripID, eid, "", map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
						"event_type":    "departure",
					})
				}
			}

			if prevSeq >= 0 {
				if int(stopSeq) == prevSeq {
					c.add("E036", "trip_id "+tripID, eid, stopSeqStr, map[string]string{
						"trip_id":       tripID,
						"stop_sequence": stopSeqStr,
					})
				}
				if stopID != "" && prevStopID != "" && stopID == prevStopID {
					c.add("E037", "trip_id "+tripID, eid, stopID, map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
					})
				}
				if stopSeq != 0 && int(stopSeq) <= prevSeq {
					c.add("E002", "trip_id "+tripID, eid, stopSeqStr, map[string]string{
						"trip_id":                tripID,
						"stop_sequence":          stopSeqStr,
						"previous_stop_sequence": fmt.Sprintf("%d", prevSeq),
					})
				}
			}
			if stopSeq != 0 {
				prevSeq = int(stopSeq)
			}
			if stopID != "" {
				prevStopID = stopID
			}

			// E045, E051: Skip in RT-only mode (requires static GTFS stop_times)
			if tripStatic != nil && stopSeq != 0 && stopID != "" {
				if st, ok := findSeq(trip, stopSeq); ok {
					if expectedStopID := st.StopID; expectedStopID != stopID {
						c.add("E045", "trip_id "+tripID, eid, stopID, map[string]string{
							"trip_id":       tripID,
							"stop_sequence": stopSeqStr,
							"expected":      expectedStopID,
							"actual":        stopID,
						})
					}
				} else {
					c.add("E051", "trip_id "+tripID, eid, stopSeqStr, map[string]string{
						"trip_id":       tripID,
						"stop_sequence": stopSeqStr,
					})
					hadE051 = true
				}
			}
			if hadE051 {
				break
			}

			// E046: Skip in RT-only mode (requires static GTFS stop_times)
			if tripStatic != nil {
				if (stu.GetArrival() == nil || stu.GetArrival().GetTime() == 0) && (stu.GetDeparture() == nil || stu.GetDeparture().GetTime() == 0) {
					if stopSeq > 0 {
						if st, ok := findSeq(trip, stopSeq); ok && !st.ArrivalSet && !st.DepartureSet {
							c.add("E046", "trip_id "+tripID, eid, "", map[string]string{
								"trip_id":       tripID,
								"stop_id":       stopID,
								"stop_sequence": stopSeqStr,
							})
						}
					}
				}
			}

			// E009: Skip in RT-only mode (requires static GTFS trips with multi-stops)
			if tripStatic != nil {
				if trip.RepeatsStop {
					if stopSeq == 0 {
						c.add("E009", "trip_id "+tripID, eid, "", map[string]string{
							"trip_id": tripID,
							"stop_id": stopID,
						})
					}
				}
			}

			var thisTime int64
			if stu.GetArrival() != nil && stu.GetArrival().GetTime() != 0 {
				thisTime = stu.GetArrival().GetTime()
			} else if stu.GetDeparture() != nil && stu.GetDeparture().GetTime() != 0 {
				thisTime = stu.GetDeparture().GetTime()
			}
			if thisTime != 0 {
				if prevArr != 0 && thisTime <= prevArr {
					c.add("E022", "trip_id "+tripID, eid, fmt.Sprintf("%d", thisTime), map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
						"current_time":  fmt.Sprintf("%d", thisTime),
						"previous_time": fmt.Sprintf("%d", prevArr),
					})
				}
				prevArr = thisTime
			}
			if stu.GetArrival() != nil && stu.GetDeparture() != nil && stu.GetArrival().GetTime() != 0 && stu.GetDeparture().GetTime() != 0 {
				if stu.GetDeparture().GetTime() < stu.GetArrival().GetTime() {
					c.add("E025", "trip_id "+tripID, eid, "", map[string]string{
						"trip_id":        tripID,
						"stop_id":        stopID,
						"stop_sequence":  stopSeqStr,
						"arrival_time":   fmt.Sprintf("%d", stu.GetArrival().GetTime()),
						"departure_time": fmt.Sprintf("%d", stu.GetDeparture().GetTime()),
					})
				}
			}
		}

		if len(tu.GetStopTimeUpdate()) == 0 && tu.GetTrip().GetScheduleRelationship() != rt.TripDescriptor_CANCELED {
			c.add("E041", "trip_id "+tripID, eid, "", map[string]string{
				"trip_id": tripID,
			})
		}
	}
}

// findSeq returns the GTFS stop of trip at stop_sequence seq.
func findSeq(trip TripInfo, seq uint32) (ScheduledStop, bool) {
	for i, n := 0, trip.NumStops(); i < n; i++ {
		if st := trip.StopAt(i); st.Sequence == seq {
			return st, true
		}
	}
	return ScheduledStop{}, false
}
