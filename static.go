package validator

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
	// FirstArrivalText is the arrival_time of the lowest stop_sequence as written in stop_times.txt.
	FirstArrivalText string
	Frequency        FrequencyKind
	// RepeatsStop is true when the trip visits one stop more than once.
	RepeatsStop bool

	md []StopTime
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
	return len(t.md)
}

// StopAt returns the i-th stop_times row in stop_sequence order.
func (t TripInfo) StopAt(i int) ScheduledStop {
	st := t.md[i]
	return ScheduledStop{
		Sequence:     uint32(st.StopSequence),
		StopID:       st.StopID,
		ArrivalSet:   st.ArrivalTime != "",
		DepartureSet: st.DepartureTime != "",
	}
}
