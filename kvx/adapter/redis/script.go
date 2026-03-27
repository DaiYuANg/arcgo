package redis

import (
	"context"
)

// Load loads a script into the script cache.
func (a *Adapter) Load(ctx context.Context, script string) (string, error) {
	sha, err := a.client.ScriptLoad(ctx, script).Result()
	return wrapRedisResult("load script", sha, err)
}

// Eval executes a script.
func (a *Adapter) Eval(ctx context.Context, script string, keys []string, args [][]byte) ([]byte, error) {
	ifaceArgs := make([]any, len(args))
	for i, v := range args {
		ifaceArgs[i] = v
	}

	val, err := a.client.Eval(ctx, script, keys, ifaceArgs...).Result()
	val, err = wrapRedisResult("eval script", val, err)
	if err != nil {
		return nil, err
	}

	return valueToBytes(val), nil
}

// EvalSHA executes a cached script by SHA.
func (a *Adapter) EvalSHA(ctx context.Context, sha string, keys []string, args [][]byte) ([]byte, error) {
	ifaceArgs := make([]any, len(args))
	for i, v := range args {
		ifaceArgs[i] = v
	}

	val, err := a.client.EvalSha(ctx, sha, keys, ifaceArgs...).Result()
	val, err = wrapRedisResult("eval script sha", val, err)
	if err != nil {
		return nil, err
	}

	return valueToBytes(val), nil
}
