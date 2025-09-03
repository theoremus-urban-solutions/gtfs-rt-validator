package validator

import "your.org/gtfsrt/validator/notice"

// DefaultValidators returns the ordered list (subset implemented so far)
func DefaultValidators() []FeedEntityValidator {
	return []FeedEntityValidator{
		CrossFeedValidator{},
		VehicleValidator{},
		TimestampValidator{},
		StopTimeUpdateValidator{},
		TripDescriptorValidator{},
		StopValidator{},
		FreqZeroValidator{},
		FreqOneValidator{},
		HeaderValidator{},
	}
}

// Rule metadata placeholder for spot-checking
var Rules = map[notice.RuleID]notice.ValidationRule{
	"E038": {ID: "E038", Severity: notice.Error, Title: "Invalid header.gtfs_realtime_version"},
}
