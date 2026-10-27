package validator

import (
	"math"
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

// Buffered widens the box by deltaDeg of latitude on every side, scaling the longitude margin by latitude as spatial4j does.
func (r Rect) Buffered(deltaDeg float64) Rect {
	if r.MaxLat+deltaDeg >= 90 || r.MinLat-deltaDeg <= -90 {
		return Rect{MinLat: math.Max(-90, r.MinLat-deltaDeg), MinLon: -180, MaxLat: math.Min(90, r.MaxLat+deltaDeg), MaxLon: 180}
	}
	poleLat := r.MinLat
	if math.Abs(r.MaxLat) > math.Abs(r.MinLat) {
		poleLat = r.MaxLat
	}
	lonDeg := 90.0
	if d := math.Asin(math.Sin(deltaDeg*math.Pi/180) / math.Cos(poleLat*math.Pi/180)); !math.IsNaN(d) {
		lonDeg = d * 180 / math.Pi
	}
	if lonDeg*2+r.MaxLon-r.MinLon >= 360 {
		return Rect{MinLat: r.MinLat - deltaDeg, MinLon: -180, MaxLat: r.MaxLat + deltaDeg, MaxLon: 180}
	}
	return Rect{
		MinLat: r.MinLat - deltaDeg,
		MinLon: r.MinLon - lonDeg,
		MaxLat: r.MaxLat + deltaDeg,
		MaxLon: r.MaxLon + lonDeg,
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
