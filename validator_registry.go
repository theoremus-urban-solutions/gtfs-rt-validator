package validator

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
