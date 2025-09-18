package validator

type errorMessage struct {
	ValidationRule validationRule `json:"validationRule"`
}

type validationRule struct {
	ErrorID          string `json:"errorId"`
	Severity         string `json:"severity"`
	Title            string `json:"title"`
	ErrorDescription string `json:"errorDescription"`
	OccurrenceSuffix string `json:"occurrenceSuffix"`
}

type occurrenceModel struct {
	Prefix string `json:"prefix"`
}

type ErrorListHelperModel struct {
	ErrorMessage   errorMessage      `json:"errorMessage"`
	OccurrenceList []occurrenceModel `json:"occurrenceList"`
}

func ToCompat(res Result) []ErrorListHelperModel {
	out := make([]ErrorListHelperModel, 0, len(res.Lists))
	for _, el := range res.Lists {
		m := ErrorListHelperModel{
			ErrorMessage: errorMessage{ValidationRule: validationRule{
				ErrorID:          string(el.Rule.ID),
				Severity:         string(el.Rule.Severity),
				Title:            el.Rule.Title,
				ErrorDescription: el.Rule.Description,
				OccurrenceSuffix: el.Rule.OccurrenceSuffix,
			}},
		}
		for _, oc := range el.Occurrences {
			m.OccurrenceList = append(m.OccurrenceList, occurrenceModel{Prefix: oc.Prefix})
		}
		out = append(out, m)
	}
	return out
}
