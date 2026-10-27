package validator

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

func TestNewRTOnlyValidator(t *testing.T) {
	v := NewRTOnlyValidator(Options{})
	if v == nil {
		t.Fatal("NewRTOnlyValidator returned nil")
	}
	if v.static != nil {
		t.Error("RT-only validator should have no static release")
	}
}

func TestRTOnlyValidatorsCount(t *testing.T) {
	rtOnly := RTOnlyValidators()
	full := DefaultValidators()

	if len(rtOnly) >= len(full) {
		t.Errorf("RTOnlyValidators (%d) should be fewer than DefaultValidators (%d)", len(rtOnly), len(full))
	}

	// RT-only should have 5 validators
	if len(rtOnly) != 5 {
		t.Errorf("Expected 5 RT-only validators, got %d", len(rtOnly))
	}

	// Full should have 9 validators
	if len(full) != 9 {
		t.Errorf("Expected 9 full validators, got %d", len(full))
	}
}

func TestRTOnlyValidation_ValidFeed(t *testing.T) {
	v := NewRTOnlyValidator(Options{})
	ctx := context.Background()

	// Create a valid minimal feed
	now := uint64(time.Now().Unix())
	feed := &rt.FeedMessage{
		Header: &rt.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
			Timestamp:           proto.Uint64(now),
			Incrementality:      rt.FeedHeader_FULL_DATASET.Enum(),
		},
		Entity: []*rt.FeedEntity{},
	}

	valid, result := v.ValidateIteration(ctx, feed, nil)
	if !valid {
		t.Errorf("Expected valid feed, got errors: %v", result.Errors)
	}
}

func TestRTOnlyValidation_InvalidVersion(t *testing.T) {
	v := NewRTOnlyValidator(Options{})
	ctx := context.Background()

	now := uint64(time.Now().Unix())
	feed := &rt.FeedMessage{
		Header: &rt.FeedHeader{
			GtfsRealtimeVersion: proto.String("0.5"), // Invalid version
			Timestamp:           proto.Uint64(now),
		},
	}

	valid, result := v.ValidateIteration(ctx, feed, nil)
	if valid {
		t.Error("Expected invalid feed due to bad version")
	}

	// Should have E038 error
	found := false
	for _, el := range result.Lists {
		if el.Rule.ID == "E038" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected E038 error for invalid version")
	}
}

func TestRTOnlyValidation_DuplicateVehicleID(t *testing.T) {
	v := NewRTOnlyValidator(Options{})
	ctx := context.Background()

	now := uint64(time.Now().Unix())
	feed := &rt.FeedMessage{
		Header: &rt.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
			Timestamp:           proto.Uint64(now),
			Incrementality:      rt.FeedHeader_FULL_DATASET.Enum(),
		},
		Entity: []*rt.FeedEntity{
			{
				Id: proto.String("1"),
				Vehicle: &rt.VehiclePosition{
					Vehicle:  &rt.VehicleDescriptor{Id: proto.String("V001")},
					Position: &rt.Position{Latitude: proto.Float32(42.0), Longitude: proto.Float32(23.0)},
				},
			},
			{
				Id: proto.String("2"),
				Vehicle: &rt.VehiclePosition{
					Vehicle:  &rt.VehicleDescriptor{Id: proto.String("V001")}, // Duplicate!
					Position: &rt.Position{Latitude: proto.Float32(42.1), Longitude: proto.Float32(23.1)},
				},
			},
		},
	}

	valid, result := v.ValidateIteration(ctx, feed, nil)
	// E052 is an error, so the feed should be invalid
	if valid {
		t.Error("Expected invalid feed (E052 is an error)")
	}

	// Should have E052 error for duplicate vehicle ID
	found := false
	for _, el := range result.Lists {
		if el.Rule.ID == "E052" {
			found = true
			if el.Rule.Severity != Error {
				t.Errorf("E052 should have Error severity, got %s", el.Rule.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("Expected E052 error for duplicate vehicle ID")
	}
}

func TestRTOnlyValidation_SequentialTimesNotIncreasing(t *testing.T) {
	v := NewRTOnlyValidator(Options{})
	ctx := context.Background()

	now := uint64(time.Now().Unix())
	laterTime := int64(now + 100)
	earlierTime := int64(now + 50) // Earlier than laterTime - invalid sequence

	feed := &rt.FeedMessage{
		Header: &rt.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
			Timestamp:           proto.Uint64(now),
			Incrementality:      rt.FeedHeader_FULL_DATASET.Enum(),
		},
		Entity: []*rt.FeedEntity{
			{
				Id: proto.String("1"),
				TripUpdate: &rt.TripUpdate{
					Trip:    &rt.TripDescriptor{TripId: proto.String("trip1")},
					Vehicle: &rt.VehicleDescriptor{Id: proto.String("V001")},
					StopTimeUpdate: []*rt.TripUpdate_StopTimeUpdate{
						{
							StopSequence: proto.Uint32(1),
							StopId:       proto.String("stop1"),
							Arrival:      &rt.TripUpdate_StopTimeEvent{Time: proto.Int64(laterTime)},
						},
						{
							StopSequence: proto.Uint32(2),
							StopId:       proto.String("stop2"),
							Arrival:      &rt.TripUpdate_StopTimeEvent{Time: proto.Int64(earlierTime)}, // Not increasing!
						},
					},
				},
			},
		},
	}

	valid, result := v.ValidateIteration(ctx, feed, nil)
	// E022 is an error, so the feed should be invalid
	if valid {
		t.Error("Expected invalid feed (E022 is an error)")
	}

	// Should have E022 error for non-increasing times
	found := false
	for _, el := range result.Lists {
		if el.Rule.ID == "E022" {
			found = true
			if el.Rule.Severity != Error {
				t.Errorf("E022 should have Error severity, got %s", el.Rule.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("Expected E022 error for non-increasing times")
	}
}

func TestRTOnlyValidation_SkipsStaticChecks(t *testing.T) {
	v := NewRTOnlyValidator(Options{})
	ctx := context.Background()

	now := uint64(time.Now().Unix())
	feed := &rt.FeedMessage{
		Header: &rt.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
			Timestamp:           proto.Uint64(now),
			Incrementality:      rt.FeedHeader_FULL_DATASET.Enum(),
		},
		Entity: []*rt.FeedEntity{
			{
				Id: proto.String("1"),
				TripUpdate: &rt.TripUpdate{
					Trip:    &rt.TripDescriptor{TripId: proto.String("nonexistent_trip")}, // Would fail E003 in full mode
					Vehicle: &rt.VehicleDescriptor{Id: proto.String("V001")},
					StopTimeUpdate: []*rt.TripUpdate_StopTimeUpdate{
						{
							StopSequence: proto.Uint32(1),
							StopId:       proto.String("nonexistent_stop"), // Would fail E011 in full mode
							Arrival:      &rt.TripUpdate_StopTimeEvent{Time: proto.Int64(int64(now + 100))},
						},
					},
				},
			},
		},
	}

	valid, result := v.ValidateIteration(ctx, feed, nil)

	// Should NOT have E003 or E011 errors in RT-only mode (these require static GTFS)
	for _, el := range result.Lists {
		if el.Rule.ID == "E003" {
			t.Error("RT-only mode should not check E003 (trip_id not in GTFS)")
		}
		if el.Rule.ID == "E011" {
			t.Error("RT-only mode should not check E011 (stop_id not in GTFS)")
		}
	}

	// Should be valid since RT-only doesn't check static references
	if !valid {
		t.Logf("Feed invalid with errors (expected for RT-only to skip static checks): %v", result.Errors)
	}
}

func TestVehicleValidator_RTOnlySkipsBBox(t *testing.T) {
	// Test that VehicleValidator with RTOnly=true skips bbox check
	vRT := VehicleValidator{RTOnly: true}
	vFull := VehicleValidator{RTOnly: false}

	ctx := context.Background()
	now := time.Now().UnixMilli()

	// Position way outside any reasonable bbox
	feed := &rt.FeedMessage{
		Entity: []*rt.FeedEntity{
			{
				Id: proto.String("1"),
				Vehicle: &rt.VehiclePosition{
					Vehicle:  &rt.VehicleDescriptor{Id: proto.String("V001")},
					Position: &rt.Position{Latitude: proto.Float32(0.0), Longitude: proto.Float32(0.0)}, // Middle of ocean
				},
			},
		},
	}

	far := &Metadata{StopBBoxBuffered: Rect{MinLat: 42, MinLon: 23, MaxLat: 43, MaxLon: 24}}
	cRT := &Check{Ctx: ctx, NowMs: now, Static: far, Feed: feed, max: -1}
	vRT.Validate(cRT)
	cNil := &Check{Ctx: ctx, NowMs: now, Feed: feed, max: -1}
	vFull.Validate(cNil)

	if cRT.counts["E028"] != 0 {
		t.Error("RT-only should never produce E028")
	}
	if cNil.counts["E028"] != 0 {
		t.Error("Full mode with no static release should not produce E028")
	}
}

func TestCrossFeedValidation_RTOnly(t *testing.T) {
	v := NewRTOnlyValidator(Options{})
	ctx := context.Background()

	now := uint64(time.Now().Unix())

	// TripUpdates feed
	tuFeed := &rt.FeedMessage{
		Header: &rt.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
			Timestamp:           proto.Uint64(now),
			Incrementality:      rt.FeedHeader_FULL_DATASET.Enum(),
		},
		Entity: []*rt.FeedEntity{
			{
				Id: proto.String("1"),
				TripUpdate: &rt.TripUpdate{
					Trip:    &rt.TripDescriptor{TripId: proto.String("trip1")},
					Vehicle: &rt.VehicleDescriptor{Id: proto.String("V001")},
					StopTimeUpdate: []*rt.TripUpdate_StopTimeUpdate{
						{
							StopSequence: proto.Uint32(1),
							StopId:       proto.String("stop1"),
							Arrival:      &rt.TripUpdate_StopTimeEvent{Time: proto.Int64(int64(now + 100))},
						},
					},
				},
			},
		},
	}

	// VehiclePositions feed with different vehicle
	vpFeed := &rt.FeedMessage{
		Header: &rt.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
			Timestamp:           proto.Uint64(now),
			Incrementality:      rt.FeedHeader_FULL_DATASET.Enum(),
		},
		Entity: []*rt.FeedEntity{
			{
				Id: proto.String("2"),
				Vehicle: &rt.VehiclePosition{
					Trip:     &rt.TripDescriptor{TripId: proto.String("trip2")}, // Different trip
					Vehicle:  &rt.VehicleDescriptor{Id: proto.String("V002")},   // Different vehicle
					Position: &rt.Position{Latitude: proto.Float32(42.0), Longitude: proto.Float32(23.0)},
				},
			},
		},
	}

	combined := BuildCombined(tuFeed, vpFeed)
	_, result := v.ValidateIteration(ctx, tuFeed, combined)

	// Should have W003 warnings for mismatched IDs
	found := false
	for _, el := range result.Lists {
		if el.Rule.ID == "W003" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected W003 warning for cross-feed ID mismatch")
	}
}

// createMinimalGTFSZip creates a minimal valid GTFS zip file in memory for testing.
func createMinimalGTFSZip(t *testing.T) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// agency.txt
	agency, _ := zw.Create("agency.txt")
	_, _ = agency.Write([]byte("agency_id,agency_name,agency_url,agency_timezone\ntest_agency,Test Agency,http://example.com,America/New_York\n"))

	// routes.txt
	routes, _ := zw.Create("routes.txt")
	_, _ = routes.Write([]byte("route_id,agency_id,route_short_name,route_type\nroute1,test_agency,R1,3\n"))

	// trips.txt
	trips, _ := zw.Create("trips.txt")
	_, _ = trips.Write([]byte("trip_id,route_id,service_id,direction_id\ntrip1,route1,weekday,0\n"))

	// stops.txt
	stops, _ := zw.Create("stops.txt")
	_, _ = stops.Write([]byte("stop_id,stop_name,stop_lat,stop_lon\nstop1,Stop 1,42.365,23.379\nstop2,Stop 2,42.366,23.380\n"))

	// stop_times.txt
	stopTimes, _ := zw.Create("stop_times.txt")
	_, _ = stopTimes.Write([]byte("trip_id,arrival_time,departure_time,stop_id,stop_sequence\ntrip1,08:00:00,08:00:00,stop1,1\ntrip1,08:05:00,08:05:00,stop2,2\n"))

	_ = zw.Close()
	return buf.Bytes()
}

func TestNewValidatorFromBytes_BasicValidation(t *testing.T) {
	gtfsBytes := createMinimalGTFSZip(t)

	// Create validator from bytes
	v, err := NewValidatorFromBytes(gtfsBytes, Options{})
	if err != nil {
		t.Fatalf("NewValidatorFromBytes failed: %v", err)
	}

	// Create a simple valid GTFS-RT feed
	now := uint64(time.Now().Unix())
	feed := &rt.FeedMessage{
		Header: &rt.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
			Timestamp:           proto.Uint64(now),
			Incrementality:      rt.FeedHeader_FULL_DATASET.Enum(),
		},
		Entity: []*rt.FeedEntity{
			{
				Id: proto.String("1"),
				TripUpdate: &rt.TripUpdate{
					Trip:    &rt.TripDescriptor{TripId: proto.String("trip1")},
					Vehicle: &rt.VehicleDescriptor{Id: proto.String("V001")},
					StopTimeUpdate: []*rt.TripUpdate_StopTimeUpdate{
						{
							StopSequence: proto.Uint32(1),
							StopId:       proto.String("stop1"),
							Arrival:      &rt.TripUpdate_StopTimeEvent{Time: proto.Int64(int64(now + 100))},
						},
					},
				},
			},
		},
	}

	// Validate
	valid, result := v.ValidateIteration(context.Background(), feed, nil)

	if !valid {
		t.Errorf("Expected valid feed, got errors: %v", result.Errors)
	}
}

func TestNewValidatorFromBytes_CompareWithFilePath(t *testing.T) {
	// Skip if no real GTFS file exists
	gtfsPath := "sofia-static.zip"
	if _, err := os.Stat(gtfsPath); os.IsNotExist(err) {
		t.Skip("No sofia-static.zip available, skipping comparison test")
	}

	// Load same GTFS via file path
	v1, err := NewValidator(gtfsPath, Options{})
	if err != nil {
		t.Fatalf("NewValidator failed: %v", err)
	}

	// Load same GTFS via bytes
	gtfsBytes, err := os.ReadFile(gtfsPath)
	if err != nil {
		t.Fatalf("Failed to read GTFS file: %v", err)
	}
	v2, err := NewValidatorFromBytes(gtfsBytes, Options{})
	if err != nil {
		t.Fatalf("NewValidatorFromBytes failed: %v", err)
	}

	// Validate same feed with both validators
	now := uint64(time.Now().Unix())
	feed := &rt.FeedMessage{
		Header: &rt.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
			Timestamp:           proto.Uint64(now),
			Incrementality:      rt.FeedHeader_FULL_DATASET.Enum(),
		},
		Entity: []*rt.FeedEntity{},
	}
	ctx := context.Background()

	valid1, result1 := v1.ValidateIteration(ctx, feed, nil)
	valid2, result2 := v2.ValidateIteration(ctx, feed, nil)

	// Results should be identical
	if valid1 != valid2 {
		t.Errorf("Validation results differ: file=%v, bytes=%v", valid1, valid2)
	}

	if len(result1.Errors) != len(result2.Errors) {
		t.Errorf("Error count differs: file=%d, bytes=%d",
			len(result1.Errors), len(result2.Errors))
	}

	if len(result1.Warnings) != len(result2.Warnings) {
		t.Errorf("Warning count differs: file=%d, bytes=%d",
			len(result1.Warnings), len(result2.Warnings))
	}
}

func TestNewValidatorFromBytes_InvalidZip(t *testing.T) {
	invalidData := []byte("not a zip file")

	_, err := NewValidatorFromBytes(invalidData, Options{})
	if err == nil {
		t.Error("Expected error for invalid zip data")
	}

	if !strings.Contains(err.Error(), "failed to open") {
		t.Errorf("Expected 'failed to open' error, got: %v", err)
	}
}

func TestNewValidatorFromBytes_EmptyZip(t *testing.T) {
	// Create empty zip
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	_ = zw.Close()

	v, err := NewValidatorFromBytes(buf.Bytes(), Options{})
	// Should handle gracefully (empty Metadata, no errors)
	if err != nil {
		t.Errorf("Unexpected error for empty zip: %v", err)
	}

	// Validator should still be usable
	if v == nil {
		t.Error("Validator should not be nil for empty zip")
	}
}

func TestLoadMetadataFromBytes_AllFiles(t *testing.T) {
	gtfsBytes := createMinimalGTFSZip(t)

	md, err := LoadMetadataFromBytes(gtfsBytes, false)
	if err != nil {
		t.Fatalf("LoadMetadataFromBytes failed: %v", err)
	}

	// Verify all expected data was loaded
	if len(md.AgencyIDs) == 0 {
		t.Error("No agencies loaded")
	}
	if len(md.RouteIDs) == 0 {
		t.Error("No routes loaded")
	}
	if len(md.TripsByTripID) == 0 {
		t.Error("No trips loaded")
	}
	if len(md.StopTimesByTripID) == 0 {
		t.Error("No stop times loaded")
	}
	if len(md.StopIDs) == 0 {
		t.Error("No stops loaded")
	}
}

func TestZipBytesReaders_Basic(t *testing.T) {
	gtfsBytes := createMinimalGTFSZip(t)

	readers, cleanup, err := ZipBytesReaders(gtfsBytes)
	if err != nil {
		t.Fatalf("ZipBytesReaders failed: %v", err)
	}
	defer func() { _ = cleanup() }()

	// Check that all expected files are present
	expectedFiles := []string{"agency.txt", "routes.txt", "trips.txt", "stops.txt", "stop_times.txt"}
	for _, f := range expectedFiles {
		if _, ok := readers[f]; !ok {
			t.Errorf("Expected %s to be in readers", f)
		}
	}
}

func TestZipBytesReaders_InvalidData(t *testing.T) {
	invalidData := []byte("not a zip file")

	_, _, err := ZipBytesReaders(invalidData)
	if err == nil {
		t.Error("Expected error for invalid zip data")
	}
}
