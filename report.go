package validator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// RenderConsole renders a human-readable summary for console output.
func RenderConsole(res Result) string {
	if len(res.Lists) == 0 {
		return "No notices."
	}
	var b strings.Builder
	ids := make([]string, 0, len(res.Lists))
	byID := make(map[string]ErrorList)
	for _, el := range res.Lists {
		id := string(el.Rule.ID)
		if _, ok := byID[id]; !ok {
			ids = append(ids, id)
			byID[id] = el
		} else {
			merged := byID[id]
			merged.Occurrences = append(merged.Occurrences, el.Occurrences...)
			byID[id] = merged
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		el := byID[id]
		fmt.Fprintf(&b, "%s [%s] x%d - %s\n", id, el.Rule.Severity, len(el.Occurrences), el.Rule.Title)
	}
	return b.String()
}

// MarshalJSON returns a pretty-printed JSON representation of Result.
func MarshalJSON(res Result) ([]byte, error) {
	return json.MarshalIndent(res, "", "  ")
}

// SummaryItem is a compact representation used by the summary renderer.
type SummaryItem struct {
	RuleID   string `json:"ruleId"`
	Severity string `json:"severity"`
	Count    int    `json:"count"`
}

// Summarize aggregates counts per rule.
func Summarize(res Result) []SummaryItem {
	counts := map[string]int{}
	sev := map[string]string{}
	for _, el := range res.Lists {
		id := string(el.Rule.ID)
		counts[id] += len(el.Occurrences)
		sev[id] = string(el.Rule.Severity)
	}
	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]SummaryItem, 0, len(ids))
	for _, id := range ids {
		out = append(out, SummaryItem{RuleID: id, Severity: strings.ToUpper(sev[id]), Count: counts[id]})
	}
	return out
}

// Compatibility JSON models for Java parity.
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
	Prefix   string            `json:"prefix"`
	EntityID string            `json:"entityId,omitempty"`
	Value    string            `json:"value,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
}

type ErrorListHelperModel struct {
	ErrorMessage   errorMessage      `json:"errorMessage"`
	OccurrenceList []occurrenceModel `json:"occurrenceList"`
}

// ToCompat converts Result to the legacy/compat JSON structure.
func ToCompat(res Result) []ErrorListHelperModel {
	out := make([]ErrorListHelperModel, 0, len(res.Lists))
	for _, el := range res.Lists {
		m := ErrorListHelperModel{ErrorMessage: errorMessage{ValidationRule: validationRule{
			ErrorID:          string(el.Rule.ID),
			Severity:         string(el.Rule.Severity),
			Title:            el.Rule.Title,
			ErrorDescription: el.Rule.Description,
			OccurrenceSuffix: el.Rule.OccurrenceSuffix,
		}}}
		for _, oc := range el.Occurrences {
			m.OccurrenceList = append(m.OccurrenceList, occurrenceModel{
				Prefix:   oc.Prefix,
				EntityID: oc.EntityID,
				Value:    oc.Value,
				Details:  oc.Details,
			})
		}
		out = append(out, m)
	}
	return out
}
