package repository_test

import (
	"context"
	"time"
)

type mockPipeline struct {
	kv       *mockKV
	commands []pipelineCmd
}

type pipelineCmd struct {
	name string
	args [][]byte
}

func (m *mockPipeline) Enqueue(command string, args ...[]byte) error {
	m.commands = append(m.commands, pipelineCmd{name: command, args: args})
	return nil
}

func (m *mockPipeline) Exec(_ context.Context) ([][]byte, error) {
	results := make([][]byte, 0, len(m.commands))
	for _, command := range m.commands {
		m.execCommand(command)
		results = append(results, []byte("OK"))
	}

	return results, nil
}

func (m *mockPipeline) execCommand(command pipelineCmd) {
	switch command.name {
	case "HSET":
		m.execHSet(command.args)
	case "EXPIRE":
		m.execExpire(command.args)
	}
}

func (m *mockPipeline) execHSet(args [][]byte) {
	if len(args) < 3 {
		return
	}

	key := string(args[0])
	m.kv.data[key] = []byte("hash")
}

func (m *mockPipeline) execExpire(args [][]byte) {
	if len(args) < 2 {
		return
	}

	key := string(args[0])
	m.kv.expiration[key] = time.Hour
}

func (m *mockPipeline) Close() error {
	return nil
}
