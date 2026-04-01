package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
)

// Pipeline creates a new pipeline.
func (a *Adapter) Pipeline() kvx.Pipeline {
	return &redisPipeline{
		pipe: a.client.Pipeline(),
	}
}

type redisPipeline struct {
	pipe redis.Pipeliner
}

// Enqueue adds a command to the pipeline.
func (p *redisPipeline) Enqueue(command string, args ...[]byte) error {
	if len(args) > kvx.MaxPipelineArgs {
		return kvx.ErrTooManyArgs
	}

	ifaceArgs := lo.Concat([]any{command}, lo.Map(args, func(v []byte, _ int) any { return v }))

	p.pipe.Do(context.Background(), ifaceArgs...)
	return nil
}

// Exec executes all queued commands.
func (p *redisPipeline) Exec(ctx context.Context) ([][]byte, error) {
	cmders, err := p.pipe.Exec(ctx)
	if err != nil {
		return nil, wrapRedisError("execute pipeline", err)
	}

	decodeErr := error(nil)
	results := lo.Reduce(cmders, func(acc [][]byte, cmder redis.Cmder, index int) [][]byte {
		if decodeErr != nil {
			return acc
		}

		cmd, ok := cmder.(*redis.Cmd)
		if !ok {
			decodeErr = fmt.Errorf("redis execute pipeline: unexpected command type %T", cmder)
			return acc
		}

		if err := cmd.Err(); err != nil {
			if !errors.Is(err, redis.Nil) {
				acc[index] = nil
			}
			return acc
		}

		acc[index] = valueToBytes(cmd.Val())
		return acc
	}, make([][]byte, len(cmders)))
	if decodeErr != nil {
		return nil, decodeErr
	}

	return results, nil
}

// Close closes the pipeline.
func (p *redisPipeline) Close() error {
	// Pipeline doesn't need explicit close in go-redis
	return nil
}
