package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	// "github.com/paulmach/orb/planar" // reserved for buffered shapes

	"your.org/gtfsrt/validator/schema"
)

// LoadMetadata parses GTFS files and builds schema.Metadata
func LoadMetadata(path string, ignoreShapes bool) (*schema.Metadata, error) {
	readers, cleanup, err := ZipOrDirReaders(path)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	md := &schema.Metadata{
		AgencyIDs:             make(map[string]struct{}),
		RouteIDs:              make(map[string]struct{}),
		TripsByTripID:         make(map[string]schema.Trip),
		StopTimesByTripID:     make(map[string][]schema.StopTime),
		StopIDs:               make(map[string]struct{}),
		StopToLocationType:    make(map[string]int),
		ExactTimesZeroTripIDs: make(map[string]struct{}),
		ExactTimesOneByTripID: make(map[string][]schema.Frequency),
		ShapePointsByShapeID:  make(map[string][]schema.ShapePoint),
		TripShapes:            make(map[string][][]float64),
		TripShapesBuffered:    make(map[string][][]float64),
		TripsWithMultiStops:   make(map[string][]string),
	}

	if r := readers["agency.txt"]; r != nil {
		if err := parseAgency(r, md); err != nil {
			return nil, err
		}
	}
	if r := readers["routes.txt"]; r != nil {
		if err := parseRoutes(r, md); err != nil {
			return nil, err
		}
	}
	if r := readers["trips.txt"]; r != nil {
		if err := parseTrips(r, md); err != nil {
			return nil, err
		}
	}
	if r := readers["stops.txt"]; r != nil {
		if err := parseStops(r, md); err != nil {
			return nil, err
		}
	}
	if r := readers["stop_times.txt"]; r != nil {
		if err := parseStopTimes(r, md); err != nil {
			return nil, err
		}
	}
	if r := readers["frequencies.txt"]; r != nil {
		if err := parseFrequencies(r, md); err != nil {
			return nil, err
		}
	}
	if !ignoreShapes {
		if r := readers["shapes.txt"]; r != nil {
			if err := parseShapes(r, md); err != nil {
				return nil, err
			}
		}
	}

	// Compute buffered bboxes
	regionBufDeg := schema.KM_TO_DEG * (schema.REGION_BUFFER_METERS / 1000.0)
	stopBuf := md.StopBBox.Buffered(regionBufDeg)
	md.StopBBoxBuffered = stopBuf
	if md.ShapeBBox != nil {
		sb := md.ShapeBBox.Buffered(regionBufDeg)
		md.ShapeBBoxBuffered = &sb
	}

	// Timezone from agency
	if tz := firstAgencyTZ(readers["agency.txt"]); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			md.TimeZone = loc
		}
	}

	// Trips with repeated stops
	for tripID, sts := range md.StopTimesByTripID {
		seen := make(map[string]int)
		for _, st := range sts {
			seen[st.StopID]++
		}
		var multi []string
		for s, c := range seen {
			if c > 1 {
				multi = append(multi, s)
			}
		}
		if len(multi) > 0 {
			md.TripsWithMultiStops[tripID] = multi
		}
	}

	// buffer computations for shapes will be added when needed in vehicle validation
	return md, nil
}

func parseAgency(r *csv.Reader, md *schema.Metadata) error {
	headers, err := r.Read()
	if err != nil {
		return err
	}
	idx := index(headers)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		id := get(rec, idx, "agency_id")
		if id == "" {
			id = "default"
		}
		md.AgencyIDs[id] = struct{}{}
	}
	return nil
}

func parseRoutes(r *csv.Reader, md *schema.Metadata) error {
	h, err := r.Read()
	if err != nil {
		return err
	}
	idx := index(h)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		id := get(rec, idx, "route_id")
		if id != "" {
			md.RouteIDs[id] = struct{}{}
		}
	}
	return nil
}

func parseTrips(r *csv.Reader, md *schema.Metadata) error {
	h, err := r.Read()
	if err != nil {
		return err
	}
	idx := index(h)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		t := schema.Trip{
			TripID:    get(rec, idx, "trip_id"),
			RouteID:   get(rec, idx, "route_id"),
			ServiceID: get(rec, idx, "service_id"),
			ShapeID:   get(rec, idx, "shape_id"),
			BlockID:   get(rec, idx, "block_id"),
		}
		if d := get(rec, idx, "direction_id"); d != "" {
			if v, err := strconv.Atoi(d); err == nil {
				t.DirectionID = &v
			}
		}
		if t.TripID != "" {
			md.TripsByTripID[t.TripID] = t
		}
	}
	return nil
}

func parseStops(r *csv.Reader, md *schema.Metadata) error {
	h, err := r.Read()
	if err != nil {
		return err
	}
	idx := index(h)
	first := true
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		id := get(rec, idx, "stop_id")
		if id == "" {
			continue
		}
		md.StopIDs[id] = struct{}{}
		if lt := get(rec, idx, "location_type"); lt != "" {
			if v, err := strconv.Atoi(lt); err == nil {
				md.StopToLocationType[id] = v
			}
		}
		lat, _ := strconv.ParseFloat(get(rec, idx, "stop_lat"), 64)
		lon, _ := strconv.ParseFloat(get(rec, idx, "stop_lon"), 64)
		if first {
			md.StopBBox = schema.Rect{MinLat: lat, MaxLat: lat, MinLon: lon, MaxLon: lon}
			first = false
		}
		md.StopBBox.Expand(lat, lon)
	}
	return nil
}

func parseStopTimes(r *csv.Reader, md *schema.Metadata) error {
	h, err := r.Read()
	if err != nil {
		return err
	}
	idx := index(h)
	byTrip := make(map[string][]schema.StopTime)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		st := schema.StopTime{
			TripID:        get(rec, idx, "trip_id"),
			ArrivalTime:   get(rec, idx, "arrival_time"),
			DepartureTime: get(rec, idx, "departure_time"),
			StopID:        get(rec, idx, "stop_id"),
		}
		if s := get(rec, idx, "stop_sequence"); s != "" {
			if v, err := strconv.Atoi(s); err == nil {
				st.StopSequence = v
			}
		}
		if st.TripID != "" {
			byTrip[st.TripID] = append(byTrip[st.TripID], st)
		}
	}
	// sort by stop_sequence
	for k, v := range byTrip {
		md.StopTimesByTripID[k] = stableSortStopTimes(v)
	}
	return nil
}

func parseFrequencies(r *csv.Reader, md *schema.Metadata) error {
	h, err := r.Read()
	if err != nil {
		return err
	}
	idx := index(h)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		f := schema.Frequency{
			TripID:    get(rec, idx, "trip_id"),
			StartTime: get(rec, idx, "start_time"),
			EndTime:   get(rec, idx, "end_time"),
		}
		if hw := get(rec, idx, "headway_secs"); hw != "" {
			if v, err := strconv.Atoi(hw); err == nil {
				f.HeadwaySecs = v
			}
		}
		if et := get(rec, idx, "exact_times"); et != "" {
			if v, err := strconv.Atoi(et); err == nil {
				f.ExactTimes = &v
			}
		}
		if f.TripID != "" {
			if f.ExactTimes != nil && *f.ExactTimes == 0 {
				md.ExactTimesZeroTripIDs[f.TripID] = struct{}{}
			}
			if f.ExactTimes != nil && *f.ExactTimes == 1 {
				md.ExactTimesOneByTripID[f.TripID] = append(md.ExactTimesOneByTripID[f.TripID], f)
			}
		}
	}
	return nil
}

func parseShapes(r *csv.Reader, md *schema.Metadata) error {
	h, err := r.Read()
	if err != nil {
		return err
	}
	idx := index(h)
	var first bool = true
	var sb schema.Rect
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		sp := schema.ShapePoint{
			ShapeID: get(rec, idx, "shape_id"),
		}
		sp.Lat, _ = strconv.ParseFloat(get(rec, idx, "shape_pt_lat"), 64)
		sp.Lon, _ = strconv.ParseFloat(get(rec, idx, "shape_pt_lon"), 64)
		if s := get(rec, idx, "shape_pt_sequence"); s != "" {
			if v, err := strconv.Atoi(s); err == nil {
				sp.Sequence = v
			}
		}
		if sp.ShapeID != "" {
			md.ShapePointsByShapeID[sp.ShapeID] = append(md.ShapePointsByShapeID[sp.ShapeID], sp)
		}
		if first {
			sb = schema.Rect{MinLat: sp.Lat, MaxLat: sp.Lat, MinLon: sp.Lon, MaxLon: sp.Lon}
			first = false
		}
		sb.Expand(sp.Lat, sp.Lon)
	}
	if !first {
		md.ShapeBBox = &sb
	}
	return nil
}

func index(headers []string) map[string]int {
	m := make(map[string]int)
	for i, h := range headers {
		m[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return m
}

func get(rec []string, idx map[string]int, key string) string {
	if i, ok := idx[strings.ToLower(key)]; ok && i < len(rec) {
		return strings.TrimSpace(rec[i])
	}
	return ""
}

func stableSortStopTimes(in []schema.StopTime) []schema.StopTime {
	// simple insertion sort to keep stable and small dependency footprint
	for i := 1; i < len(in); i++ {
		j := i
		for j > 0 && in[j-1].StopSequence > in[j].StopSequence {
			in[j-1], in[j] = in[j], in[j-1]
			j--
		}
	}
	return in
}

func firstAgencyTZ(r *csv.Reader) string {
	if r == nil {
		return ""
	}
	// Cannot re-read easily; rely on earlier parse to set timezone
	_ = fmt.Sprintf
	return ""
}
