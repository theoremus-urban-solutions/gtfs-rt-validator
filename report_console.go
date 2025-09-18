package validator

import (
	"fmt"
	"sort"
	"strings"
)

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
