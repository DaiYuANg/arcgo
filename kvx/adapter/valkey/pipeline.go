package valkey

import (
	"context"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
	"github.com/valkey-io/valkey-go"
)

// Pipeline creates a new pipeline.
func (a *Adapter) Pipeline() kvx.Pipeline {
	return &valkeyPipeline{
		client: a.client,
	}
}

type valkeyPipeline struct {
	client valkey.Client
	cmds   []valkey.Completed
}

// Enqueue adds a command to the pipeline.
func (p *valkeyPipeline) Enqueue(command string, args ...[]byte) error {
	if len(args) > kvx.MaxPipelineArgs {
		return kvx.ErrTooManyArgs
	}

	cmd := p.client.B().Arbitrary(command).Args(binaryArgs(args)...).Build()
	p.cmds = lo.Concat(p.cmds, []valkey.Completed{cmd})
	return nil
}

// Exec executes all queued commands.
func (p *valkeyPipeline) Exec(ctx context.Context) ([][]byte, error) {
	if len(p.cmds) == 0 {
		return nil, nil
	}

	resps := p.client.DoMulti(ctx, p.cmds...)
	readErr := error(nil)
	results := lo.Reduce(resps, func(acc [][]byte, resp valkey.ValkeyResult, index int) [][]byte {
		if readErr != nil {
			return acc
		}
		if err := resp.Error(); err != nil {
			if !valkey.IsValkeyNil(err) {
				acc[index] = nil
			}
			return acc
		}

		value, err := bytesFromResult("read pipeline result", resp)
		if err != nil {
			readErr = err
			return acc
		}
		acc[index] = value
		return acc
	}, make([][]byte, len(resps)))
	if readErr != nil {
		return nil, readErr
	}

	return results, nil
}

// Close closes the pipeline.
func (p *valkeyPipeline) Close() error {
	// No explicit close needed
	return nil
}
