package validator

import (
	"fmt"
	"strconv"

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
			if trip, p = static.Trip(tripID, tu.GetTrip().GetStartDate()); p != Present {
				tripStatic = nil
			}
		}
		stus := tu.GetStopTimeUpdate()
		// gi walks the GTFS stops once, in step with the updates, as canonical does.
		gi, nStops := 0, trip.NumStops()
		var (
			e009, w009, ordered, fromStopID = false, false, true, false
			nSeqs                           int
			lastSeq, prevSeq                uint32
			prevStopID                      string
		)
		addSeq := func(seq uint32) {
			if nSeqs > 0 && seq <= lastSeq {
				ordered = false
			}
			lastSeq = seq
			nSeqs++
		}

		for i, stu := range stus {
			stopID := stu.GetStopId()
			stopSeq := stu.GetStopSequence()
			hasSeq, hasStop := stu.StopSequence != nil, stu.StopId != nil

			if !w009 && stu.ScheduleRelationship == nil {
				c.add("W009", "trip_id "+tripID, eid, "", map[string]string{
					"trip_id":       tripID,
					"stop_id":       stopID,
					"stop_sequence": u32s(stopSeq),
				})
				w009 = true
			}

			if !e009 && tripStatic != nil && trip.RepeatsStop && !hasSeq {
				c.add("E009", "trip_id "+tripID, eid, "", map[string]string{
					"trip_id": tripID,
					"stop_id": stopID,
				})
				e009 = true
			}
			if i > 0 {
				if hasSeq && stopSeq == prevSeq {
					c.add("E036", "trip_id "+tripID, eid, u32s(stopSeq), map[string]string{
						"trip_id":       tripID,
						"stop_sequence": u32s(stopSeq),
					})
				}
				if prevStopID != "" && hasStop && stopID == prevStopID {
					c.add("E037", "trip_id "+tripID, eid, stopID, map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": u32s(stopSeq),
					})
				}
			}
			prevSeq, prevStopID = stopSeq, stopID
			if hasSeq {
				addSeq(stopSeq)
			}

			unknownSeq := false
			if tripStatic != nil {
				for gi < nStops {
					st := trip.StopAt(gi)
					foundSeq := hasSeq && st.Sequence == stopSeq
					if foundSeq {
						if hasStop && st.StopID != stopID {
							c.add("E045", "trip_id "+tripID, eid, stopID, map[string]string{
								"trip_id":       tripID,
								"stop_sequence": u32s(stopSeq),
								"expected":      st.StopID,
								"actual":        stopID,
							})
						}
						checkE046(c, eid, tripID, stu, st)
					}
					foundStop := hasStop && st.StopID == stopID
					gi++
					if foundSeq {
						break
					}
					if hasSeq && gi == nStops {
						unknownSeq = true
					}
					if !hasSeq && foundStop {
						addSeq(st.Sequence)
						fromStopID = true
						checkE046(c, eid, tripID, stu, st)
						break
					}
				}
			}

			if !hasSeq && !hasStop {
				c.add("E040", "trip_id "+tripID, eid, "", map[string]string{
					"trip_id": tripID,
				})
			}
			sr := stu.GetScheduleRelationship()
			if sr == rt.TripUpdate_StopTimeUpdate_NO_DATA {
				if stu.Arrival != nil || stu.Departure != nil {
					c.add("E042", "trip_id "+tripID, eid, "", map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": u32s(stopSeq),
					})
				}
			} else if sr != rt.TripUpdate_StopTimeUpdate_SKIPPED && stu.Arrival == nil && stu.Departure == nil {
				c.add("E043", "trip_id "+tripID, eid, "", map[string]string{
					"trip_id":       tripID,
					"stop_id":       stopID,
					"stop_sequence": u32s(stopSeq),
				})
			}
			// SKIPPED stops need no delay or time.
			if sr != rt.TripUpdate_StopTimeUpdate_SKIPPED {
				for _, ev := range [2]struct {
					e    *rt.TripUpdate_StopTimeEvent
					name string
				}{{stu.Arrival, "arrival"}, {stu.Departure, "departure"}} {
					if ev.e != nil && ev.e.Delay == nil && ev.e.Time == nil {
						c.add("E044", "trip_id "+tripID, eid, "", map[string]string{
							"trip_id":       tripID,
							"stop_id":       stopID,
							"stop_sequence": u32s(stopSeq),
							"event_type":    ev.name,
						})
					}
				}
			}

			if unknownSeq {
				c.add("E051", "trip_id "+tripID, eid, u32s(stopSeq), map[string]string{
					"trip_id":       tripID,
					"stop_sequence": u32s(stopSeq),
				})
				break
			}
		}

		if !ordered || (fromStopID && nSeqs < len(stus)) {
			c.add("E002", "trip_id "+tripID, eid, "", map[string]string{
				"trip_id": tripID,
			})
		}

		checkStopTimes(c, eid, tripID, tu.GetStopTimeUpdate())

		if len(tu.GetStopTimeUpdate()) == 0 && tu.GetTrip().GetScheduleRelationship() != rt.TripDescriptor_CANCELED {
			c.add("E041", "trip_id "+tripID, eid, "", map[string]string{
				"trip_id": tripID,
			})
		}
	}
}

// checkStopTimes raises E022 and E025 in a pass of its own, as canonical does, so no break in the sequence checks can skip them.
func checkStopTimes(c *Check, eid, tripID string, stus []*rt.TripUpdate_StopTimeUpdate) {
	var prevArr int64
	for _, stu := range stus {
		if stu.GetScheduleRelationship() == rt.TripUpdate_StopTimeUpdate_NO_DATA {
			continue
		}
		stopID := stu.GetStopId()
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
					"stop_sequence": strconv.FormatUint(uint64(stu.GetStopSequence()), 10),
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
					"stop_sequence":  strconv.FormatUint(uint64(stu.GetStopSequence()), 10),
					"arrival_time":   fmt.Sprintf("%d", stu.GetArrival().GetTime()),
					"departure_time": fmt.Sprintf("%d", stu.GetDeparture().GetTime()),
				})
			}
		}
	}
}

// u32s formats a stop_sequence for a notice.
func u32s(x uint32) string { return strconv.FormatUint(uint64(x), 10) }

// checkE046 raises E046 for each sent arrival or departure that has no time where GTFS has none either.
func checkE046(c *Check, eid, tripID string, stu *rt.TripUpdate_StopTimeUpdate, st ScheduledStop) {
	if ev := stu.Arrival; ev != nil && ev.Time == nil && !st.ArrivalSet {
		c.add("E046", "trip_id "+tripID, eid, "arrival", map[string]string{
			"trip_id":       tripID,
			"stop_id":       st.StopID,
			"stop_sequence": strconv.FormatUint(uint64(st.Sequence), 10),
			"event":         "arrival",
		})
	}
	if ev := stu.Departure; ev != nil && ev.Time == nil && !st.DepartureSet {
		c.add("E046", "trip_id "+tripID, eid, "departure", map[string]string{
			"trip_id":       tripID,
			"stop_id":       st.StopID,
			"stop_sequence": strconv.FormatUint(uint64(st.Sequence), 10),
			"event":         "departure",
		})
	}
}
