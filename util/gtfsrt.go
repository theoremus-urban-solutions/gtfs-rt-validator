package util

import (
	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

func IsValidVersion(h *rt.FeedHeader) bool {
	if h == nil {
		return false
	}
	v := h.GetGtfsRealtimeVersion()
	return v == "1.0" || v == "2.0"
}

func IsV2OrHigher(h *rt.FeedHeader) bool {
	if h == nil {
		return false
	}
	return h.GetGtfsRealtimeVersion() >= "2.0"
}

func IsPositionValid(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func IsBearingValid(hasBearing bool, bearing float32) bool {
	if !hasBearing {
		return true
	}
	return bearing >= 0 && bearing <= 360
}

func ToMilesPerHour(mps float32) float32 {
	return mps * 2.2369363
}

func ToMiles(meters float64) float64 {
	return meters * 0.000621371
}

// IsCombinedFeed returns true if at least two different entity types exist
func IsCombinedFeed(feed *rt.FeedMessage) bool {
	if feed == nil {
		return false
	}
	hasTU, hasVP, hasAL := false, false, false
	for _, e := range feed.GetEntity() {
		if e.GetTripUpdate() != nil {
			hasTU = true
		}
		if e.GetVehicle() != nil {
			hasVP = true
		}
		if e.GetAlert() != nil {
			hasAL = true
		}
		if (hasTU && hasVP) || (hasTU && hasAL) || (hasVP && hasAL) {
			return true
		}
	}
	return false
}
