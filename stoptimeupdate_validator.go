package validator

import (
	"context"

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
		tripID := tu.GetTrip().GetTripId()
		if tripID == "" {
			continue
		}

		var hadE051 bool
		var prevSeq int = -1
		var prevStopID string
		var prevArr int64

		for _, stu := range tu.GetStopTimeUpdate() {
			if stu.GetStopId() == "" && stu.GetStopSequence() == 0 {
				out = append(out, el("E040", "trip_id "+tripID))
			}

			if stu.GetScheduleRelationship() == rt.TripUpdate_StopTimeUpdate_NO_DATA {
				if stu.GetArrival() != nil || stu.GetDeparture() != nil {
					out = append(out, el("E042", "trip_id "+tripID))
				}
				continue
			}

			if stu.GetScheduleRelationship() != rt.TripUpdate_StopTimeUpdate_SKIPPED && stu.GetScheduleRelationship() != rt.TripUpdate_StopTimeUpdate_NO_DATA {
				if stu.GetArrival() == nil && stu.GetDeparture() == nil {
					out = append(out, el("W029", "trip_id "+tripID))
				}
			}

			if ev := stu.GetArrival(); ev != nil {
				if ev.GetDelay() == 0 && ev.GetTime() == 0 {
					out = append(out, el("W030", "trip_id "+tripID))
				}
			}
			if ev := stu.GetDeparture(); ev != nil {
				if ev.GetDelay() == 0 && ev.GetTime() == 0 {
					out = append(out, el("W030", "trip_id "+tripID))
				}
			}

			if prevSeq >= 0 {
				if int(stu.GetStopSequence()) == prevSeq {
					out = append(out, el("E036", "trip_id "+tripID))
				}
				if stu.GetStopId() != "" && prevStopID != "" && stu.GetStopId() == prevStopID {
					out = append(out, el("W012", "trip_id "+tripID))
				}
				if stu.GetStopSequence() != 0 && int(stu.GetStopSequence()) <= prevSeq {
					out = append(out, el("E002", "trip_id "+tripID))
				}
			}
			if stu.GetStopSequence() != 0 {
				prevSeq = int(stu.GetStopSequence())
			}
			if stu.GetStopId() != "" {
				prevStopID = stu.GetStopId()
			}

			// W031, W034: Skip in RT-only mode (requires static GTFS stop_times)
			if !v.RTOnly && meta != nil && stu.GetStopSequence() != 0 && stu.GetStopId() != "" {
				m := seqMapCache[tripID]
				if m == nil {
					m = GetSeqMap()
					for _, st := range meta.StopTimesByTripID[tripID] {
						m[st.StopSequence] = st.StopID
					}
					seqMapCache[tripID] = m
				}
				if sid, ok := m[int(stu.GetStopSequence())]; ok {
					if sid != stu.GetStopId() {
						out = append(out, el("W031", "trip_id "+tripID))
					}
				} else {
					out = append(out, el("W034", "trip_id "+tripID))
					hadE051 = true
				}
			}
			if hadE051 {
				break
			}

			// W032: Skip in RT-only mode (requires static GTFS stop_times)
			if !v.RTOnly && meta != nil {
				if (stu.GetArrival() == nil || stu.GetArrival().GetTime() == 0) && (stu.GetDeparture() == nil || stu.GetDeparture().GetTime() == 0) {
					if seq := int(stu.GetStopSequence()); seq > 0 {
						for _, st := range meta.StopTimesByTripID[tripID] {
							if st.StopSequence == seq {
								if st.ArrivalTime == "" && st.DepartureTime == "" {
									out = append(out, el("W032", "trip_id "+tripID))
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
					if stu.GetStopSequence() == 0 {
						out = append(out, el("W017", "trip_id "+tripID))
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
					out = append(out, el("W010", "trip_id "+tripID))
				}
				prevArr = thisTime
			}
			if stu.GetArrival() != nil && stu.GetDeparture() != nil && stu.GetArrival().GetTime() != 0 && stu.GetDeparture().GetTime() != 0 {
				if stu.GetDeparture().GetTime() < stu.GetArrival().GetTime() {
					out = append(out, el("E025", "trip_id "+tripID))
				}
			}
		}

		if len(tu.GetStopTimeUpdate()) == 0 && tu.GetTrip().GetScheduleRelationship() != rt.TripDescriptor_CANCELED {
			out = append(out, el("W013", "trip_id "+tripID))
		}

		if m := seqMapCache[tripID]; m != nil {
			PutSeqMap(m)
		}
		delete(seqMapCache, tripID)
	}
	return out
}
