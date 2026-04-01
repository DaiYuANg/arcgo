package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
)

const hashUpsertScript = `
local key = KEYS[1]
local removeCount = tonumber(ARGV[1])
local ttlMs = tonumber(ARGV[2])
local fieldCount = tonumber(ARGV[3])
local argIndex = 4

if removeCount > 0 then
	for i = 1, removeCount do
		redis.call('DEL', KEYS[i + 1])
	end
end

local hashArgs = {}
for i = 1, fieldCount * 2 do
	hashArgs[i] = ARGV[argIndex]
	argIndex = argIndex + 1
end

if #hashArgs > 0 then
	redis.call('HSET', key, unpack(hashArgs))
end

if ttlMs > 0 then
	redis.call('PEXPIRE', key, ttlMs)
end

for i = removeCount + 2, #KEYS do
	redis.call('SET', KEYS[i], '1')
end

return 1
`

const jsonUpsertScript = `
local key = KEYS[1]
local removeCount = tonumber(ARGV[1])
local ttlMs = tonumber(ARGV[2])
local payload = ARGV[3]

if removeCount > 0 then
	for i = 1, removeCount do
		redis.call('DEL', KEYS[i + 1])
	end
end

redis.call('JSON.SET', key, '$', payload)

if ttlMs > 0 then
	redis.call('PEXPIRE', key, ttlMs)
end

for i = removeCount + 2, #KEYS do
	redis.call('SET', KEYS[i], '1')
end

return 1
`

const deleteScript = `
for i = 2, #KEYS do
	redis.call('DEL', KEYS[i])
end
redis.call('DEL', KEYS[1])
return 1
`

const hashFieldUpdateScript = `
local key = KEYS[1]
redis.call('HSET', key, ARGV[1], ARGV[2])
if #KEYS >= 2 then
	redis.call('DEL', KEYS[2])
end
if #KEYS >= 3 then
	redis.call('SET', KEYS[3], '1')
end
return 1
`

const jsonFieldUpdateScript = `
local key = KEYS[1]
redis.call('JSON.SET', key, ARGV[1], ARGV[2])
if #KEYS >= 2 then
	redis.call('DEL', KEYS[2])
end
if #KEYS >= 3 then
	redis.call('SET', KEYS[3], '1')
end
return 1
`

func execHashUpsertScript(ctx context.Context, script kvx.Script, key string, hashData map[string][]byte, expiration time.Duration, removeEntries, addEntries []string) error {
	keys := lo.Concat([]string{key}, removeEntries, addEntries)
	args := collectionx.NewListWithCapacity[[]byte](3+len(hashData)*2,
		[]byte(strconv.Itoa(len(removeEntries))),
		[]byte(strconv.FormatInt(expiration.Milliseconds(), 10)),
		[]byte(strconv.Itoa(len(hashData))),
	)
	args.Add(encodeHashData(hashData)...)

	return evalScript(ctx, script, hashUpsertScript, keys, args.Values(), "execute hash upsert script")
}

func execJSONUpsertScript(ctx context.Context, script kvx.Script, key string, payload []byte, expiration time.Duration, removeEntries, addEntries []string) error {
	keys := lo.Concat([]string{key}, removeEntries, addEntries)
	args := [][]byte{
		[]byte(strconv.Itoa(len(removeEntries))),
		[]byte(strconv.FormatInt(expiration.Milliseconds(), 10)),
		payload,
	}
	return evalScript(ctx, script, jsonUpsertScript, keys, args, "execute JSON upsert script")
}

func execDeleteScript(ctx context.Context, script kvx.Script, key string, removeEntries []string) error {
	keys := lo.Concat([]string{key}, removeEntries)
	return evalScript(ctx, script, deleteScript, keys, nil, "execute delete script")
}

func execHashFieldUpdateScript(ctx context.Context, script kvx.Script, key, field string, value []byte, removeEntries, addEntries []string) error {
	keys := buildFieldUpdateKeys(key, removeEntries, addEntries)
	args := [][]byte{[]byte(field), value}
	return evalScript(ctx, script, hashFieldUpdateScript, keys, args, "execute hash field update script")
}

func execJSONFieldUpdateScript(ctx context.Context, script kvx.Script, key, path string, value []byte, removeEntries, addEntries []string) error {
	keys := buildFieldUpdateKeys(key, removeEntries, addEntries)
	args := [][]byte{[]byte(path), value}
	return evalScript(ctx, script, jsonFieldUpdateScript, keys, args, "execute JSON field update script")
}

func buildFieldUpdateKeys(key string, removeEntries, addEntries []string) []string {
	keys := collectionx.NewListWithCapacity[string](3, key)
	if len(removeEntries) > 0 {
		keys.Add(removeEntries[0])
	}
	if len(addEntries) > 0 {
		keys.Add(addEntries[0])
	}
	return keys.Values()
}

func evalScript(ctx context.Context, script kvx.Script, source string, keys []string, args [][]byte, action string) error {
	_, err := script.Eval(ctx, source, keys, args)
	return wrapRepositoryError(err, action)
}
