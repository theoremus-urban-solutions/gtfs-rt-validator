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
	Prefix   string            `json:"prefix"`
	EntityID string            `json:"entity_id,omitempty"`
	Value    string            `json:"value,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
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
	"E028": {ID: "E028", Severity: Error, Title: "Vehicle position outside service area"},
	"E052": {ID: "E052", Severity: Error, Title: "Duplicate vehicle ID"},

	// StopTimeUpdate validators (ERRORS)
	"E002": {ID: "E002", Severity: Error, Title: "stop_sequence not increasing"},
	"E009": {ID: "E009", Severity: Error, Title: "stop_sequence not provided for trip with multiple visits to same stop"},
	"E022": {ID: "E022", Severity: Error, Title: "Sequential stop_time_update times not increasing"},
	"E025": {ID: "E025", Severity: Error, Title: "Departure before arrival in stop_time_update"},
	"E036": {ID: "E036", Severity: Error, Title: "Duplicate stop_sequence in trip"},
	"E037": {ID: "E037", Severity: Error, Title: "Duplicate stop_id in sequential stop_time_updates"},
	"E040": {ID: "E040", Severity: Error, Title: "stop_time_update missing stop_id and stop_sequence"},
	"E041": {ID: "E041", Severity: Error, Title: "trip_update with no stop_time_updates for non-canceled trip"},
	"E042": {ID: "E042", Severity: Error, Title: "NO_DATA stop_time_update should not have arrival/departure"},
	"E043": {ID: "E043", Severity: Error, Title: "stop_time_update missing arrival and departure"},
	"E044": {ID: "E044", Severity: Error, Title: "stop_time_event with no time or delay"},
	"E045": {ID: "E045", Severity: Error, Title: "stop_id does not match GTFS for stop_sequence"},
	"E046": {ID: "E046", Severity: Error, Title: "Missing time in stop_time_update when GTFS has no times"},
	"E051": {ID: "E051", Severity: Error, Title: "stop_sequence not found in GTFS"},

	// TripDescriptor validators (ERRORS)
	"E003": {ID: "E003", Severity: Error, Title: "trip_id not found in GTFS"},
	"E004": {ID: "E004", Severity: Error, Title: "route_id not found in GTFS"},
	"E016": {ID: "E016", Severity: Error, Title: "trip_id exists in GTFS but marked as ADDED"},
	"E020": {ID: "E020", Severity: Error, Title: "Invalid start_time format"},
	"E021": {ID: "E021", Severity: Error, Title: "Invalid start_date format"},
	"E023": {ID: "E023", Severity: Error, Title: "start_time does not match GTFS"},
	"E024": {ID: "E024", Severity: Error, Title: "direction_id does not match GTFS"},
	"E030": {ID: "E030", Severity: Error, Title: "Alert trip_id does not belong to route_id"},
	"E031": {ID: "E031", Severity: Error, Title: "Alert informed_entity route_id does not match trip route_id"},
	"E032": {ID: "E032", Severity: Error, Title: "Alert missing informed_entity"},
	"E033": {ID: "E033", Severity: Error, Title: "Alert informed_entity missing specifier"},
	"E034": {ID: "E034", Severity: Error, Title: "agency_id not found in GTFS"},
	"E035": {ID: "E035", Severity: Error, Title: "trip.trip_id does not belong to trip.route_id"},

	// Stop validators (ERRORS)
	"E011": {ID: "E011", Severity: Error, Title: "stop_id not found in GTFS"},
	"E015": {ID: "E015", Severity: Error, Title: "stop_id is not a stop (wrong location_type)"},

	// Frequency validators (ERRORS)
	"E006": {ID: "E006", Severity: Error, Title: "Frequency-based trip missing required field"},
	"E013": {ID: "E013", Severity: Error, Title: "Frequency-based exact_times=0 trip not UNSCHEDULED"},
	"E019": {ID: "E019", Severity: Error, Title: "Frequency-based exact_times=1 start_time invalid"},

	// Cross-feed validators (ERRORS)
	"E047": {ID: "E047", Severity: Error, Title: "VehiclePosition and TripUpdate pairing mismatch"},
	"E029": {ID: "E029", Severity: Error, Title: "Vehicle position far from trip shape"},

	// ============ WARNINGS ============

	// Timestamp warnings
	"W001": {ID: "W001", Severity: Warning, Title: "Timestamp not populated"},
	"W007": {ID: "W007", Severity: Warning, Title: "Refresh interval more than 35 seconds"},
	"W008": {ID: "W008", Severity: Warning, Title: "Header timestamp older than 65 seconds"},

	// Vehicle warnings
	"W002": {ID: "W002", Severity: Warning, Title: "Vehicle ID not populated"},
	"W004": {ID: "W004", Severity: Warning, Title: "Vehicle speed unrealistic"},

	// Cross-feed warnings
	"W003": {ID: "W003", Severity: Warning, Title: "ID in one feed missing from the other"},

	// TripDescriptor warnings
	"W006": {ID: "W006", Severity: Warning, Title: "trip_update.trip missing trip_id"},
	"W009": {ID: "W009", Severity: Warning, Title: "schedule_relationship not populated"},

	// Frequency warnings
	"W005": {ID: "W005", Severity: Warning, Title: "Frequency-based trip missing vehicle_id"},
}

func GetRule(id RuleID) ValidationRule {
	if r, ok := rules[id]; ok {
		return r
	}
	return ValidationRule{ID: id}
}

// errorListDetailed creates an ErrorList with a detailed Occurrence
func errorListDetailed(id, prefix, entityID, value string, details map[string]string) ErrorList {
	return ErrorList{
		Rule: GetRule(RuleID(id)),
		Occurrences: []Occurrence{{
			Prefix:   prefix,
			EntityID: entityID,
			Value:    value,
			Details:  details,
		}},
	}
}
