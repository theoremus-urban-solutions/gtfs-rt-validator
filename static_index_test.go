package validator

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

// sofiaZip is a real release for the parity test and benchmarks; tests skip when it is not set.
func sofiaZip(tb testing.TB) string {
	p := os.Getenv("GTFS_RT_SOFIA_ZIP")
	if p == "" {
		tb.Skip("GTFS_RT_SOFIA_ZIP not set")
	}
	return p
}

func checkParity(t *testing.T, md *Metadata, ix *StaticIndex) {
	t.Helper()
	trips, stops := 0, 0
	stopIDs := map[string]struct{}{"NO_SUCH_STOP": {}}
	for id := range md.StopIDs {
		stopIDs[id] = struct{}{}
	}
	for id := range md.TripsByTripID {
		want, wp := md.Trip(id, "")
		got, gp := ix.Trip(id, "")
		if wp != gp || want.RouteID != got.RouteID || want.DirectionID != got.DirectionID || want.FirstArrival != got.FirstArrival ||
			want.Frequency != got.Frequency || want.RepeatsStop != got.RepeatsStop || want.NumStops() != got.NumStops() {
			t.Fatalf("trip %s: metadata %+v %v, index %+v %v", id, want, wp, got, gp)
		}
		for i := 0; i < want.NumStops(); i++ {
			if w, g := want.StopAt(i), got.StopAt(i); w != g {
				t.Fatalf("trip %s stop %d: metadata %+v, index %+v", id, i, w, g)
			}
			stopIDs[want.StopAt(i).StopID] = struct{}{}
			stops++
		}
		trips++
	}
	for _, id := range []string{"", "NO_SUCH_TRIP"} {
		if _, p := ix.Trip(id, ""); p != Absent {
			t.Errorf("trip %q: got %v, want Absent", id, p)
		}
	}
	for id := range stopIDs {
		wlt, wp := md.Stop(id, "")
		glt, gp := ix.Stop(id, "")
		if wlt != glt || wp != gp {
			t.Fatalf("stop %s: metadata %d %v, index %d %v", id, wlt, wp, glt, gp)
		}
	}
	routeIDs := map[string]struct{}{"NO_SUCH_ROUTE": {}}
	for id := range md.RouteIDs {
		routeIDs[id] = struct{}{}
	}
	for _, tr := range md.TripsByTripID {
		routeIDs[tr.RouteID] = struct{}{}
	}
	for id := range routeIDs {
		if w, g := md.Route(id), ix.Route(id); w != g {
			t.Fatalf("route %s: metadata %v, index %v", id, w, g)
		}
	}
	for _, id := range []string{"default", "A", "NO_SUCH_AGENCY"} {
		if w, g := md.Agency(id), ix.Agency(id); w != g {
			t.Fatalf("agency %s: metadata %v, index %v", id, w, g)
		}
	}
	for id := range md.AgencyIDs {
		if ix.Agency(id) != Present {
			t.Fatalf("agency %s missing from index", id)
		}
	}
	wa, _ := md.ServiceArea()
	ga, _ := ix.ServiceArea()
	if wa != ga {
		t.Fatalf("service area: metadata %+v, index %+v", wa, ga)
	}
	t.Logf("%d trips, %d stop_times rows, %d stops and %d routes agree; service area %+v", trips, stops, len(stopIDs), len(routeIDs), ga)
}

func TestStaticIndexParityFixture(t *testing.T) {
	zipData := zipOf(t, fixtureFiles)
	md, err := LoadMetadataFromBytes(zipData, false)
	if err != nil {
		t.Fatal(err)
	}
	ix, err := LoadStaticIndex(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatal(err)
	}
	checkParity(t, md, ix)
}

func TestStaticIndexParitySofia(t *testing.T) {
	path := sofiaZip(t)
	md, err := LoadMetadata(path, false)
	if err != nil {
		t.Fatal(err)
	}
	ix, err := LoadStaticIndexPath(path)
	if err != nil {
		t.Fatal(err)
	}
	checkParity(t, md, ix)
}

func TestStaticIndexDirectory(t *testing.T) {
	dir := t.TempDir()
	for name, body := range fixtureFiles {
		if err := os.WriteFile(dir+"/"+name, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ix, err := LoadStaticIndexPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, p := ix.Trip("T1", ""); p != Present {
		t.Errorf("T1: got %v, want Present", p)
	}
}

// TestStaticIndexSofiaLoad holds the load to the publisher's budget: 1.5 s and 12 MiB kept.
func TestStaticIndexSofiaLoad(t *testing.T) {
	path := sofiaZip(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	ix, err := LoadStaticIndex(bytes.NewReader(data), int64(len(data)))
	took := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	kept := float64(after.HeapAlloc-before.HeapAlloc) / (1 << 20)
	t.Logf("Sofia: %d trips, %d stop_times rows, %d stops; load %v, kept %.1f MiB, allocated %.0f MiB during load",
		len(ix.tripRecs), len(ix.stopRecs), len(ix.stopNames), took.Round(time.Millisecond), kept, float64(after.TotalAlloc-before.TotalAlloc)/(1<<20))
	if took > 1500*time.Millisecond && !raceEnabled {
		t.Errorf("load took %v, budget 1.5 s", took)
	}
	if kept > 12 {
		t.Errorf("index keeps %.1f MiB, budget 12 MiB", kept)
	}
	runtime.KeepAlive(ix)
}

func BenchmarkLoadStaticIndex(b *testing.B) {
	data, err := os.ReadFile(sofiaZip(b))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := LoadStaticIndex(bytes.NewReader(data), int64(len(data))); err != nil {
			b.Fatal(err)
		}
	}
}

// cleanTripUpdates builds a clean TripUpdates feed of whole Sofia trips with about want stop updates.
func cleanTripUpdates(tb testing.TB, ix *StaticIndex, want int, now time.Time) *rt.FeedMessage {
	tb.Helper()
	feed := &rt.FeedMessage{Header: &rt.FeedHeader{GtfsRealtimeVersion: proto.String("2.0"), Incrementality: rt.FeedHeader_FULL_DATASET.Enum(), Timestamp: proto.Uint64(uint64(now.Unix()))}}
	n := 0
	for id := range ix.trips {
		info, _ := ix.Trip(id, "")
		if info.Frequency != NotFrequency || info.RepeatsStop || info.FirstArrival < 0 || info.NumStops() < 2 {
			continue
		}
		td := &rt.TripDescriptor{TripId: proto.String(id), RouteId: proto.String(info.RouteID), StartDate: proto.String(now.Format("20060102")),
			StartTime: proto.String(formatClock(info.FirstArrival)), ScheduleRelationship: rt.TripDescriptor_SCHEDULED.Enum()}
		if info.DirectionID >= 0 {
			td.DirectionId = proto.Uint32(uint32(info.DirectionID))
		}
		tu := &rt.TripUpdate{Trip: td, Timestamp: proto.Uint64(uint64(now.Unix())), Vehicle: &rt.VehicleDescriptor{Id: proto.String("v" + id)}}
		t0 := now.Unix() + 600
		for i := 0; i < info.NumStops(); i++ {
			st := info.StopAt(i)
			tu.StopTimeUpdate = append(tu.StopTimeUpdate, &rt.TripUpdate_StopTimeUpdate{
				StopSequence: proto.Uint32(st.Sequence), StopId: proto.String(st.StopID),
				Arrival:              &rt.TripUpdate_StopTimeEvent{Delay: proto.Int32(0), Time: proto.Int64(t0 + int64(i)*60)},
				ScheduleRelationship: rt.TripUpdate_StopTimeUpdate_SCHEDULED.Enum(),
			})
		}
		feed.Entity = append(feed.Entity, &rt.FeedEntity{Id: proto.String(id), TripUpdate: tu})
		if n += info.NumStops(); n >= want {
			break
		}
	}
	return feed
}

func BenchmarkValidateTripUpdates(b *testing.B) {
	ix, err := LoadStaticIndexPath(sofiaZip(b))
	if err != nil {
		b.Fatal(err)
	}
	now := time.Now()
	feed := cleanTripUpdates(b, ix, 7000, now)
	opts := Options{Now: now}
	if res := ValidateFeed(context.Background(), ix, feed, nil, opts); !res.Valid || len(res.Counts) != 0 {
		b.Fatalf("feed is not clean: %v", res.Counts)
	}
	stus := 0
	for _, e := range feed.Entity {
		stus += len(e.TripUpdate.StopTimeUpdate)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateFeed(context.Background(), ix, feed, nil, opts)
	}
	b.ReportMetric(float64(stus), "stop_updates")
}
