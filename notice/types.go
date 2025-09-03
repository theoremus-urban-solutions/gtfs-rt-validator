package notice

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
