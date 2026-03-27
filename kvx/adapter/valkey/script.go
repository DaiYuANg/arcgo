package valkey

import (
	"context"
)

// Load loads a script into the script cache.
func (a *Adapter) Load(ctx context.Context, script string) (string, error) {
	resp := a.client.Do(ctx, a.client.B().ScriptLoad().Script(script).Build())

	return stringFromResult("load script", resp)
}

// Eval executes a script.
func (a *Adapter) Eval(ctx context.Context, script string, keys []string, args [][]byte) ([]byte, error) {
	cmd := a.client.B().Eval().Script(script).Numkeys(int64(len(keys))).Key(keys...).Arg(binaryArgs(args)...)
	resp := a.client.Do(ctx, cmd.Build())

	return bytesFromResult("eval script", resp)
}

// EvalSHA executes a cached script by SHA.
func (a *Adapter) EvalSHA(ctx context.Context, sha string, keys []string, args [][]byte) ([]byte, error) {
	cmd := a.client.B().Evalsha().Sha1(sha).Numkeys(int64(len(keys))).Key(keys...).Arg(binaryArgs(args)...)
	resp := a.client.Do(ctx, cmd.Build())

	return bytesFromResult("eval script sha", resp)
}
