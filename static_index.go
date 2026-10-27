package validator

import (
	"archive/zip"
	"bytes"
	"cmp"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// StaticIndex is a compact read-only copy of a GTFS release that keeps only what the rules read.
type StaticIndex struct {
	trips    map[string]int32
	tripRecs []tripRec
	stopRecs []stopRec

	// routes holds every route_id seen in routes.txt or trips.txt; routeInFile marks those in routes.txt.
	routes      map[string]int32
	routeNames  []string
	routeInFile []bool

	// stops holds every stop_id seen in stops.txt or stop_times.txt; stopLT is -1 for those missing from stops.txt.
	stops     map[string]int32
	stopNames []string
	stopLT    []int8

	agencies map[string]struct{}
	area     Rect
}

var _ Static = (*StaticIndex)(nil)

type tripRec struct {
	off, n       int32
	route        int32 // -1 when blank
	firstArrival int32
	dir          int8
	freq         FrequencyKind
	repeats      bool
}

// stopRec is one stop_times row; the low 30 bits of stop index stopNames and the top two say arrival_time and departure_time are set.
type stopRec struct {
	seq  int32
	stop uint32
}

const (
	stopMask   = 1<<30 - 1
	stopArrSet = 1 << 30
	stopDepSet = 1 << 31
)

type openers map[string]func() (io.ReadCloser, error)

// LoadStaticIndex reads a GTFS zip and fails on a missing file, column or row, or a CSV error.
func LoadStaticIndex(r io.ReaderAt, size int64) (*StaticIndex, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("failed to open GTFS zip: %w", err)
	}
	files := openers{}
	for _, f := range zr.File {
		if base := strings.ToLower(filepath.Base(f.Name)); isKnownGTFS(base) {
			files[base] = f.Open
		}
	}
	return buildStaticIndex(files)
}

// LoadStaticIndexPath reads a GTFS zip file or a directory of GTFS files.
func LoadStaticIndexPath(path string) (*StaticIndex, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return LoadStaticIndex(f, info.Size())
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	files := openers{}
	for _, e := range entries {
		if base := strings.ToLower(e.Name()); !e.IsDir() && isKnownGTFS(base) {
			p := filepath.Join(path, e.Name())
			files[base] = func() (io.ReadCloser, error) { return os.Open(p) }
		}
	}
	return buildStaticIndex(files)
}

// eachRow opens one file, checks its header and calls fn for every row with the index of each wanted column; the row slice is reused.
func (files openers) eachRow(name string, want []string, fn func(rec [][]byte, at []int)) error {
	open := files[name]
	if open == nil {
		return nil
	}
	rc, err := open()
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	defer rc.Close()
	r := newCSVScanner(rc)
	h, err := r.Read()
	if err == io.EOF {
		return fmt.Errorf("%s has no rows", name)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	header := make([]string, len(h))
	for i, f := range h {
		header[i] = string(f)
	}
	cols, err := checkHeader(name, header)
	if err != nil {
		return err
	}
	at := make([]int, len(want))
	for i, w := range want {
		at[i] = -1
		if j, ok := cols[w]; ok {
			at[i] = j
		}
	}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		fn(rec, at)
	}
}

func field(rec [][]byte, i int) []byte {
	if i >= 0 && i < len(rec) {
		return bytes.TrimSpace(rec[i])
	}
	return nil
}

// atoi and atof skip blank fields, where strconv would allocate an error.
func atoi(b []byte) (int, bool) {
	if len(b) == 0 {
		return 0, false
	}
	v, err := strconv.Atoi(string(b))
	return v, err == nil
}

func atof(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(string(b), 64)
	return v
}

func buildStaticIndex(files openers) (*StaticIndex, error) {
	for _, name := range requiredFiles {
		if files[name] == nil {
			return nil, fmt.Errorf("%s is missing", name)
		}
	}
	ix := &StaticIndex{
		trips:    make(map[string]int32),
		routes:   make(map[string]int32),
		stops:    make(map[string]int32),
		agencies: make(map[string]struct{}),
	}

	// shapes.txt is only read for its bounding box, so it is read beside the other files.
	type boxResult struct {
		box *Rect
		err error
	}
	shapes := make(chan boxResult, 1)
	go func() {
		var box *Rect
		err := files.eachRow("shapes.txt", []string{"shape_pt_lat", "shape_pt_lon"}, func(rec [][]byte, at []int) {
			box = expand(box, atof(field(rec, at[0])), atof(field(rec, at[1])))
		})
		shapes <- boxResult{box, err}
	}()

	if err := files.eachRow("agency.txt", []string{"agency_id"}, func(rec [][]byte, at []int) {
		id := field(rec, at[0])
		if len(id) == 0 {
			id = []byte("default")
		}
		if _, ok := ix.agencies[string(id)]; !ok {
			ix.agencies[string(id)] = struct{}{}
		}
	}); err != nil {
		return nil, err
	}

	nRoutes := 0
	if err := files.eachRow("routes.txt", []string{"route_id"}, func(rec [][]byte, at []int) {
		if id := field(rec, at[0]); len(id) > 0 {
			ix.routeInFile[ix.route(id)] = true
			nRoutes++
		}
	}); err != nil {
		return nil, err
	}

	if err := files.eachRow("trips.txt", []string{"trip_id", "route_id", "direction_id"}, func(rec [][]byte, at []int) {
		id := field(rec, at[0])
		if len(id) == 0 {
			return
		}
		tr := tripRec{route: -1, firstArrival: -1, dir: -1}
		if rid := field(rec, at[1]); len(rid) > 0 {
			tr.route = ix.route(rid)
		}
		if v, ok := atoi(field(rec, at[2])); ok {
			tr.dir = int8(v)
		}
		if i, ok := ix.trips[string(id)]; ok {
			ix.tripRecs[i] = tr
			return
		}
		ix.trips[string(id)] = int32(len(ix.tripRecs))
		ix.tripRecs = append(ix.tripRecs, tr)
	}); err != nil {
		return nil, err
	}

	nStops := 0
	var stopBox *Rect
	if err := files.eachRow("stops.txt", []string{"stop_id", "location_type", "stop_lat", "stop_lon"}, func(rec [][]byte, at []int) {
		id := field(rec, at[0])
		if len(id) == 0 {
			return
		}
		i := ix.stop(id)
		if ix.stopLT[i] < 0 {
			ix.stopLT[i] = 0
			nStops++
		}
		if v, ok := atoi(field(rec, at[1])); ok {
			ix.stopLT[i] = int8(v)
		}
		stopBox = expand(stopBox, atof(field(rec, at[2])), atof(field(rec, at[3])))
	}); err != nil {
		return nil, err
	}

	if err := ix.loadStopTimes(files); err != nil {
		return nil, err
	}

	if err := files.eachRow("frequencies.txt", []string{"trip_id", "exact_times"}, func(rec [][]byte, at []int) {
		i, ok := ix.trips[string(field(rec, at[0]))]
		if !ok {
			return
		}
		// A blank exact_times means 0.
		exact := 0
		if et := field(rec, at[1]); len(et) > 0 {
			if exact, ok = atoi(et); !ok {
				return
			}
		}
		switch tr := &ix.tripRecs[i]; {
		case exact == 0:
			tr.freq = FrequencyInexact
		case exact == 1 && tr.freq == NotFrequency:
			tr.freq = FrequencyExact
		}
	}); err != nil {
		return nil, err
	}

	sr := <-shapes
	if sr.err != nil {
		return nil, sr.err
	}
	shapeBox := sr.box

	for name, n := range map[string]int{
		"agency.txt":     len(ix.agencies),
		"routes.txt":     nRoutes,
		"trips.txt":      len(ix.tripRecs),
		"stops.txt":      nStops,
		"stop_times.txt": len(ix.stopRecs),
	} {
		if n == 0 {
			return nil, fmt.Errorf("%s has no rows", name)
		}
	}

	buf := KM_TO_DEG * (REGION_BUFFER_METERS / 1000.0)
	if shapeBox != nil {
		ix.area = shapeBox.Buffered(buf)
	} else {
		ix.area = stopBox.Buffered(buf)
	}
	return ix, nil
}

func expand(r *Rect, lat, lon float64) *Rect {
	if r == nil {
		return &Rect{MinLat: lat, MaxLat: lat, MinLon: lon, MaxLon: lon}
	}
	r.Expand(lat, lon)
	return r
}

// route returns the index of a route_id, adding it when new.
func (ix *StaticIndex) route(b []byte) int32 {
	if i, ok := ix.routes[string(b)]; ok {
		return i
	}
	id := string(b)
	i := int32(len(ix.routeNames))
	ix.routes[id] = i
	ix.routeNames = append(ix.routeNames, id)
	ix.routeInFile = append(ix.routeInFile, false)
	return i
}

// stop returns the index of a stop_id, adding it as missing from stops.txt when new.
func (ix *StaticIndex) stop(b []byte) int32 {
	if i, ok := ix.stops[string(b)]; ok {
		return i
	}
	id := string(b)
	i := int32(len(ix.stopNames))
	ix.stops[id] = i
	ix.stopNames = append(ix.stopNames, id)
	ix.stopLT = append(ix.stopLT, -1)
	return i
}

// loadStopTimes groups stop_times rows by trip with a counting sort, then orders each trip by stop_sequence.
func (ix *StaticIndex) loadStopTimes(files openers) error {
	type row struct {
		trip int32
		rec  stopRec
	}
	// rows grow in fixed chunks so no copy is made as they grow.
	const chunk = 1 << 16
	var chunks [][]row
	rows := make([]row, 0, chunk)
	counts := make([]int32, len(ix.tripRecs)+1)
	minSeq := make([]int32, len(ix.tripRecs))
	err := files.eachRow("stop_times.txt", []string{"trip_id", "stop_id", "stop_sequence", "arrival_time", "departure_time"}, func(rec [][]byte, at []int) {
		t, ok := ix.trips[string(field(rec, at[0]))]
		if !ok {
			return
		}
		s := uint32(ix.stop(field(rec, at[1])))
		var seq int32
		if v, ok := atoi(field(rec, at[2])); ok {
			seq = int32(v)
		}
		arr := field(rec, at[3])
		if len(arr) > 0 {
			s |= stopArrSet
		}
		if len(field(rec, at[4])) > 0 {
			s |= stopDepSet
		}
		if len(rows) == chunk {
			chunks = append(chunks, rows)
			rows = make([]row, 0, chunk)
		}
		rows = append(rows, row{t, stopRec{seq: seq, stop: s}})
		counts[t+1]++
		// The first arrival comes from the first row with the lowest stop_sequence.
		if counts[t+1] == 1 || seq < minSeq[t] {
			minSeq[t] = seq
			ix.tripRecs[t].firstArrival = -1
			if v, ok := parseClock(string(arr)); ok {
				ix.tripRecs[t].firstArrival = v
			}
		}
	})
	if err != nil {
		return err
	}
	if len(ix.stopNames) > stopMask {
		return fmt.Errorf("stop_times.txt names more than %d stops", stopMask)
	}
	for i := 1; i < len(counts); i++ {
		counts[i] += counts[i-1]
	}
	chunks = append(chunks, rows)
	ix.stopRecs = make([]stopRec, counts[len(counts)-1])
	next := make([]int32, len(ix.tripRecs))
	copy(next, counts)
	for _, c := range chunks {
		for _, r := range c {
			ix.stopRecs[next[r.trip]] = r.rec
			next[r.trip]++
		}
	}
	chunks, rows = nil, nil
	// seen[stop] == trip+1 marks a stop the trip already visited.
	seen := make([]int32, len(ix.stopNames))
	for t := range ix.tripRecs {
		tr := &ix.tripRecs[t]
		tr.off, tr.n = counts[t], counts[t+1]-counts[t]
		seg := ix.stopRecs[tr.off : tr.off+tr.n]
		slices.SortStableFunc(seg, func(a, b stopRec) int { return cmp.Compare(a.seq, b.seq) })
		for _, r := range seg {
			s := r.stop & stopMask
			if seen[s] == int32(t)+1 {
				tr.repeats = true
				break
			}
			seen[s] = int32(t) + 1
		}
	}
	return nil
}

func (ix *StaticIndex) Trip(tripID, _ string) (TripInfo, Presence) {
	i, ok := ix.trips[tripID]
	if !ok {
		return TripInfo{}, Absent
	}
	tr := &ix.tripRecs[i]
	info := TripInfo{
		DirectionID:  int(tr.dir),
		FirstArrival: tr.firstArrival,
		Frequency:    tr.freq,
		RepeatsStop:  tr.repeats,
		recs:         ix.stopRecs[tr.off : tr.off+tr.n : tr.off+tr.n],
		stopNames:    ix.stopNames,
	}
	if tr.route >= 0 {
		info.RouteID = ix.routeNames[tr.route]
	}
	return info, Present
}

func (ix *StaticIndex) Stop(stopID, _ string) (int, Presence) {
	i, ok := ix.stops[stopID]
	if !ok || ix.stopLT[i] < 0 {
		return 0, Absent
	}
	return int(ix.stopLT[i]), Present
}

func (ix *StaticIndex) Route(routeID string) Presence {
	if i, ok := ix.routes[routeID]; ok && ix.routeInFile[i] {
		return Present
	}
	return Absent
}

func (ix *StaticIndex) Agency(agencyID string) Presence {
	if _, ok := ix.agencies[agencyID]; ok {
		return Present
	}
	return Absent
}

func (ix *StaticIndex) ServiceArea() (Rect, bool) { return ix.area, true }
