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

	ifaceArgs := append([]any{command}, lo.Map(args, func(v []byte, _ int) any { return v })...)

	p.pipe.Do(context.Background(), ifaceArgs...)
	return nil
}

// Exec executes all queued commands.
func (p *redisPipeline) Exec(ctx context.Context) ([][]byte, error) {
	cmders, err := p.pipe.Exec(ctx)
	if err != nil {
		return nil, wrapRedisError("execute pipeline", err)
	}

	results := make([][]byte, len(cmders))
	for i, cmder := range cmders {
		cmd, ok := cmder.(*redis.Cmd)
		if !ok {
			return nil, fmt.Errorf("redis execute pipeline: unexpected command type %T", cmder)
		}

		if err := cmd.Err(); err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}

			results[i] = nil
			continue
		}

		results[i] = valueToBytes(cmd.Val())
	}

	return results, nil
}

// Close closes the pipeline.
func (p *redisPipeline) Close() error {
	// Pipeline doesn't need explicit close in go-redis
	return nil
}
