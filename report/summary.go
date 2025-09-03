package report

import (
	"sort"
	"strings"

	"your.org/gtfsrt/validator/notice"
)

type SummaryItem struct {
	RuleID   string `json:"ruleId"`
	Severity string `json:"severity"`
	Count    int    `json:"count"`
}

func Summarize(res notice.Result) []SummaryItem {
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
