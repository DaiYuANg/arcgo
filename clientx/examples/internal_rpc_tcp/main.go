package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/DaiYuANg/archgo/clientx/preset"
	clienttcp "github.com/DaiYuANg/archgo/clientx/tcp"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer func() { _ = ln.Close() }()

	serverErr := make(chan error, 1)
	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			serverErr <- acceptErr
			return
		}
		defer func() { _ = conn.Close() }()

		reader := bufio.NewReader(conn)
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			serverErr <- readErr
			return
		}
		_, writeErr := conn.Write([]byte("ack:" + strings.TrimSpace(line) + "\n"))
		serverErr <- writeErr
	}()

	client, err := preset.NewInternalRPC(
		clienttcp.Config{Address: ln.Addr().String()},
		preset.WithInternalRPCDisableRetry(),
		preset.WithInternalRPCTimeoutGuard(2*time.Second),
	)
	if err != nil {
		panic(err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := client.Dial(ctx)
	if err != nil {
		panic(err)
	}
	defer func() { _ = conn.Close() }()

	if _, err = conn.Write([]byte("ping\n")); err != nil {
		panic(err)
	}

	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		panic(err)
	}
	fmt.Printf("tcp reply=%q\n", strings.TrimSpace(reply))

	if err = <-serverErr; err != nil {
		panic(err)
	}
}
