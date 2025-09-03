package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"your.org/gtfsrt/validator/notice"
	"your.org/gtfsrt/validator/pools"
	"your.org/gtfsrt/validator/schema"
)

type StopTimeUpdateValidator struct{}

func (StopTimeUpdateValidator) Validate(ctx context.Context, _ int64, meta *schema.Metadata, feed, _, _ *rt.FeedMessage) []notice.ErrorList {
	var out []notice.ErrorList
	// Build per-trip stop_sequence->stop_id maps lazily
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
			// E040: must have stop_id or stop_sequence
			if stu.GetStopId() == "" && stu.GetStopSequence() == 0 {
				out = append(out, el("E040", "trip_id "+tripID))
			}

			// E041: handled after loop if no STUs exist (not here)

			// E042: NO_DATA must not have arrival/departure
			if stu.GetScheduleRelationship() == rt.TripUpdate_StopTimeUpdate_NO_DATA {
				if stu.GetArrival() != nil || stu.GetDeparture() != nil {
					out = append(out, el("E042", "trip_id "+tripID))
				}
				continue
			}

			// E043: if not SKIPPED/NO_DATA must have arrival or departure
			if stu.GetScheduleRelationship() != rt.TripUpdate_StopTimeUpdate_SKIPPED && stu.GetScheduleRelationship() != rt.TripUpdate_StopTimeUpdate_NO_DATA {
				if stu.GetArrival() == nil && stu.GetDeparture() == nil {
					out = append(out, el("E043", "trip_id "+tripID))
				}
			}

			// E044: arrival/departure must have delay or time
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

			// E002/E036/E037 ordering and duplicates
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

			// E045: if both seq and id present, must match GTFS
			if stu.GetStopSequence() != 0 && stu.GetStopId() != "" {
				m := seqMapCache[tripID]
				if m == nil {
					m = pools.GetSeqMap()
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
					// E051: sequence not in GTFS; early-exit for this trip
					out = append(out, el("E051", "trip_id "+tripID))
					hadE051 = true
				}
			}
			if hadE051 {
				break
			}

			// E046: if time missing in STU, ensure GTFS has times at that stop
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

			// E009: Trip with repeated stops must include stop_sequence
			if _, hasRepeat := meta.TripsWithMultiStops[tripID]; hasRepeat {
				if stu.GetStopSequence() == 0 {
					out = append(out, el("E009", "trip_id "+tripID))
				}
			}

			// E022/E025 time ordering across STUs
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

		// E041: no STUs
		if len(tu.GetStopTimeUpdate()) == 0 && tu.GetTrip().GetScheduleRelationship() != rt.TripDescriptor_CANCELED {
			out = append(out, el("E041", "trip_id "+tripID))
		}

		// return pooled maps for this trip
		if m := seqMapCache[tripID]; m != nil {
			pools.PutSeqMap(m)
		}
		delete(seqMapCache, tripID)
	}
	return out
}
