package dbx

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type IDGenerator interface {
	GenerateID(ctx context.Context, column ColumnMeta) (any, error)
}

type defaultIDGenerator struct {
	mu           sync.Mutex
	lastUnixMs   int64
	snowflakeSeq int64
}

func newDefaultIDGenerator() *defaultIDGenerator {
	return &defaultIDGenerator{}
}

func (g *defaultIDGenerator) GenerateID(_ context.Context, column ColumnMeta) (any, error) {
	switch column.IDStrategy {
	case IDStrategySnowflake:
		return g.nextSnowflakeID(), nil
	case IDStrategyUUID:
		return g.nextUUID(column.UUIDVersion)
	default:
		return nil, fmt.Errorf("dbx: unsupported id strategy %q", column.IDStrategy)
	}
}

func (g *defaultIDGenerator) nextUUID(version string) (string, error) {
	switch version {
	case "", "v7":
		id, err := uuid.NewV7()
		if err != nil {
			return "", err
		}
		return id.String(), nil
	case "v4":
		return uuid.NewString(), nil
	default:
		return "", fmt.Errorf("dbx: unsupported uuid version %q", version)
	}
}

func (g *defaultIDGenerator) nextSnowflakeID() int64 {
	const sequenceMask int64 = (1 << 12) - 1

	g.mu.Lock()
	defer g.mu.Unlock()

	nowMs := time.Now().UnixMilli()
	if nowMs == g.lastUnixMs {
		g.snowflakeSeq = (g.snowflakeSeq + 1) & sequenceMask
		if g.snowflakeSeq == 0 {
			for nowMs <= g.lastUnixMs {
				nowMs = time.Now().UnixMilli()
			}
		}
	} else {
		g.snowflakeSeq = 0
	}
	g.lastUnixMs = nowMs

	// 41-bit timestamp + 12-bit sequence, node id kept as 0 in default generator.
	return (nowMs << 22) | g.snowflakeSeq
}
