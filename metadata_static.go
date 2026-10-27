package validator

// Metadata answers the Static lookups from its maps, ignoring the service date.
var _ Static = (*Metadata)(nil)

func (md *Metadata) Trip(tripID, _ string) (TripInfo, Presence) {
	trip, ok := md.TripsByTripID[tripID]
	if !ok {
		return TripInfo{}, Absent
	}
	info := TripInfo{RouteID: trip.RouteID, DirectionID: -1, FirstArrival: -1, md: md.StopTimesByTripID[tripID]}
	if trip.DirectionID != nil {
		info.DirectionID = *trip.DirectionID
	}
	if len(info.md) > 0 {
		if t, ok := parseClock(info.md[0].ArrivalTime); ok {
			info.FirstArrival = t
		}
	}
	if _, ok := md.ExactTimesZeroTripIDs[tripID]; ok {
		info.Frequency = FrequencyInexact
	} else if len(md.ExactTimesOneByTripID[tripID]) > 0 {
		info.Frequency = FrequencyExact
	}
	_, info.RepeatsStop = md.TripsWithMultiStops[tripID]
	return info, Present
}

func (md *Metadata) Stop(stopID, _ string) (int, Presence) {
	if _, ok := md.StopIDs[stopID]; !ok {
		return 0, Absent
	}
	return md.StopToLocationType[stopID], Present
}

func (md *Metadata) Route(routeID string) Presence {
	if _, ok := md.RouteIDs[routeID]; ok {
		return Present
	}
	return Absent
}

func (md *Metadata) Agency(agencyID string) Presence {
	if _, ok := md.AgencyIDs[agencyID]; ok {
		return Present
	}
	return Absent
}

func (md *Metadata) ServiceArea() (Rect, bool) {
	if md.ShapeBBoxBuffered != nil {
		return *md.ShapeBBoxBuffered, true
	}
	return md.StopBBoxBuffered, true
}
