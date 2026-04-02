package valkey

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
)

// XPending gets pending entries information.
func (a *Adapter) XPending(_ context.Context, _, _ string) (*kvx.PendingInfo, error) {
	return nil, errValkeyUnsupported("XPending")
}

// XPendingRange gets pending entries in a range.
func (a *Adapter) XPendingRange(_ context.Context, _, _, _, _ string, _ int64) (collectionx.List[kvx.PendingEntry], error) {
	return nil, errValkeyUnsupported("XPendingRange")
}

// XClaim claims pending entries for a consumer.
func (a *Adapter) XClaim(_ context.Context, _, _, _ string, _ time.Duration, _ []string) (collectionx.List[kvx.StreamEntry], error) {
	return nil, errValkeyUnsupported("XClaim")
}

// XAutoClaim auto-claims pending entries.
func (a *Adapter) XAutoClaim(_ context.Context, _, _, _ string, _ time.Duration, _ string, _ int64) (string, collectionx.List[kvx.StreamEntry], error) {
	return "", nil, errValkeyUnsupported("XAutoClaim")
}

// XInfoGroups gets info about consumer groups.
func (a *Adapter) XInfoGroups(_ context.Context, _ string) (collectionx.List[kvx.GroupInfo], error) {
	return nil, errValkeyUnsupported("XInfoGroups")
}

// XInfoConsumers gets info about consumers in a group.
func (a *Adapter) XInfoConsumers(_ context.Context, _, _ string) (collectionx.List[kvx.ConsumerInfo], error) {
	return nil, errValkeyUnsupported("XInfoConsumers")
}

// XInfoStream gets info about a stream.
func (a *Adapter) XInfoStream(_ context.Context, _ string) (*kvx.StreamInfo, error) {
	return nil, errValkeyUnsupported("XInfoStream")
}
