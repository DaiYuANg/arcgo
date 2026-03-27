package redis

import (
	"context"
	"errors"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
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
	_, err := a.client.SetArgs(ctx, key, token, goredis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
	}).Result()
	if errors.Is(err, goredis.Nil) {
		return false, nil
	}

	if err != nil {
		return false, wrapRedisError("acquire lock", err)
	}

	return true, nil
}

// Release releases a lock.
func (a *Adapter) Release(ctx context.Context, key, token string) (bool, error) {
	val, err := a.client.Eval(ctx, releaseLockScript, []string{key}, token).Int()
	return val == 1, wrapRedisError("release lock", err)
}

// Extend extends the lock TTL.
func (a *Adapter) Extend(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	val, err := a.client.Eval(ctx, extendLockScript, []string{key}, token, strconv.FormatInt(ttl.Milliseconds(), 10)).Int()
	return val == 1, wrapRedisError("extend lock", err)
}
