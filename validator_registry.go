package validator

var defaultValidators = []FeedEntityValidator{
	HeaderValidator{},
	TimestampValidator{},
	CrossFeedValidator{},
	VehicleValidator{},
	StopTimeUpdateValidator{},
	TripDescriptorValidator{},
	StopValidator{},
	FreqZeroValidator{},
	FreqOneValidator{},
}

var rtOnlyValidators = []FeedEntityValidator{
	HeaderValidator{},
	TimestampValidator{},
	CrossFeedValidator{},
	VehicleValidator{RTOnly: true},
	StopTimeUpdateValidator{RTOnly: true},
}

// DefaultValidators returns all validators (requires static GTFS for full validation).
func DefaultValidators() []FeedEntityValidator {
	return append([]FeedEntityValidator(nil), defaultValidators...)
}

// RTOnlyValidators returns validators that only check GTFS-RT spec compliance
// without requiring static GTFS data. Use this for fast fail-fast validation
// before loading expensive static GTFS metadata.
func RTOnlyValidators() []FeedEntityValidator {
	return append([]FeedEntityValidator(nil), rtOnlyValidators...)
}
