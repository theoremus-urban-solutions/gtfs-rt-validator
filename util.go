package validator

import (
	"time"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

const (
	MinPosixTime = int64(1104537600) // 2005-01-01
	MaxPosixTime = int64(1991620134) // parity with Java
)

func IsPosix(tsSec int64) bool {
	return tsSec >= MinPosixTime && tsSec <= MaxPosixTime
}

// GetAge returns age in seconds
func GetAge(currentMillis int64, tsSec int64) int64 {
	if tsSec <= 0 {
		return -1
	}
	return currentMillis/1000 - tsSec
}

func IsInFuture(currentMillis int64, tsSec int64, toleranceSec int64) bool {
	nowSec := currentMillis / 1000
	return tsSec > nowSec+toleranceSec
}

func PosixToClock(tsSec int64, tz *time.Location) string {
	if tz == nil {
		tz = time.UTC
	}
	return time.Unix(tsSec, 0).In(tz).Format(time.RFC3339)
}

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
