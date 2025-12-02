package validator

// DefaultValidators returns all validators (requires static GTFS for full validation).
func DefaultValidators() []FeedEntityValidator {
	return []FeedEntityValidator{
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
}

// RTOnlyValidators returns validators that only check GTFS-RT spec compliance
// without requiring static GTFS data. Use this for fast fail-fast validation
// before loading expensive static GTFS metadata.
func RTOnlyValidators() []FeedEntityValidator {
	return []FeedEntityValidator{
		HeaderValidator{},
		TimestampValidator{},
		CrossFeedValidator{},
		VehicleValidator{RTOnly: true},
		StopTimeUpdateValidator{RTOnly: true},
	}
}
