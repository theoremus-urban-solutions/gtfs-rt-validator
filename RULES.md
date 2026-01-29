# Implemented rules

Rules are based on the [MobilityData canonical GTFS-RT validator](https://github.com/MobilityData/gtfs-realtime-validator/blob/master/RULES.md) rule IDs and severities.

### Table of Errors

Errors indicate spec violations, format errors, or logically impossible data.

| Error ID      | Error Title         |
|---------------|---------------------------|
| [E001](#E001) | Not in POSIX time
| [E002](#E002) | `stop_time_updates` not strictly sorted
| [E003](#E003) | GTFS-rt `trip_id` does not exist in GTFS data
| [E004](#E004) | GTFS-rt `route_id` does not exist in GTFS data
| [E006](#E006) | Missing required trip field for frequency-based `exact_times` = 0
| [E009](#E009) | GTFS-rt `stop_sequence` isn't provided for trip that visits same `stop_id` more than once
| [E011](#E011) | GTFS-rt `stop_id` does not exist in GTFS data
| [E012](#E012) | Header `timestamp` should be greater than or equal to all other `timestamps`
| [E013](#E013) | Frequency type 0 trip `schedule_relationship` should be `UNSCHEDULED` or empty
| [E015](#E015) | All `stop_ids` referenced in GTFS-rt feeds must have `location_type` = 0
| [E016](#E016) | `trip_ids` with `schedule_relationship` `ADDED` must not be in GTFS data
| [E017](#E017) | GTFS-rt content changed but has the same header `timestamp`
| [E018](#E018) | GTFS-rt header `timestamp` decreased between two sequential iterations
| [E019](#E019) | GTFS-rt frequency type 1 trip `start_time` must be a multiple of GTFS `headway_secs` later than GTFS `start_time`
| [E020](#E020) | Invalid `start_time` format
| [E021](#E021) | Invalid `start_date` format
| [E022](#E022) | Sequential `stop_time_update` times are not increasing
| [E023](#E023) | trip `start_time` does not match first GTFS `arrival_time`
| [E024](#E024) | trip `direction_id` does not match GTFS data
| [E025](#E025) | `stop_time_update` departure time is before arrival time
| [E026](#E026) | Invalid vehicle `position`
| [E027](#E027) | Invalid vehicle `bearing`
| [E028](#E028) | Vehicle `position` outside agency coverage area
| [E029](#E029) | Vehicle `position` far from trip shape
| [E030](#E030) | GTFS-rt alert `trip_id` does not belong to GTFS-rt alert `route_id`
| [E031](#E031) | Alert `informed_entity.route_id` does not match `informed_entity.trip.route_id`
| [E032](#E032) | Alert does not have an `informed_entity`
| [E033](#E033) | Alert `informed_entity` does not have any specifiers
| [E034](#E034) | GTFS-rt `agency_id` does not exist in GTFS data
| [E035](#E035) | GTFS-rt `trip.trip_id` does not belong to GTFS-rt `trip.route_id`
| [E036](#E036) | Sequential `stop_time_updates` have the same `stop_sequence`
| [E037](#E037) | Sequential `stop_time_updates` have the same `stop_id`
| [E038](#E038) | Invalid `header.gtfs_realtime_version`
| [E039](#E039) | `FULL_DATASET` feeds should not include `entity.is_deleted`
| [E040](#E040) | `stop_time_update` doesn't contain `stop_id` or `stop_sequence`
| [E041](#E041) | `trip` doesn't have any `stop_time_updates`
| [E042](#E042) | `arrival` or `departure` provided for `NO_DATA` `stop_time_update`
| [E043](#E043) | `stop_time_update` doesn't have `arrival` or `departure`
| [E044](#E044) | `stop_time_update` `arrival/departure` doesn't have `delay` or `time`
| [E045](#E045) | GTFS-rt `stop_time_update` `stop_sequence` and `stop_id` do not match GTFS
| [E046](#E046) | GTFS-rt `stop_time_update` without `time` doesn't have arrival/departure time in GTFS
| [E047](#E047) | `VehiclePosition` and `TripUpdate` ID pairing mismatch
| [E048](#E048) | `header` `timestamp` not populated (GTFS-rt v2.0 and higher)
| [E049](#E049) | `header` `incrementality` not populated (GTFS-rt v2.0 and higher)
| [E050](#E050) | `timestamp` is in the future
| [E051](#E051) | GTFS-rt `stop_sequence` not found in GTFS data
| [E052](#E052) | `vehicle.id` is not unique

### Table of Warnings

Warnings indicate data quality issues or missing optional data.

| Warning ID    | Warning Title             |
|---------------|---------------------------|
| [W001](#W001) | `timestamps` not populated
| [W002](#W002) | `vehicle_id` not populated
| [W003](#W003) | ID in one feed missing from the other
| [W004](#W004) | vehicle `speed` is unrealistic
| [W005](#W005) | Missing `vehicle_id` in `trip_update` for frequency-based `exact_times` = 0
| [W006](#W006) | `trip_update` missing `trip_id`
| [W007](#W007) | Refresh interval is more than 35 seconds
| [W008](#W008) | Header `timestamp` is older than 65 seconds
| [W009](#W009) | `schedule_relationship` not populated

# Errors

<a name="E001"/>

### E001 - Not in POSIX time

All times and timestamps must be in [POSIX time](https://en.wikipedia.org/wiki/Unix_time) (i.e., number of **seconds** since January 1st 1970 00:00:00 UTC).

*Common mistakes* - Accidentally using Java's `System.currentTimeMillis()`, which is the number of **milliseconds** since January 1st 1970 00:00:00 UTC.

*Possible solution* - Use `TimeUnit.MILLISECONDS.toSeconds(System.currentTimeMillis())` to convert from milliseconds to seconds.

#### References:
* [`header.timestamp`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-feedheader)
* [`trip_update.timestamp`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripupdate)
* [`vehicle_postion.timestamp`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-vehicleposition)
* [`stop_time_update.arrival/departure.time`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeevent)
* [`alert.active_period.start` and `alert.active_period.end`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-timerange)

<a name="E002"/>

### E002 - `stop_time_updates` not strictly sorted

`stop_time_updates` for a given `trip_id` must be strictly ordered by increasing `stop_sequence` - this also means that no `stop_sequence` should be repeated.

#### References:
* [Stop Time Updates description](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/trip-updates.md#stop-time-updates)
* [`stop_time_update` reference](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)
* [GTFS `stop_times.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#stop_timestxt)

<a name="E003"/>

### E003 - GTFS-rt `trip_id` does not exist in GTFS data

All `trip_ids` provided in the GTFS-rt feed must exist in the GTFS data, unless their `schedule_relationship` is set to `ADDED`.

#### References:
* [`trip.trip_id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)
* [`trip.schedule_relationship`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#enum-schedulerelationship-1)
* [GTFS `trips.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#tripstxt)

<a name="E004"/>

### E004 - GTFS-rt `route_id` does not exist in GTFS data

All `route_ids` provided in the GTFS-rt feed must exist in the GTFS data.

#### References:
* [`trip.route_id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)
* [GTFS `routes.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#routestxt)

<a name="E006"/>

### E006 - Missing required trip field for frequency-based exact_times = 0

Frequency-based `exact_times` = 0 `trip_updates` must contain `trip_id`, `start_time`, and `start_date`.

#### References:
* [`trip_update.trip`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/trip-updates.md#systems-with-repeated-trip_ids)

<a name="E009"/>

### E009 - GTFS-rt `stop_sequence` isn't provided for `trip` that visits same `stop_id` more than once

If a GTFS `trip` contains multiple references to the same `stop_id` (i.e., the vehicle visits the same `stop_id` more than once in the same trip), then GTFS-rt `stop_time_updates` for this trip must include `stop_sequence`.

#### References:
* [`trip_update`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripupdate)

<a name="E011"/>

### E011 - GTFS-rt `stop_id` does not exist in GTFS data

All `stop_ids` referenced in GTFS-rt feeds must exist in the GTFS data in `stops.txt`.

#### References:
* [`stop_time_update.stop_id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)
* [`position.stop_id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-vehicleposition)
* [GTFS `stops.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#stopstxt)

<a name="E012"/>

### E012 - Header `timestamp` should be greater than or equal to all other timestamps

No timestamps for individual entities (TripUpdate, VehiclePosition, Alerts) in the feeds should be greater than the header timestamp.

#### References:
* [`header`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-feedheader)

<a name="E013"/>

### E013 - Frequency type 0 trip `schedule_relationship` should be `UNSCHEDULED` or empty

For frequency-based exact_times=0 trips, schedule_relationship should be `UNSCHEDULED` or empty.

#### References:
* [Trip Updates -> Trip Descriptor description](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/trip-updates.md#trip-descriptor)
* [`trip_update.trip.schedule_relationship`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#enum-schedulerelationship-1)

<a name="E015"/>

### E015 - All `stop_ids` referenced in GTFS-rt feeds must have `location_type` = 0

All `stop_ids` referenced in GTFS-rt TripUpdates and VehiclePositions feeds must have the `location_type` = 0 in GTFS `stops.txt`.

#### References:
* [GTFS `stop_times.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#stop_timestxt)
* [GTFS `stops.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#stopstxt)

<a name="E016"/>

### E016 - `trip_ids` with `schedule_relationship` `ADDED` must not be in GTFS data

Trips that have a `schedule_relationship` of `ADDED` must **NOT** be included in the GTFS data.

#### References:
* [`trip.schedule_relationship`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#enum-schedulerelationship-1)
* [`trip.trip_id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)
* [GTFS `trips.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#tripstxt)

<a name="E017"/>

### E017 - GTFS-rt content changed but has the same header `timestamp`

The GTFS-rt header `timestamp` value should always change if the feed contents change.

#### References:
 * [`header.timestamp`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-feedheader)

<a name="E018"/>

### E018 - GTFS-rt header `timestamp` decreased between two sequential iterations

The GTFS-rt header `timestamp` should be monotonically increasing.

#### References:
* [`header.timestamp`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-feedheader)

<a name="E019"/>

### E019 - GTFS-rt frequency type 1 trip `start_time` must be a multiple of GTFS data `start_time`

For frequency-based trips defined in `frequencies.txt` with `exact_times` = 1, the GTFS-rt trip `start_time` must be some multiple (including zero) of `headway_secs` later than the `start_time` in `frequencies.txt`.

#### References:
* [`trip.start_time`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)

<a name="E020"/>

### E020 - Invalid `start_time` format

`start_time` must be in the format `HH:MM:SS` or `H:MM:SS`.

#### References:
* [`trip.start_time`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)

<a name="E021"/>

### E021 - Invalid `start_date` format

`start_date` must be in the `YYYYMMDD` format.

#### References:
* [`trip.start_date`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)

<a name="E022"/>

### E022 - Sequential `stop_time_update` times are not increasing

Sequential `stop_time_update` `arrival` and/or `departure` times should always increase - earlier `stop_time_updates` should have earlier times than later `stop_time_updates`.

#### References:
* [Stop Time Updates description](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/trip-updates.md#stop-time-updates)
* [`stop_time_update` reference](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)

<a name="E023"/>

### E023 - trip `start_time` does not match first GTFS `arrival_time`

For normal scheduled trips (not defined in `frequencies.txt`), the GTFS-realtime trip `start_time` must match the first GTFS `arrival_time` in `stop_times.txt`.

#### References:
* [`trip.start_time`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)
* [GTFS `stop_times.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#stop_timestxt)

<a name="E024"/>

### E024 - trip `direction_id` does not match GTFS data

GTFS-rt trip `direction_id` must match the `direction_id` in GTFS `trips.txt`.

#### References:
* [`trip.direction_id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)
* [GTFS `trips.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#tripstxt)

<a name="E025"/>

### E025 - `stop_time_update` departure time is before arrival time

Within the same `stop_time_update`, the departure time should never come before the arrival time.

#### References:
* [Stop Time Updates description](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/trip-updates.md#stop-time-updates)
* [`stop_time_update` reference](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)

<a name="E026"/>

### E026 - Invalid vehicle `position`

Vehicle position must be valid WGS84 coordinates - latitude between -90 and 90, longitude between -180 and 180.

#### References:
* [`vehicle.position`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-position)

<a name="E027"/>

### E027 - Invalid vehicle `bearing`

Vehicle bearing must be between 0 and 360 degrees.

#### References:
* [`vehicle.position.bearing`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-position)

<a name="E028"/>

### E028 - Vehicle `position` outside agency coverage area

The vehicle `position` should be inside the agency coverage area, defined by a buffer surrounding the GTFS `shapes.txt` data or `stops.txt` locations.

#### References:
* [`vehicle.position`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-position)

<a name="E029"/>

### E029 - Vehicle `position` far from trip shape

The vehicle `position` should be within a buffer surrounding the GTFS `shapes.txt` data for the current trip.

#### References:
* [GTFS `shapes.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#shapestxt)
* [`vehicle.position`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-position)

<a name="E030"/>

### E030 - GTFS-rt alert `trip_id` does not belong to GTFS-rt alert `route_id`

The GTFS-rt `alert.informed_entity.trip.trip_id` should belong to the specified `alert.informed_entity.route_id` in GTFS `trips.txt`.

#### References:
* [`alert.informed_entity`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-entityselector)
* [GTFS `trips.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#tripstxt)

<a name="E031"/>

### E031 - Alert `informed_entity.route_id` does not match `informed_entity.trip.route_id`

The `alert.informed_entity.trip.route_id` should be the same as the specified `alert.informed_entity.route_id`.

#### References:
* [`alert.informed_entity`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-entityselector)

<a name="E032"/>

### E032 - Alert does not have an `informed_entity`

All alerts must have at least one `informed_entity`.

#### References:
* [`alert.informed_entity`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-entityselector)

<a name="E033"/>

### E033 - Alert `informed_entity` does not have any specifiers

Alert `informed_entity` should have at least one specified value (`route_id`, `trip_id`, `stop_id`, etc).

#### References:
* [`alert.informed_entity`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-entityselector)

<a name="E034"/>

### E034 - GTFS-rt `agency_id` does not exist in GTFS data

All `agency_ids` provided in the GTFS-rt `alert.informed_entity.agency_id` should exist in GTFS `agency.txt`.

#### References:
* [`alert.informed_entity`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-entityselector)

<a name="E035"/>

### E035 - GTFS-rt `trip.trip_id` does not belong to GTFS-rt `trip.route_id`

The GTFS-rt `trip.trip_id` should belong to the specified `trip.route_id` in GTFS `trips.txt`.

#### References:
* [`trip.trip_id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)
* [`trip.route_id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)

<a name="E036"/>

### E036 - Sequential `stop_time_updates` have the same `stop_sequence`

Sequential GTFS-rt trip `stop_time_updates` should never have the same `stop_sequence`.

#### References:
* [`trip.stop_time_update`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/trip-updates.md#stop-time-updates)
* [GTFS `stop_times.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#stop_timestxt)

<a name="E037"/>

### E037 - Sequential `stop_time_updates` have the same `stop_id`

Sequential GTFS-rt trip `stop_time_updates` shouldn't have the same `stop_id`.

#### References:
* [`trip.stop_time_update`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/trip-updates.md#stop-time-updates)

<a name="E038"/>

### E038 - Invalid `header.gtfs_realtime_version`

`header.gtfs_realtime_version` is required and must be a valid value (`1.0` or `2.0`).

#### References:
* [`header.gtfs_realtime_version`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-feedheader)

<a name="E039"/>

### E039 - `FULL_DATASET` feeds should not include `entity.is_deleted`

The `entity.is_deleted` field should only be included in feeds with `header.incrementality` of `DIFFERENTIAL`.

#### References:
* [`header.incrementality`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-feedheader)
* [`entity.is_deleted`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-feedentity)

<a name="E040"/>

### E040 - `stop_time_update` doesn't contain `stop_id` or `stop_sequence`

All `stop_time_updates` must contain `stop_id` or `stop_sequence`.

#### References:
* [`trip.stop_time_update`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)

<a name="E041"/>

### E041 - `trip` doesn't have any `stop_time_updates`

Unless a `trip's` `schedule_relationship` is `CANCELED`, a `trip` should have at least one `stop_time_update`.

#### References:
* [`trip_update`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripupdate)
* [`trip_update.stop_time_update`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)

<a name="E042"/>

### E042 - `arrival` or `departure` provided for `NO_DATA` `stop_time_update`

If a `stop_time_update` has a `schedule_relationship` of `NO_DATA`, then neither `arrival` nor `departure` should be provided.

#### References:
* [`stop_time_update`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)

<a name="E043"/>

### E043 - `stop_time_update` doesn't have `arrival` or `departure`

If a `stop_time_update` doesn't have a `schedule_relationship` of `SKIPPED` or `NO_DATA`, then either `arrival` or `departure` must be provided.

#### References:
* [`stop_time_update`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)

<a name="E044"/>

### E044 - `stop_time_update` `arrival/departure` doesn't have `delay` or `time`

`stop_time_update.arrival` and `stop_time_update.departure` must have either `delay` or `time`.

#### References:
* [Stop Time Updates description](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/trip-updates.md#stop-time-updates)
* [`stop_time_update` reference](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)

<a name="E045"/>

### E045 - GTFS-rt `stop_time_update` `stop_sequence` and `stop_id` do not match GTFS

If GTFS-rt stop_time_update contains both stop_sequence and stop_id, the values must match the GTFS data in stop_times.txt.

#### References:
* [`stop_time_update`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)

<a name="E046"/>

### E046 - GTFS-rt `stop_time_update` without `time` doesn't have arrival/departure_time in GTFS

If only `delay` is provided in a `stop_time_update` (and not `time`), then the GTFS `stop_times.txt` must contain arrival/departure times for the corresponding stops.

#### References:
* [`stop_time_update`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)
* [GTFS `stop_times.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#stop_timestxt)

<a name="E047"/>

### E047 - `VehiclePosition` and `TripUpdate` ID pairing mismatch

If separate `VehiclePositions` and `TripUpdates` feeds are provided, `VehicleDescriptor` or `TripDescriptor` ID value pairing should match between the two feeds.

Note that this is different from W003, which simply checks if an ID provided in one feed is missing from the other.

#### References:
* [`vehicle.id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-vehicledescriptor)
* [`trip.trip_id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)

<a name="E048"/>

### E048 - `header` `timestamp` not populated (GTFS-rt v2.0 and higher)

`timestamp` must be populated in `FeedHeader` for `gtfs_realtime_version` v2.0 and higher.

#### References:
* [`header.timestamp`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-feedheader)

<a name="E049"/>

### E049 - `header` `incrementality` not populated (GTFS-rt v2.0 and higher)

`incrementality` must be populated in `FeedHeader` for `gtfs_realtime_version` v2.0 and higher.

#### References:
* [`header.incrementality`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-feedheader)

<a name="E050"/>

### E050 - `timestamp` is in the future

All timestamps should be less than the current time.

#### References:
* [`header.timestamp`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-feedheader)

<a name="E051"/>

### E051 - GTFS-rt `stop_sequence` not found in GTFS data

All `stop_time_update` `stop_sequences` in GTFS-realtime data must appear in GTFS `stop_times.txt` for that trip.

#### References:
* [`stop_time_update`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-stoptimeupdate)
* [GTFS `stop_times.txt`](https://github.com/google/transit/blob/master/gtfs/spec/en/reference.md#stop_timestxt)

<a name="E052"/>

### E052 - `vehicle.id` is not unique

Each vehicle should have a unique ID.

#### References:
* [`vehicle.id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-vehicledescriptor)

# Warnings

<a name="W001"/>

### W001 - `timestamp` not populated

`timestamps` should be populated for `FeedHeader`, `TripUpdates`, `VehiclePositions`, and `Alerts`.

<a name="W002"/>

### W002 - `vehicle_id` not populated

`vehicle_id` should be populated for TripUpdates and VehiclePositions.

<a name="W003"/>

### W003 - ID in one feed missing from the other

If separate `VehiclePositions` and `TripUpdates` feeds are provided, a `trip_id` or `vehicle_id` provided in one feed should also be provided in the other.

Note that this is different from E047, which checks for a mismatch of IDs between the feeds.

#### References:
* [`vehicle.id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-vehicledescriptor)
* [`trip.trip_id`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-tripdescriptor)

<a name="W004"/>

### W004 - vehicle `speed` is unrealistic

`vehicle.position.speed` has an unrealistic speed that may be incorrect. Speeds are flagged if greater than 26 m/s (~60 mph).

#### References:
* [`vehicle.position.speed`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#message-position)

<a name="W005"/>

### W005 - Missing `vehicle_id` in `trip_update` for frequency-based exact_times = 0

Frequency-based exact_times = 0 trip_updates should contain `vehicle_id` to disambiguate predictions.

#### References:
* [`trip_update.trip`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/trip-updates.md#alternative-trip-matching)

<a name="W006"/>

### W006 - `trip` missing `trip_id`

`trips` should include a `trip_id`.

#### References:
* [`trip_update.trip`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/trip-updates.md#alternative-trip-matching)

<a name="W007"/>

### W007 - Refresh interval is more than 35 seconds

GTFS-realtime feeds should be refreshed at least every 35 seconds.

<a name="W008"/>

### W008 - Header `timestamp` is older than 65 seconds

The data in a GTFS-realtime feed should always be less than one minute old.

<a name="W009"/>

### W009 - `schedule_relationship` not populated

`trip.schedule_relationship` and `stop_time_update.schedule_relationship` should be populated.

#### References:
* [`trip.schedule_relationship`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#enum-schedulerelationship-1)
* [`stop_time_update.schedule_relationship`](https://github.com/google/transit/blob/master/gtfs-realtime/spec/en/reference.md#enum-schedulerelationship)
