package validator

import (
	"context"
	"time"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

const defaultMaxSamplesPerRule = 5

// PrevHeader is the previous build of the same feed, for E017, E018 and W007.
type PrevHeader struct {
	Timestamp   uint64
	EntityCount int
}

// HeaderOf returns the PrevHeader to pass when validating the next build of this feed.
func HeaderOf(feed *rt.FeedMessage) PrevHeader {
	return PrevHeader{Timestamp: feed.GetHeader().GetTimestamp(), EntityCount: len(feed.GetEntity())}
}

// Check holds one feed's inputs and collects what the rules find.
type Check struct {
	Ctx context.Context
	// NowMs is the check time in Unix milliseconds.
	NowMs int64
	// Static is nil when only GTFS-RT rules run.
	Static   Static
	Feed     *rt.FeedMessage
	Prev     *PrevHeader
	Combined *rt.FeedMessage

	max    int
	lists  []ErrorList
	byRule map[RuleID]int
	counts map[RuleID]int
}

// ValidateFeed checks one feed against static and the same feed's previous header; it keeps no state and is safe for concurrent use.
func ValidateFeed(ctx context.Context, static Static, feed *rt.FeedMessage, prev *PrevHeader, opts Options) Result {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	max := opts.MaxSamplesPerRule
	if max == 0 {
		max = defaultMaxSamplesPerRule
	}
	c := &Check{Ctx: ctx, NowMs: now.UnixMilli(), Static: static, Feed: feed, Prev: prev, Combined: opts.Combined, max: max}
	rules := defaultValidators
	if static == nil {
		rules = rtOnlyValidators
	}
	for _, r := range rules {
		r.Validate(c)
	}
	return c.result()
}

func (c *Check) add(id RuleID, prefix, entityID, value string, details map[string]string) {
	if c.byRule == nil {
		c.byRule = make(map[RuleID]int)
		c.counts = make(map[RuleID]int)
	}
	c.counts[id]++
	i, ok := c.byRule[id]
	if !ok {
		i = len(c.lists)
		c.byRule[id] = i
		c.lists = append(c.lists, ErrorList{Rule: GetRule(id)})
	}
	if c.max < 0 || len(c.lists[i].Occurrences) < c.max {
		c.lists[i].Occurrences = append(c.lists[i].Occurrences, Occurrence{Prefix: prefix, EntityID: entityID, Value: value, Details: details})
	}
}

func (c *Check) result() Result {
	res := Result{Lists: c.lists, Counts: c.counts, Valid: true}
	for _, l := range c.lists {
		switch l.Rule.Severity {
		case Error:
			res.Valid = false
			res.Errors = append(res.Errors, l.Occurrences...)
		case Warning:
			res.Warnings = append(res.Warnings, l.Occurrences...)
		}
	}
	return res
}
