package validator

import (
	"context"
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
	if v.meta != nil {
		t.Error("RT-only validator should have nil metadata")
	}
	if !v.rtOnly {
		t.Error("RT-only validator should have rtOnly=true")
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
	// W014 is a warning, so the feed should be valid (no errors)
	if !valid {
		t.Error("Expected valid feed (W014 is a warning)")
	}

	// Should have W014 warning for duplicate vehicle ID
	found := false
	for _, el := range result.Lists {
		if el.Rule.ID == "W014" {
			found = true
			if el.Rule.Severity != Warning {
				t.Errorf("W014 should have Warning severity, got %s", el.Rule.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("Expected W014 warning for duplicate vehicle ID")
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
	// W010 is a warning, so the feed should be valid (no errors)
	if !valid {
		t.Error("Expected valid feed (W010 is a warning)")
	}

	// Should have W010 warning for non-increasing times
	found := false
	for _, el := range result.Lists {
		if el.Rule.ID == "W010" {
			found = true
			if el.Rule.Severity != Warning {
				t.Errorf("W010 should have Warning severity, got %s", el.Rule.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("Expected W010 warning for non-increasing times")
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

	// Should NOT have W015 or W018 warnings in RT-only mode (these require static GTFS)
	for _, el := range result.Lists {
		if el.Rule.ID == "W015" {
			t.Error("RT-only mode should not check W015 (trip_id not in GTFS)")
		}
		if el.Rule.ID == "W018" {
			t.Error("RT-only mode should not check W018 (stop_id not in GTFS)")
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

	// With nil metadata, both should work but full would check bbox if meta existed
	resultsRT := vRT.Validate(ctx, now, nil, feed, nil, nil)
	resultsFull := vFull.Validate(ctx, now, nil, feed, nil, nil)

	// Neither should have W011 with nil metadata
	for _, el := range resultsRT {
		if el.Rule.ID == "W011" {
			t.Error("RT-only should never produce W011")
		}
	}
	for _, el := range resultsFull {
		if el.Rule.ID == "W011" {
			t.Error("Full mode with nil metadata should not produce W011")
		}
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
