package redis

import (
	"context"

	"github.com/DaiYuANg/arcgo/kvx"
)

// XInfoGroups gets info about consumer groups.
func (a *Adapter) XInfoGroups(ctx context.Context, key string) ([]kvx.GroupInfo, error) {
	result, err := a.client.XInfoGroups(ctx, key).Result()
	result, err = wrapRedisResult("get stream groups info", result, err)
	if err != nil {
		return nil, err
	}

	return convertGroupInfos(result), nil
}

// XInfoConsumers gets info about consumers in a group.
func (a *Adapter) XInfoConsumers(ctx context.Context, key, group string) ([]kvx.ConsumerInfo, error) {
	result, err := a.client.XInfoConsumers(ctx, key, group).Result()
	result, err = wrapRedisResult("get stream consumers info", result, err)
	if err != nil {
		return nil, err
	}

	return convertConsumerInfos(result), nil
}

// XInfoStream gets info about a stream.
func (a *Adapter) XInfoStream(ctx context.Context, key string) (*kvx.StreamInfo, error) {
	result, err := a.client.XInfoStream(ctx, key).Result()
	result, err = wrapRedisResult("get stream info", result, err)
	if err != nil {
		return nil, err
	}

	return convertStreamInfo(result), nil
}
