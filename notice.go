package validator

type Severity string

const (
	Error   Severity = "ERROR"
	Warning Severity = "WARNING"
)

type RuleID string

type ValidationRule struct {
	ID               RuleID
	Severity         Severity
	Title            string
	Description      string
	OccurrenceSuffix string
}

type Occurrence struct {
	Prefix string
}

type ErrorList struct {
	Rule        ValidationRule
	Occurrences []Occurrence
}

type Result struct {
	Lists    []ErrorList
	Errors   []Occurrence
	Warnings []Occurrence
}

var rules = map[RuleID]ValidationRule{
	// Timestamp validators (ERRORS)
	"E001": {ID: "E001", Severity: Error, Title: "Not in POSIX time"},
	"E012": {ID: "E012", Severity: Error, Title: "Header timestamp less than entity timestamp"},
	"E017": {ID: "E017", Severity: Error, Title: "Content changed but same header timestamp"},
	"E018": {ID: "E018", Severity: Error, Title: "Header timestamp decreased"},
	"E048": {ID: "E048", Severity: Error, Title: "Header timestamp not populated (v2+)"},
	"E050": {ID: "E050", Severity: Error, Title: "Timestamp is in the future"},

	// Header validators (ERRORS)
	"E038": {ID: "E038", Severity: Error, Title: "Invalid header.gtfs_realtime_version"},
	"E039": {ID: "E039", Severity: Error, Title: "FULL_DATASET must not include is_deleted"},
	"E049": {ID: "E049", Severity: Error, Title: "Header incrementality not populated (v2+)"},

	// Vehicle validators (ERRORS)
	"E026": {ID: "E026", Severity: Error, Title: "Invalid vehicle position"},
	"E027": {ID: "E027", Severity: Error, Title: "Invalid vehicle bearing"},

	// StopTimeUpdate validators (ERRORS)
	"E002": {ID: "E002", Severity: Error, Title: "stop_sequence not increasing"},
	"E025": {ID: "E025", Severity: Error, Title: "Departure before arrival in stop_time_update"},
	"E036": {ID: "E036", Severity: Error, Title: "Duplicate stop_sequence in trip"},
	"E040": {ID: "E040", Severity: Error, Title: "stop_time_update missing stop_id and stop_sequence"},
	"E042": {ID: "E042", Severity: Error, Title: "NO_DATA stop_time_update should not have arrival/departure"},

	// TripDescriptor validators (ERRORS)
	"E020": {ID: "E020", Severity: Error, Title: "Invalid start_time format"},
	"E021": {ID: "E021", Severity: Error, Title: "Invalid start_date format"},
	"E032": {ID: "E032", Severity: Error, Title: "Alert missing informed_entity"},
	"E033": {ID: "E033", Severity: Error, Title: "Alert informed_entity missing specifier"},

	// Frequency validators (ERRORS)
	"E006": {ID: "E006", Severity: Error, Title: "Frequency-based trip missing required field"},
	"E019": {ID: "E019", Severity: Error, Title: "Frequency-based exact_times=1 start_time invalid"},

	// ============ WARNINGS ============

	// Timestamp warnings
	"W001": {ID: "W001", Severity: Warning, Title: "Timestamp not populated"},
	"W007": {ID: "W007", Severity: Warning, Title: "Refresh interval more than 35 seconds"},
	"W008": {ID: "W008", Severity: Warning, Title: "Header timestamp older than 65 seconds"},

	// Vehicle warnings
	"W002": {ID: "W002", Severity: Warning, Title: "Vehicle ID not populated"},
	"W004": {ID: "W004", Severity: Warning, Title: "Vehicle speed unrealistic"},
	"W011": {ID: "W011", Severity: Warning, Title: "Vehicle position outside service area"},
	"W014": {ID: "W014", Severity: Warning, Title: "Duplicate vehicle ID"},
	"W024": {ID: "W024", Severity: Warning, Title: "Vehicle position far from trip shape"},

	// Cross-feed warnings
	"W003": {ID: "W003", Severity: Warning, Title: "ID in one feed missing from the other"},
	"W033": {ID: "W033", Severity: Warning, Title: "VehiclePosition and TripUpdate pairing mismatch"},

	// StopTimeUpdate warnings
	"W010": {ID: "W010", Severity: Warning, Title: "Sequential stop_time_update times not increasing"},
	"W012": {ID: "W012", Severity: Warning, Title: "Duplicate stop_id in sequential stop_time_updates"},
	"W013": {ID: "W013", Severity: Warning, Title: "trip_update with no stop_time_updates for non-canceled trip"},
	"W017": {ID: "W017", Severity: Warning, Title: "stop_sequence not provided for trip with multiple visits to same stop"},
	"W029": {ID: "W029", Severity: Warning, Title: "stop_time_update missing arrival and departure"},
	"W030": {ID: "W030", Severity: Warning, Title: "stop_time_event with no time or delay"},
	"W031": {ID: "W031", Severity: Warning, Title: "stop_id does not match GTFS for stop_sequence"},
	"W032": {ID: "W032", Severity: Warning, Title: "Missing time in stop_time_update when GTFS has no times"},
	"W034": {ID: "W034", Severity: Warning, Title: "stop_sequence not found in GTFS"},

	// TripDescriptor warnings (reference mismatches)
	"W006": {ID: "W006", Severity: Warning, Title: "trip_update.trip missing trip_id"},
	"W015": {ID: "W015", Severity: Warning, Title: "trip_id not found in GTFS"},
	"W016": {ID: "W016", Severity: Warning, Title: "route_id not found in GTFS"},
	"W019": {ID: "W019", Severity: Warning, Title: "Frequency-based exact_times=0 trip not UNSCHEDULED"},
	"W021": {ID: "W021", Severity: Warning, Title: "trip_id exists in GTFS but marked as ADDED"},
	"W022": {ID: "W022", Severity: Warning, Title: "start_time does not match GTFS"},
	"W023": {ID: "W023", Severity: Warning, Title: "direction_id does not match GTFS"},
	"W025": {ID: "W025", Severity: Warning, Title: "Alert trip_id does not belong to route_id"},
	"W026": {ID: "W026", Severity: Warning, Title: "Alert informed_entity route_id does not match trip route_id"},
	"W027": {ID: "W027", Severity: Warning, Title: "agency_id not found in GTFS"},
	"W028": {ID: "W028", Severity: Warning, Title: "trip.trip_id does not belong to trip.route_id"},

	// Stop warnings (reference mismatches)
	"W018": {ID: "W018", Severity: Warning, Title: "stop_id not found in GTFS"},
	"W020": {ID: "W020", Severity: Warning, Title: "stop_id is not a stop (wrong location_type)"},

	// Frequency warnings
	"W005": {ID: "W005", Severity: Warning, Title: "Frequency-based trip missing vehicle_id"},
}

func GetRule(id RuleID) ValidationRule {
	if r, ok := rules[id]; ok {
		return r
	}
	return ValidationRule{ID: id}
}
