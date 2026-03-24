package dbx

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/google/uuid"
	"github.com/segmentio/ksuid"
)

type IDGenerator interface {
	GenerateID(ctx context.Context, column ColumnMeta) (any, error)
}

const (
	DefaultNodeID uint16 = 1
	MinNodeID     uint16 = 1
	MaxNodeID     uint16 = 1023
)

type defaultIDGenerator struct {
	mu             sync.Mutex
	nodeID         uint16
	lastUnixMs     int64
	snowflakeSeq   int64
}

func NewSnowflakeGenerator(nodeID uint16) (IDGenerator, error) {
	if nodeID < MinNodeID || nodeID > MaxNodeID {
		return nil, &NodeIDOutOfRangeError{NodeID: nodeID, Min: MinNodeID, Max: MaxNodeID}
	}
	return &defaultIDGenerator{nodeID: nodeID}, nil
}

func ResolveNodeIDFromHostName() uint16 {
	hostName, err := os.Hostname()
	if err != nil || hostName == "" {
		return DefaultNodeID
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(hostName))
	id := uint16(hasher.Sum32() % (uint32(MaxNodeID) + 1))
	if id < MinNodeID {
		return MinNodeID
	}
	return id
}

func newDefaultIDGenerator(nodeID uint16) *defaultIDGenerator {
	return &defaultIDGenerator{nodeID: nodeID}
}

func (g *defaultIDGenerator) GenerateID(_ context.Context, column ColumnMeta) (any, error) {
	switch column.IDStrategy {
	case IDStrategySnowflake:
		return g.nextSnowflakeID(), nil
	case IDStrategyUUID:
		return g.nextUUID(column.UUIDVersion)
	case IDStrategyULID:
		return ulid.Make().String(), nil
	case IDStrategyKSUID:
		return ksuid.New().String(), nil
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

	// 41-bit timestamp + 10-bit node id + 12-bit sequence.
	return (nowMs << 22) | (int64(g.nodeID) << 12) | g.snowflakeSeq
}
