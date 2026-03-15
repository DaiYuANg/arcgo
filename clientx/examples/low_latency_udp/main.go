package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/DaiYuANg/archgo/clientx/preset"
	clientudp "github.com/DaiYuANg/archgo/clientx/udp"
)

func main() {
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer func() { _ = server.Close() }()

	serverErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 128)
		n, addr, readErr := server.ReadFrom(buf)
		if readErr != nil {
			serverErr <- readErr
			return
		}
		_, writeErr := server.WriteTo([]byte("ack:"+string(buf[:n])), addr)
		serverErr <- writeErr
	}()

	client, err := preset.NewLowLatencyUDP(
		clientudp.Config{Address: server.LocalAddr().String()},
		preset.WithLowLatencyUDPReadTimeout(500*time.Millisecond),
		preset.WithLowLatencyUDPWriteTimeout(500*time.Millisecond),
		preset.WithLowLatencyUDPTimeoutGuard(700*time.Millisecond),
	)
	if err != nil {
		panic(err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := client.Dial(ctx)
	if err != nil {
		panic(err)
	}
	defer func() { _ = conn.Close() }()

	if _, err = conn.Write([]byte("ping")); err != nil {
		panic(err)
	}

	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil {
		panic(err)
	}
	fmt.Printf("udp reply=%q\n", strings.TrimSpace(string(buf[:n])))

	if err = <-serverErr; err != nil {
		panic(err)
	}
}
