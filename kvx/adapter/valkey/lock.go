package valkey

import (
	"context"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-go"
)

const releaseLockScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('DEL', KEYS[1])
end
return 0
`

const extendLockScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return 0
`

// Acquire tries to acquire a lock.
func (a *Adapter) Acquire(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	resp := a.client.Do(ctx, a.client.B().Set().Key(key).Value(token).Nx().Px(ttl).Build())
	if err := resp.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return false, nil
		}

		return false, wrapValkeyError("acquire lock", err)
	}

	return true, nil
}

// Release releases a lock.
func (a *Adapter) Release(ctx context.Context, key, token string) (bool, error) {
	resp, err := a.Eval(ctx, releaseLockScript, []string{key}, [][]byte{[]byte(token)})
	if err != nil {
		return false, err
	}
	return string(resp) == "1", nil
}

// Extend extends the lock TTL.
func (a *Adapter) Extend(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	resp, err := a.Eval(ctx, extendLockScript, []string{key}, [][]byte{
		[]byte(token),
		[]byte(strconv.FormatInt(ttl.Milliseconds(), 10)),
	})
	if err != nil {
		return false, err
	}
	return string(resp) == "1", nil
}
