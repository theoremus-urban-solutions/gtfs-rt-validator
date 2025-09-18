package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type StopTimeUpdateValidator struct{}

func (StopTimeUpdateValidator) Validate(ctx context.Context, _ int64, meta *Metadata, feed, _, _ *rt.FeedMessage) []ErrorList {
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
					out = append(out, el("E043", "trip_id "+tripID))
				}
			}

			if ev := stu.GetArrival(); ev != nil {
				if ev.GetDelay() == 0 && ev.GetTime() == 0 {
					out = append(out, el("E044", "trip_id "+tripID))
				}
			}
			if ev := stu.GetDeparture(); ev != nil {
				if ev.GetDelay() == 0 && ev.GetTime() == 0 {
					out = append(out, el("E044", "trip_id "+tripID))
				}
			}

			if prevSeq >= 0 {
				if int(stu.GetStopSequence()) == prevSeq {
					out = append(out, el("E036", "trip_id "+tripID))
				}
				if stu.GetStopId() != "" && prevStopID != "" && stu.GetStopId() == prevStopID {
					out = append(out, el("E037", "trip_id "+tripID))
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

			if stu.GetStopSequence() != 0 && stu.GetStopId() != "" {
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
						out = append(out, el("E045", "trip_id "+tripID))
					}
				} else {
					out = append(out, el("E051", "trip_id "+tripID))
					hadE051 = true
				}
			}
			if hadE051 {
				break
			}

			if (stu.GetArrival() == nil || stu.GetArrival().GetTime() == 0) && (stu.GetDeparture() == nil || stu.GetDeparture().GetTime() == 0) {
				if seq := int(stu.GetStopSequence()); seq > 0 {
					for _, st := range meta.StopTimesByTripID[tripID] {
						if st.StopSequence == seq {
							if st.ArrivalTime == "" && st.DepartureTime == "" {
								out = append(out, el("E046", "trip_id "+tripID))
							}
							break
						}
					}
				}
			}

			if _, hasRepeat := meta.TripsWithMultiStops[tripID]; hasRepeat {
				if stu.GetStopSequence() == 0 {
					out = append(out, el("E009", "trip_id "+tripID))
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
					out = append(out, el("E022", "trip_id "+tripID))
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
			out = append(out, el("E041", "trip_id "+tripID))
		}

		if m := seqMapCache[tripID]; m != nil {
			PutSeqMap(m)
		}
		delete(seqMapCache, tripID)
	}
	return out
}
