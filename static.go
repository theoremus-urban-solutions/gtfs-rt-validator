package validator

import "strconv"

// Presence says whether the static release knows an id.
type Presence uint8

const (
	// Absent means the loaded release does not hold the id.
	Absent Presence = iota
	// Present means the loaded release holds the id.
	Present
	// NotCovered means no loaded release can answer, so rules skip the check.
	NotCovered
)

// FrequencyKind says how frequencies.txt lists a trip.
type FrequencyKind uint8

const (
	NotFrequency FrequencyKind = iota
	// FrequencyInexact is a trip listed with exact_times 0.
	FrequencyInexact
	// FrequencyExact is a trip listed with exact_times 1.
	FrequencyExact
)

// Static is the static GTFS release the rules check a feed against.
type Static interface {
	Trip(tripID, startDate string) (TripInfo, Presence)
	Stop(stopID, startDate string) (locationType int, p Presence)
	Route(routeID string) Presence
	Agency(agencyID string) Presence
	// ServiceArea is the buffered box E028 checks; ok is false when the release has no box.
	ServiceArea() (area Rect, ok bool)
}

// TripInfo is what the rules read about one GTFS trip.
type TripInfo struct {
	RouteID string
	// DirectionID is -1 when trips.txt leaves it blank.
	DirectionID int
	// FirstArrival is the lowest stop_sequence's arrival_time in seconds after midnight, or -1 when blank.
	FirstArrival int32
	Frequency    FrequencyKind
	// RepeatsStop is true when the trip visits one stop more than once.
	RepeatsStop bool

	// md backs a Metadata trip; recs and stopNames back a StaticIndex trip.
	md        []StopTime
	recs      []stopRec
	stopNames []string
}

// ScheduledStop is one stop_times.txt row of a trip.
type ScheduledStop struct {
	Sequence     uint32
	StopID       string
	ArrivalSet   bool
	DepartureSet bool
}

// NumStops is the number of stop_times rows of the trip.
func (t TripInfo) NumStops() int {
	if t.recs != nil {
		return len(t.recs)
	}
	return len(t.md)
}

// StopAt returns the i-th stop_times row in stop_sequence order.
func (t TripInfo) StopAt(i int) ScheduledStop {
	if t.recs != nil {
		r := t.recs[i]
		return ScheduledStop{
			Sequence:     uint32(r.seq),
			StopID:       t.stopNames[r.stop&stopMask],
			ArrivalSet:   r.stop&stopArrSet != 0,
			DepartureSet: r.stop&stopDepSet != 0,
		}
	}
	st := t.md[i]
	return ScheduledStop{
		Sequence:     uint32(st.StopSequence),
		StopID:       st.StopID,
		ArrivalSet:   st.ArrivalTime != "",
		DepartureSet: st.DepartureTime != "",
	}
}

// parseClock reads a GTFS H:MM:SS or HH:MM:SS time as seconds after midnight.
func parseClock(s string) (int32, bool) {
	var parts [3]int32
	n, digits := 0, 0
	for i := 0; i < len(s); i++ {
		switch ch := s[i]; {
		case ch >= '0' && ch <= '9':
			if digits++; digits > 3 || (n > 0 && digits > 2) {
				return 0, false
			}
			parts[n] = parts[n]*10 + int32(ch-'0')
		case ch == ':' && digits > 0 && n < 2 && (n == 0 || digits == 2):
			n, digits = n+1, 0
		default:
			return 0, false
		}
	}
	if n != 2 || digits != 2 || parts[1] > 59 || parts[2] > 59 {
		return 0, false
	}
	return parts[0]*3600 + parts[1]*60 + parts[2], true
}

// formatClock writes seconds after midnight as HH:MM:SS.
func formatClock(sec int32) string {
	h, m, s := sec/3600, sec/60%60, sec%60
	b := []byte{byte('0' + h/10%10), byte('0' + h%10), ':', byte('0' + m/10), byte('0' + m%10), ':', byte('0' + s/10), byte('0' + s%10)}
	if h >= 100 {
		return strconv.Itoa(int(h)) + string(b[2:])
	}
	return string(b)
}
