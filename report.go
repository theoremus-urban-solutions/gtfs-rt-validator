package validator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// aggregateByRuleID merges ErrorLists with the same Rule.ID.
func aggregateByRuleID(lists []ErrorList) []ErrorList {
	if len(lists) == 0 {
		return nil
	}
	ids := make([]string, 0, len(lists))
	byID := make(map[string]ErrorList)
	for _, el := range lists {
		id := string(el.Rule.ID)
		if existing, ok := byID[id]; !ok {
			ids = append(ids, id)
			byID[id] = el
		} else {
			existing.Occurrences = append(existing.Occurrences, el.Occurrences...)
			byID[id] = existing
		}
	}
	sort.Strings(ids)
	out := make([]ErrorList, 0, len(ids))
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out
}

// RenderConsole renders a human-readable summary for console output.
func RenderConsole(res Result) string {
	if len(res.Lists) == 0 {
		return "No notices."
	}
	var b strings.Builder
	for _, el := range aggregateByRuleID(res.Lists) {
		fmt.Fprintf(&b, "%s [%s] x%d - %s\n", string(el.Rule.ID), el.Rule.Severity, len(el.Occurrences), el.Rule.Title)
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
	aggregated := aggregateByRuleID(res.Lists)
	out := make([]ErrorListHelperModel, 0, len(aggregated))
	for _, el := range aggregated {
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
