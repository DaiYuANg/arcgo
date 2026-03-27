package valkey

import (
	"context"

	"github.com/DaiYuANg/arcgo/kvx"
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

	argStrs := make([]string, len(args))
	for i, v := range args {
		argStrs[i] = valkey.BinaryString(v)
	}

	cmd := p.client.B().Arbitrary(command).Args(argStrs...).Build()
	p.cmds = append(p.cmds, cmd)
	return nil
}

// Exec executes all queued commands.
func (p *valkeyPipeline) Exec(ctx context.Context) ([][]byte, error) {
	if len(p.cmds) == 0 {
		return nil, nil
	}

	resps := p.client.DoMulti(ctx, p.cmds...)

	results := make([][]byte, len(resps))
	for i, resp := range resps {
		if err := resp.Error(); err != nil {
			if valkey.IsValkeyNil(err) {
				continue
			}

			results[i] = nil
			continue
		}

		value, err := bytesFromResult("read pipeline result", resp)
		if err != nil {
			return nil, err
		}
		results[i] = value
	}

	return results, nil
}

// Close closes the pipeline.
func (p *valkeyPipeline) Close() error {
	// No explicit close needed
	return nil
}
