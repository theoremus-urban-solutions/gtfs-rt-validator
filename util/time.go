package util

import (
	"time"
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
