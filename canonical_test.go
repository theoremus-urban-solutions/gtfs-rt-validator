package validator

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

// fixtureFiles is the review's static GTFS: seven trips on R1 covering loops, frequencies, untimed stops and one-digit hours.
var fixtureFiles = map[string]string{
	"agency.txt": "agency_id,agency_name,agency_url,agency_timezone\nA,Agency,https://x.example,Europe/Sofia\n",
	"routes.txt": "route_id,agency_id,route_short_name,route_type\nR1,A,1,3\nR2,A,2,3\n",
	"trips.txt": "route_id,service_id,trip_id,direction_id\n" +
		"R1,S,T1,0\nR1,S,T_LOOP,0\nR1,S,T_FREQ_BLANK,0\nR1,S,T_FREQ0,0\nR1,S,T_FREQ1,0\nR1,S,T_UNTIMED,0\nR1,S,T_SHORT,0\n",
	"stops.txt": "stop_id,stop_name,stop_lat,stop_lon,location_type\nS1,a,42.70,23.30,0\nS2,b,42.71,23.31,0\nS3,c,42.72,23.32,0\nS4,d,42.73,23.33,0\nST,station,42.70,23.30,1\n",
	"stop_times.txt": "trip_id,arrival_time,departure_time,stop_id,stop_sequence\n" +
		"T1,06:00:00,06:00:00,S1,1\nT1,06:05:00,06:05:00,S2,2\nT1,06:10:00,06:10:00,S3,3\n" +
		"T_LOOP,07:00:00,07:00:00,S1,0\nT_LOOP,07:05:00,07:05:00,S2,1\nT_LOOP,07:10:00,07:10:00,S1,2\n" +
		"T_FREQ_BLANK,06:00:00,06:00:00,S1,1\nT_FREQ_BLANK,06:05:00,06:05:00,S2,2\n" +
		"T_FREQ0,06:00:00,06:00:00,S1,1\nT_FREQ0,06:05:00,06:05:00,S2,2\n" +
		"T_FREQ1,06:00:00,06:00:00,S1,1\nT_FREQ1,06:05:00,06:05:00,S2,2\n" +
		"T_UNTIMED,08:00:00,08:00:00,S1,1\nT_UNTIMED,,,S2,2\nT_UNTIMED,08:10:00,08:10:00,S3,3\n" +
		"T_SHORT,5:30:00,5:30:00,S1,1\nT_SHORT,5:35:00,5:35:00,S2,2\n",
	"frequencies.txt": "trip_id,start_time,end_time,headway_secs,exact_times\n" +
		"T_FREQ_BLANK,06:00:00,09:00:00,600,\nT_FREQ0,06:00:00,09:00:00,600,0\nT_FREQ1,06:00:00,07:00:00,600,1\n",
}

func zipOf(t testing.TB, files map[string]string) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(files[n])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// backends loads the same zip through every Static implementation.
func backends(t testing.TB, zipData []byte) map[string]Static {
	t.Helper()
	md, err := LoadMetadataFromBytes(zipData, false)
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	ix, err := LoadStaticIndex(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	return map[string]Static{"metadata": md, "index": ix}
}

const fxNow = uint64(1793094553)

var fxT0 = int64(fxNow + 600)

func fxHeader(ts uint64) *rt.FeedHeader {
	return &rt.FeedHeader{GtfsRealtimeVersion: proto.String("2.0"), Incrementality: rt.FeedHeader_FULL_DATASET.Enum(), Timestamp: proto.Uint64(ts)}
}

func fxFeed(es ...*rt.FeedEntity) *rt.FeedMessage {
	return &rt.FeedMessage{Header: fxHeader(fxNow), Entity: es}
}

func fxTU(trip, start string, sr *rt.TripDescriptor_ScheduleRelationship, stus ...*rt.TripUpdate_StopTimeUpdate) *rt.FeedEntity {
	td := &rt.TripDescriptor{TripId: proto.String(trip), RouteId: proto.String("R1"), StartDate: proto.String("20261027"), ScheduleRelationship: sr}
	if start != "" {
		td.StartTime = proto.String(start)
	}
	return &rt.FeedEntity{Id: proto.String("e"), TripUpdate: &rt.TripUpdate{Trip: td, Timestamp: proto.Uint64(fxNow),
		Vehicle: &rt.VehicleDescriptor{Id: proto.String("V-e")}, StopTimeUpdate: stus}}
}

func fxSTU(seq *uint32, stop string, arr, dep *rt.TripUpdate_StopTimeEvent, sr rt.TripUpdate_StopTimeUpdate_ScheduleRelationship) *rt.TripUpdate_StopTimeUpdate {
	s := &rt.TripUpdate_StopTimeUpdate{StopSequence: seq, Arrival: arr, Departure: dep, ScheduleRelationship: sr.Enum()}
	if stop != "" {
		s.StopId = proto.String(stop)
	}
	return s
}

func fxAt(t int64) *rt.TripUpdate_StopTimeEvent {
	return &rt.TripUpdate_StopTimeEvent{Time: proto.Int64(t)}
}

func fxAlert(ie ...*rt.EntitySelector) *rt.FeedEntity {
	return &rt.FeedEntity{Id: proto.String("a"), Alert: &rt.Alert{InformedEntity: ie,
		HeaderText: &rt.TranslatedString{Translation: []*rt.TranslatedString_Translation{{Text: proto.String("x")}}}}}
}

func fxVP(id, trip string, ts uint64, pos *rt.Position) *rt.FeedEntity {
	return &rt.FeedEntity{Id: proto.String(id), Vehicle: &rt.VehiclePosition{Vehicle: &rt.VehicleDescriptor{Id: proto.String(id)}, Timestamp: proto.Uint64(ts), Position: pos,
		Trip: &rt.TripDescriptor{TripId: proto.String(trip), RouteId: proto.String("R9")}}}
}

func seqp(x uint32) *uint32 { return &x }

type canonicalCase struct {
	name string
	feed *rt.FeedMessage
	prev *PrevHeader
	// want is the canonical MobilityData verdict unless the comment says the gap is deferred.
	want map[RuleID]int
}

func canonicalCases() []canonicalCase {
	S := rt.TripDescriptor_SCHEDULED.Enum()
	sch, skp := rt.TripUpdate_StopTimeUpdate_SCHEDULED, rt.TripUpdate_StopTimeUpdate_SKIPPED
	pos := &rt.Position{Latitude: proto.Float32(42.71), Longitude: proto.Float32(23.31)}
	vpOlder := &rt.FeedMessage{Header: fxHeader(fxNow - 1), Entity: []*rt.FeedEntity{fxVP("v", "T1", fxNow-1, pos), fxVP("v2", "T1", fxNow-1, pos)}}
	return []canonicalCase{
		{name: "0 clean trip T1", feed: fxFeed(fxTU("T1", "06:00:00", S, fxSTU(seqp(1), "S1", fxAt(fxT0), fxAt(fxT0), sch), fxSTU(seqp(2), "S2", fxAt(fxT0+300), fxAt(fxT0+300), sch))),
			want: map[RuleID]int{}},
		{name: "1 on-time stop sent as arrival delay 0 with no time", feed: fxFeed(fxTU("T1", "", S, fxSTU(seqp(1), "S1", &rt.TripUpdate_StopTimeEvent{Delay: proto.Int32(0)}, nil, sch))),
			want: map[RuleID]int{}},
		{name: "2 skipped stop carrying an empty arrival", feed: fxFeed(fxTU("T1", "", S, fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch), fxSTU(seqp(2), "S2", &rt.TripUpdate_StopTimeEvent{}, nil, skp))),
			want: map[RuleID]int{}},
		{name: "3 skipped stop by sequence only at an untimed GTFS stop", feed: fxFeed(fxTU("T_UNTIMED", "", S, fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch), fxSTU(seqp(2), "", nil, nil, skp))),
			want: map[RuleID]int{}},
		{name: "4 loop trip from stop_sequence 0", feed: fxFeed(fxTU("T_LOOP", "", S, fxSTU(seqp(0), "S1", fxAt(fxT0), nil, sch), fxSTU(seqp(1), "S2", fxAt(fxT0+300), nil, sch))),
			want: map[RuleID]int{}},
		{name: "5 stop_sequence 0 with no stop_id", feed: fxFeed(fxTU("T_LOOP", "", S, fxSTU(seqp(0), "", fxAt(fxT0), nil, sch))),
			want: map[RuleID]int{}},
		{name: "6 exact_times 0 trip with schedule_relationship unset", feed: fxFeed(fxTU("T_FREQ0", "06:10:00", nil, fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch))),
			want: map[RuleID]int{"W009": 1}},
		{name: "7 blank exact_times trip starting 06:10:00", feed: fxFeed(fxTU("T_FREQ_BLANK", "06:10:00", rt.TripDescriptor_UNSCHEDULED.Enum(), fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch))),
			want: map[RuleID]int{}},
		{name: "8 GTFS first arrival 5:30:00 and start_time 05:30:00", feed: fxFeed(fxTU("T_SHORT", "05:30:00", S, fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch))),
			want: map[RuleID]int{}},
		{name: "9 unknown SCHEDULED trip", feed: fxFeed(fxTU("T_GONE", "", S, fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch), fxSTU(seqp(2), "S2", fxAt(fxT0+60), nil, sch))),
			want: map[RuleID]int{"E003": 1}},
		{name: "10 unknown ADDED trip", feed: fxFeed(fxTU("T_NEW", "", rt.TripDescriptor_ADDED.Enum(), fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch))),
			want: map[RuleID]int{}},
		{name: "11 unknown trip with equal times at stops 1 and 2", feed: fxFeed(fxTU("T_GONE", "", S, fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch), fxSTU(seqp(2), "S2", fxAt(fxT0), nil, sch))),
			want: map[RuleID]int{"E003": 1, "E022": 1}},
		{name: "12 exact_times 1 trip off the headway grid", feed: fxFeed(fxTU("T_FREQ1", "06:05:00", S, fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch))),
			want: map[RuleID]int{}},
		{name: "13 alert for route_type 0 only", feed: fxFeed(fxAlert(&rt.EntitySelector{RouteType: proto.Int32(0)})),
			want: map[RuleID]int{}},
		{name: "14 alert for trip.route_id only", feed: fxFeed(fxAlert(&rt.EntitySelector{Trip: &rt.TripDescriptor{RouteId: proto.String("R1")}})),
			want: map[RuleID]int{}},
		{name: "15 alert route_id R2 with trip T1 that runs on R1", feed: fxFeed(fxAlert(&rt.EntitySelector{RouteId: proto.String("R2"), Trip: &rt.TripDescriptor{TripId: proto.String("T1")}})),
			want: map[RuleID]int{"E030": 1}},
		{name: "16 alert whose second selector is empty", feed: fxFeed(fxAlert(&rt.EntitySelector{RouteId: proto.String("R1")}, &rt.EntitySelector{})),
			want: map[RuleID]int{"E033": 1}},
		{name: "17 vehicle with no position", feed: fxFeed(fxVP("v", "T1", fxNow, nil)),
			want: map[RuleID]int{}},
		// Canonical gives E003 and E004; trip checks on VehiclePositions are deferred.
		{name: "18 vehicle on unknown trip and route", feed: fxFeed(fxVP("v", "T_GONE", fxNow, pos)),
			want: map[RuleID]int{}},
		{name: "19 version 2.0 header with no incrementality", feed: &rt.FeedMessage{Header: &rt.FeedHeader{GtfsRealtimeVersion: proto.String("2.0"), Timestamp: proto.Uint64(fxNow)}},
			want: map[RuleID]int{"E049": 1}},
		{name: "20 arrival time sent as 0", feed: fxFeed(fxTU("T1", "", S, fxSTU(seqp(1), "S1", fxAt(0), nil, sch))),
			want: map[RuleID]int{}},
		{name: "21 VehiclePositions with its own previous header after a TripUpdates build in the same second", feed: &rt.FeedMessage{Header: fxHeader(fxNow), Entity: []*rt.FeedEntity{fxVP("v", "T1", fxNow, pos), fxVP("v2", "T1", fxNow, pos)}},
			want: map[RuleID]int{}},
		{name: "22 VehiclePositions built a second before a TripUpdates build", feed: vpOlder,
			want: map[RuleID]int{}},
		{name: "23 same feed rebuilt in the same second with one more entity", feed: &rt.FeedMessage{Header: fxHeader(fxNow), Entity: []*rt.FeedEntity{fxVP("v", "T1", fxNow, pos), fxVP("v2", "T1", fxNow, pos)}},
			prev: &PrevHeader{Timestamp: fxNow, EntityCount: 1}, want: map[RuleID]int{"E017": 1}},
		{name: "24 known trip with equal times at stops 2 and 3", feed: fxFeed(fxTU("T1", "", S, fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch), fxSTU(seqp(2), "S2", fxAt(fxT0+300), nil, sch), fxSTU(seqp(3), "S3", fxAt(fxT0+300), nil, sch))),
			want: map[RuleID]int{"E022": 1}},
		{name: "25 known trip with an unknown stop_sequence before equal times", feed: fxFeed(fxTU("T1", "", S, fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch), fxSTU(seqp(7), "S2", fxAt(fxT0+300), nil, sch), fxSTU(seqp(3), "S3", fxAt(fxT0+300), nil, sch))),
			want: map[RuleID]int{"E022": 1, "E051": 1}},
		{name: "26 vehicle 1,400 m east of the easternmost stop", feed: fxFeed(fxVP("v", "T1", fxNow, &rt.Position{Latitude: proto.Float32(42.71), Longitude: proto.Float32(23.347)})),
			want: map[RuleID]int{}},
		{name: "27 vehicle 1,800 m east of the easternmost stop", feed: fxFeed(fxVP("v", "T1", fxNow, &rt.Position{Latitude: proto.Float32(42.71), Longitude: proto.Float32(23.352)})),
			want: map[RuleID]int{"E028": 1}},
		{name: "29 blank exact_times trip sent as SCHEDULED", feed: fxFeed(fxTU("T_FREQ_BLANK", "06:10:00", S, fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch))),
			want: map[RuleID]int{"E013": 1}},
		{name: "30 scheduled stop with an empty arrival", feed: fxFeed(fxTU("T1", "", S, fxSTU(seqp(1), "S1", &rt.TripUpdate_StopTimeEvent{}, nil, sch))),
			want: map[RuleID]int{"E044": 1}},
		{name: "31 loop trip sent by stop_id only", feed: fxFeed(fxTU("T_LOOP", "", S, fxSTU(nil, "S1", fxAt(fxT0), nil, sch), fxSTU(nil, "S2", fxAt(fxT0+300), nil, sch))),
			want: map[RuleID]int{"E009": 1}},
		{name: "32 known trip sent in reverse stop_sequence order", feed: fxFeed(fxTU("T1", "", S, fxSTU(seqp(3), "S3", fxAt(fxT0), nil, sch), fxSTU(seqp(2), "S2", fxAt(fxT0+60), nil, sch), fxSTU(seqp(1), "S1", fxAt(fxT0+120), nil, sch))),
			want: map[RuleID]int{"E002": 1}},
		{name: "33 update with neither stop_sequence nor stop_id", feed: fxFeed(fxTU("T1", "", S, fxSTU(nil, "", fxAt(fxT0), nil, sch))),
			want: map[RuleID]int{"E040": 1}},
		{name: "34 vehicle at latitude 91", feed: fxFeed(fxVP("v", "T1", fxNow, &rt.Position{Latitude: proto.Float32(91), Longitude: proto.Float32(23.31)})),
			want: map[RuleID]int{"E026": 1}},
		{name: "35 alert route_id R1 with trip route_id R2 and no trip_id", feed: fxFeed(fxAlert(&rt.EntitySelector{RouteId: proto.String("R1"), Trip: &rt.TripDescriptor{RouteId: proto.String("R2")}})),
			want: map[RuleID]int{"E031": 1}},
		{name: "36 alert trip T1 given route R2", feed: fxFeed(fxAlert(&rt.EntitySelector{Trip: &rt.TripDescriptor{TripId: proto.String("T1"), RouteId: proto.String("R2")}})),
			want: map[RuleID]int{"E035": 1}},
		{name: "37 alert trip T1 given route R9 that is not in GTFS", feed: fxFeed(fxAlert(&rt.EntitySelector{Trip: &rt.TripDescriptor{TripId: proto.String("T1"), RouteId: proto.String("R9")}})),
			want: map[RuleID]int{}},
		{name: "28 untimed GTFS stop sent with arrival and departure delays only", feed: fxFeed(fxTU("T_UNTIMED", "", S, fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch),
			fxSTU(seqp(2), "S2", &rt.TripUpdate_StopTimeEvent{Delay: proto.Int32(60)}, &rt.TripUpdate_StopTimeEvent{Delay: proto.Int32(60)}, sch))),
			want: map[RuleID]int{"E046": 2}},
	}
}

func countsString(m map[RuleID]int) string {
	keys := make([]string, 0, len(m))
	for k, n := range m {
		keys = append(keys, fmt.Sprintf("%s x%d", k, n))
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "nothing"
	}
	return strings.Join(keys, " ")
}

func TestCanonicalVerdicts(t *testing.T) {
	zipData := zipOf(t, fixtureFiles)
	opts := Options{Now: time.Unix(int64(fxNow), 0)}
	for name, static := range backends(t, zipData) {
		for _, tc := range canonicalCases() {
			res := ValidateFeed(context.Background(), static, tc.feed, tc.prev, opts)
			if got, want := countsString(res.Counts), countsString(tc.want); got != want {
				t.Errorf("%s / %s: got %s, want %s", name, tc.name, got, want)
			}
		}
	}
}

// notCovered answers NotCovered for every id, as a publisher does before its first release load.
type notCovered struct{}

func (notCovered) Trip(string, string) (TripInfo, Presence) { return TripInfo{}, NotCovered }
func (notCovered) Stop(string, string) (int, Presence)      { return 0, NotCovered }
func (notCovered) Route(string) Presence                    { return NotCovered }
func (notCovered) Agency(string) Presence                   { return NotCovered }
func (notCovered) ServiceArea() (Rect, bool)                { return Rect{}, false }

func TestNotCoveredSkipsStaticChecks(t *testing.T) {
	S := rt.TripDescriptor_SCHEDULED.Enum()
	sch := rt.TripUpdate_StopTimeUpdate_SCHEDULED
	feed := fxFeed(fxTU("T_GONE", "06:00:00", S, fxSTU(seqp(1), "S_GONE", fxAt(fxT0), nil, sch)))
	res := ValidateFeed(context.Background(), notCovered{}, feed, nil, Options{Now: time.Unix(int64(fxNow), 0)})
	if got := countsString(res.Counts); got != "nothing" {
		t.Errorf("got %s, want nothing", got)
	}
}

func TestValidateFeedConcurrentFeedsShareNothing(t *testing.T) {
	static := backends(t, zipOf(t, fixtureFiles))["metadata"]
	opts := Options{Now: time.Unix(int64(fxNow), 0)}
	pos := &rt.Position{Latitude: proto.Float32(42.71), Longitude: proto.Float32(23.31)}
	sch := rt.TripUpdate_StopTimeUpdate_SCHEDULED
	tu := fxFeed(fxTU("T1", "", rt.TripDescriptor_SCHEDULED.Enum(), fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch)))
	vp := &rt.FeedMessage{Header: fxHeader(fxNow), Entity: []*rt.FeedEntity{fxVP("v", "T1", fxNow, pos), fxVP("v2", "T1", fxNow, pos)}}
	errs := make(chan string, 200)
	run := func(feed *rt.FeedMessage) {
		var prev *PrevHeader
		for i := 0; i < 100; i++ {
			res := ValidateFeed(context.Background(), static, feed, prev, opts)
			if !res.Valid {
				errs <- countsString(res.Counts)
			}
			h := HeaderOf(feed)
			prev = &h
		}
	}
	done := make(chan struct{})
	go func() { run(tu); done <- struct{}{} }()
	run(vp)
	<-done
	close(errs)
	for e := range errs {
		t.Errorf("clean build failed: %s", e)
	}
}

// loaders loads a zip through every Static implementation and returns each error.
func loaders(zipData []byte) map[string]error {
	_, mdErr := LoadMetadataFromBytes(zipData, false)
	_, ixErr := LoadStaticIndex(bytes.NewReader(zipData), int64(len(zipData)))
	return map[string]error{"metadata": mdErr, "index": ixErr}
}

func withFile(name, body string) map[string]string {
	f := make(map[string]string, len(fixtureFiles))
	for k, v := range fixtureFiles {
		f[k] = v
	}
	if body == "" {
		delete(f, name)
	} else {
		f[name] = body
	}
	return f
}

func TestStrictLoad(t *testing.T) {
	stops := fixtureFiles["stops.txt"]
	cases := []struct {
		name    string
		files   map[string]string
		wantErr string
	}{
		{"stops.txt with a byte order mark", withFile("stops.txt", "\ufeff"+stops), ""},
		{"stops.txt missing", withFile("stops.txt", ""), "stops.txt is missing"},
		{"trips.txt missing", withFile("trips.txt", ""), "trips.txt is missing"},
		{"stops.txt header only", withFile("stops.txt", "stop_id,stop_name,stop_lat,stop_lon\n"), "stops.txt has no rows"},
		{"stop_times.txt without stop_sequence", withFile("stop_times.txt", "trip_id,arrival_time,departure_time,stop_id\nT1,06:00:00,06:00:00,S1\n"), "stop_times.txt has no stop_sequence column"},
		{"stops.txt with an unclosed quote", withFile("stops.txt", strings.Replace(stops, "S2,b,", "S2,\"b,", 1)), "stops.txt"},
	}
	for _, tc := range cases {
		for name, err := range loaders(zipOf(t, tc.files)) {
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("%s / %s: unexpected error %v", name, tc.name, err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Errorf("%s / %s: got error %v, want one containing %q", name, tc.name, err, tc.wantErr)
			}
		}
	}
	md, err := LoadMetadataFromBytes(zipOf(t, withFile("stops.txt", "\ufeff"+stops)), false)
	if err != nil || len(md.StopIDs) != 5 {
		t.Errorf("byte order mark: got %d stops and error %v, want 5 stops", len(md.StopIDs), err)
	}
}

func TestCanonicalCleanTripAfterByteOrderMark(t *testing.T) {
	zipData := zipOf(t, withFile("stops.txt", "\ufeff"+fixtureFiles["stops.txt"]))
	sch := rt.TripUpdate_StopTimeUpdate_SCHEDULED
	feed := fxFeed(fxTU("T1", "", rt.TripDescriptor_SCHEDULED.Enum(), fxSTU(seqp(1), "S1", fxAt(fxT0), nil, sch), fxSTU(seqp(2), "S2", fxAt(fxT0+300), nil, sch)))
	for name, static := range backends(t, zipData) {
		res := ValidateFeed(context.Background(), static, feed, nil, Options{Now: time.Unix(int64(fxNow), 0)})
		if got := countsString(res.Counts); got != "nothing" {
			t.Errorf("%s: got %s, want nothing", name, got)
		}
	}
}

func TestParseClock(t *testing.T) {
	for in, want := range map[string]int32{"5:30:00": 19800, "05:30:00": 19800, "25:01:02": 90062, "100:00:00": 360000, "": -1, "5:3:00": -1, "05:60:00": -1, "05:30": -1, "x5:30:00": -1, "05:30:00 ": -1} {
		got, ok := parseClock(in)
		if !ok {
			got = -1
		}
		if got != want {
			t.Errorf("parseClock(%q) = %d, want %d", in, got, want)
		}
	}
	for in, want := range map[int32]string{19800: "05:30:00", 90062: "25:01:02", 0: "00:00:00", 360000: "100:00:00"} {
		if got := formatClock(in); got != want {
			t.Errorf("formatClock(%d) = %q, want %q", in, got, want)
		}
	}
}
