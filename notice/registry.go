package notice

var rules = map[RuleID]ValidationRule{
	"E001": {ID: "E001", Severity: Error, Title: "Not in POSIX time"},
	"E012": {ID: "E012", Severity: Error, Title: "Header timestamp less than entity timestamp"},
	"E017": {ID: "E017", Severity: Error, Title: "Content changed but same header timestamp"},
	"E018": {ID: "E018", Severity: Error, Title: "Header timestamp decreased"},
	"E022": {ID: "E022", Severity: Error, Title: "Sequential stop_time_update times not increasing"},
	"E025": {ID: "E025", Severity: Error, Title: "Departure before arrival in stop_time_update"},
	"E038": {ID: "E038", Severity: Error, Title: "Invalid header.gtfs_realtime_version"},
	"E039": {ID: "E039", Severity: Error, Title: "FULL_DATASET must not include is_deleted"},
	"E047": {ID: "E047", Severity: Error, Title: "VehiclePosition and TripUpdate pairing mismatch"},
	"E048": {ID: "E048", Severity: Error, Title: "Header timestamp not populated (v2+)"},
	"E049": {ID: "E049", Severity: Error, Title: "Header incrementality not populated (v2+)"},
	"E050": {ID: "E050", Severity: Error, Title: "Timestamp is in the future"},
	"W001": {ID: "W001", Severity: Warning, Title: "Timestamp not populated"},
	"W003": {ID: "W003", Severity: Warning, Title: "ID in one feed missing from the other"},
	"W007": {ID: "W007", Severity: Warning, Title: "Refresh interval more than 35 seconds"},
	"W008": {ID: "W008", Severity: Warning, Title: "Header timestamp older than 65 seconds"},
}

func GetRule(id RuleID) ValidationRule {
	if r, ok := rules[id]; ok {
		return r
	}
	return ValidationRule{ID: id}
}
