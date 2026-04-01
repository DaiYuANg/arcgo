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
	results := make([][]byte, len(resps))
	for index, resp := range resps {
		value, shouldSet, readErr := decodePipelineResult(resp)
		if readErr != nil {
			return nil, readErr
		}
		if shouldSet {
			results[index] = value
		}
	}

	return results, nil
}

func decodePipelineResult(resp valkey.ValkeyResult) ([]byte, bool, error) {
	if err := resp.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, false, nil
		}
		return nil, true, nil
	}

	value, err := bytesFromResult("read pipeline result", resp)
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

// Close closes the pipeline.
func (p *valkeyPipeline) Close() error {
	// No explicit close needed
	return nil
}
