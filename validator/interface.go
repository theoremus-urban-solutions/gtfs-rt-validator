package validator

import (
	"context"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"your.org/gtfsrt/validator/notice"
	"your.org/gtfsrt/validator/schema"
)

type FeedEntityValidator interface {
	Validate(ctx context.Context, now int64, meta *schema.Metadata, feed, prev, combined *rt.FeedMessage) []notice.ErrorList
}
