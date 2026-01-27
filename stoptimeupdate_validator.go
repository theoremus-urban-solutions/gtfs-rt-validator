package validator

import (
	"context"
	"fmt"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type StopTimeUpdateValidator struct {
	RTOnly bool // When true, skip checks that require static GTFS (W017, W031, W032, W034)
}

func (v StopTimeUpdateValidator) Validate(ctx context.Context, _ int64, meta *Metadata, feed, _, _ *rt.FeedMessage) []ErrorList {
	var out []ErrorList
	seqMapCache := make(map[string]map[int]string)

	for _, e := range feed.GetEntity() {
		tu := e.GetTripUpdate()
		if tu == nil {
			continue
		}
		eid := e.GetId()
		tripID := tu.GetTrip().GetTripId()
		if tripID == "" {
			continue
		}

		var hadE051 bool
		var prevSeq int = -1
		var prevStopID string
		var prevArr int64

		for _, stu := range tu.GetStopTimeUpdate() {
			stopID := stu.GetStopId()
			stopSeq := stu.GetStopSequence()
			stopSeqStr := fmt.Sprintf("%d", stopSeq)

			if stopID == "" && stopSeq == 0 {
				out = append(out, errorListDetailed("E040", "trip_id "+tripID, eid, "", map[string]string{
					"trip_id": tripID,
				}))
			}

			if stu.GetScheduleRelationship() == rt.TripUpdate_StopTimeUpdate_NO_DATA {
				if stu.GetArrival() != nil || stu.GetDeparture() != nil {
					out = append(out, errorListDetailed("E042", "trip_id "+tripID, eid, "", map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
					}))
				}
				continue
			}

			if stu.GetScheduleRelationship() != rt.TripUpdate_StopTimeUpdate_SKIPPED && stu.GetScheduleRelationship() != rt.TripUpdate_StopTimeUpdate_NO_DATA {
				if stu.GetArrival() == nil && stu.GetDeparture() == nil {
					out = append(out, errorListDetailed("W029", "trip_id "+tripID, eid, "", map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
					}))
				}
			}

			if ev := stu.GetArrival(); ev != nil {
				if ev.GetDelay() == 0 && ev.GetTime() == 0 {
					out = append(out, errorListDetailed("W030", "trip_id "+tripID, eid, "", map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
						"event_type":    "arrival",
					}))
				}
			}
			if ev := stu.GetDeparture(); ev != nil {
				if ev.GetDelay() == 0 && ev.GetTime() == 0 {
					out = append(out, errorListDetailed("W030", "trip_id "+tripID, eid, "", map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
						"event_type":    "departure",
					}))
				}
			}

			if prevSeq >= 0 {
				if int(stopSeq) == prevSeq {
					out = append(out, errorListDetailed("E036", "trip_id "+tripID, eid, stopSeqStr, map[string]string{
						"trip_id":       tripID,
						"stop_sequence": stopSeqStr,
					}))
				}
				if stopID != "" && prevStopID != "" && stopID == prevStopID {
					out = append(out, errorListDetailed("W012", "trip_id "+tripID, eid, stopID, map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
					}))
				}
				if stopSeq != 0 && int(stopSeq) <= prevSeq {
					out = append(out, errorListDetailed("E002", "trip_id "+tripID, eid, stopSeqStr, map[string]string{
						"trip_id":               tripID,
						"stop_sequence":         stopSeqStr,
						"previous_stop_sequence": fmt.Sprintf("%d", prevSeq),
					}))
				}
			}
			if stopSeq != 0 {
				prevSeq = int(stopSeq)
			}
			if stopID != "" {
				prevStopID = stopID
			}

			// W031, W034: Skip in RT-only mode (requires static GTFS stop_times)
			if !v.RTOnly && meta != nil && stopSeq != 0 && stopID != "" {
				m := seqMapCache[tripID]
				if m == nil {
					m = GetSeqMap()
					for _, st := range meta.StopTimesByTripID[tripID] {
						m[st.StopSequence] = st.StopID
					}
					seqMapCache[tripID] = m
				}
				if expectedStopID, ok := m[int(stopSeq)]; ok {
					if expectedStopID != stopID {
						out = append(out, errorListDetailed("W031", "trip_id "+tripID, eid, stopID, map[string]string{
							"trip_id":       tripID,
							"stop_sequence": stopSeqStr,
							"expected":      expectedStopID,
							"actual":        stopID,
						}))
					}
				} else {
					out = append(out, errorListDetailed("W034", "trip_id "+tripID, eid, stopSeqStr, map[string]string{
						"trip_id":       tripID,
						"stop_sequence": stopSeqStr,
					}))
					hadE051 = true
				}
			}
			if hadE051 {
				break
			}

			// W032: Skip in RT-only mode (requires static GTFS stop_times)
			if !v.RTOnly && meta != nil {
				if (stu.GetArrival() == nil || stu.GetArrival().GetTime() == 0) && (stu.GetDeparture() == nil || stu.GetDeparture().GetTime() == 0) {
					if seq := int(stopSeq); seq > 0 {
						for _, st := range meta.StopTimesByTripID[tripID] {
							if st.StopSequence == seq {
								if st.ArrivalTime == "" && st.DepartureTime == "" {
									out = append(out, errorListDetailed("W032", "trip_id "+tripID, eid, "", map[string]string{
										"trip_id":       tripID,
										"stop_id":       stopID,
										"stop_sequence": stopSeqStr,
									}))
								}
								break
							}
						}
					}
				}
			}

			// W017: Skip in RT-only mode (requires static GTFS trips with multi-stops)
			if !v.RTOnly && meta != nil {
				if _, hasRepeat := meta.TripsWithMultiStops[tripID]; hasRepeat {
					if stopSeq == 0 {
						out = append(out, errorListDetailed("W017", "trip_id "+tripID, eid, "", map[string]string{
							"trip_id": tripID,
							"stop_id": stopID,
						}))
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
					out = append(out, errorListDetailed("W010", "trip_id "+tripID, eid, fmt.Sprintf("%d", thisTime), map[string]string{
						"trip_id":       tripID,
						"stop_id":       stopID,
						"stop_sequence": stopSeqStr,
						"current_time":  fmt.Sprintf("%d", thisTime),
						"previous_time": fmt.Sprintf("%d", prevArr),
					}))
				}
				prevArr = thisTime
			}
			if stu.GetArrival() != nil && stu.GetDeparture() != nil && stu.GetArrival().GetTime() != 0 && stu.GetDeparture().GetTime() != 0 {
				if stu.GetDeparture().GetTime() < stu.GetArrival().GetTime() {
					out = append(out, errorListDetailed("E025", "trip_id "+tripID, eid, "", map[string]string{
						"trip_id":        tripID,
						"stop_id":        stopID,
						"stop_sequence":  stopSeqStr,
						"arrival_time":   fmt.Sprintf("%d", stu.GetArrival().GetTime()),
						"departure_time": fmt.Sprintf("%d", stu.GetDeparture().GetTime()),
					}))
				}
			}
		}

		if len(tu.GetStopTimeUpdate()) == 0 && tu.GetTrip().GetScheduleRelationship() != rt.TripDescriptor_CANCELED {
			out = append(out, errorListDetailed("W013", "trip_id "+tripID, eid, "", map[string]string{
				"trip_id": tripID,
			}))
		}

		if m := seqMapCache[tripID]; m != nil {
			PutSeqMap(m)
		}
		delete(seqMapCache, tripID)
	}
	return out
}
