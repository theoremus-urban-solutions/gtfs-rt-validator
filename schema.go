package validator

import (
	"time"
)

type Agency struct {
	AgencyID string
	Name     string
	URL      string
	Timezone string
	Lang     string
}

type Route struct {
	RouteID   string
	AgencyID  string
	ShortName string
	LongName  string
	Type      int
}

type Trip struct {
	TripID      string
	RouteID     string
	ServiceID   string
	ShapeID     string
	DirectionID *int
	BlockID     string
}

type Stop struct {
	StopID       string
	Name         string
	Lat          float64
	Lon          float64
	LocationType *int
}

type StopTime struct {
	TripID        string
	ArrivalTime   string
	DepartureTime string
	StopID        string
	StopSequence  int
}

type ShapePoint struct {
	ShapeID  string
	Lat      float64
	Lon      float64
	Sequence int
}

type Frequency struct {
	TripID      string
	StartTime   string
	EndTime     string
	HeadwaySecs int
	ExactTimes  *int
}

type Rect struct {
	MinLat float64
	MinLon float64
	MaxLat float64
	MaxLon float64
}

func (r *Rect) Expand(lat, lon float64) {
	if lat < r.MinLat {
		r.MinLat = lat
	}
	if lat > r.MaxLat {
		r.MaxLat = lat
	}
	if lon < r.MinLon {
		r.MinLon = lon
	}
	if lon > r.MaxLon {
		r.MaxLon = lon
	}
}

func (r Rect) Buffered(deltaDeg float64) Rect {
	return Rect{
		MinLat: r.MinLat - deltaDeg,
		MinLon: r.MinLon - deltaDeg,
		MaxLat: r.MaxLat + deltaDeg,
		MaxLon: r.MaxLon + deltaDeg,
	}
}

func (r Rect) Contains(lat, lon float64) bool {
	return lat >= r.MinLat && lat <= r.MaxLat && lon >= r.MinLon && lon <= r.MaxLon
}

type Metadata struct {
	AgencyIDs             map[string]struct{}
	RouteIDs              map[string]struct{}
	TripsByTripID         map[string]Trip
	StopTimesByTripID     map[string][]StopTime
	StopIDs               map[string]struct{}
	StopToLocationType    map[string]int
	ExactTimesZeroTripIDs map[string]struct{}
	ExactTimesOneByTripID map[string][]Frequency
	ShapePointsByShapeID  map[string][]ShapePoint
	TripShapes            map[string][][]float64
	TripShapesBuffered    map[string][][]float64
	StopBBox              Rect
	StopBBoxBuffered      Rect
	ShapeBBox             *Rect
	ShapeBBoxBuffered     *Rect
	TripsWithMultiStops   map[string][]string
	TimeZone              *time.Location
}

const (
	REGION_BUFFER_METERS = 1609.0
	TRIP_BUFFER_METERS   = 200.0
	KM_TO_DEG            = 1.0 / 111.32
)
