---
title: 'clientx'
linkTitle: 'clientx'
description: 'Protocol-Oriented Client Packages with Shared Conventions (HTTP/TCP/UDP)'
weight: 8
---

## clientx

`clientx` is a protocol-oriented client package set for common network protocols.

Current direction:

- First-wave protocols: `http`, `tcp`, `udp`
- Shared config primitives (`RetryConfig`, `TLSConfig`)
- Keep protocol APIs explicit and composable, while sharing engineering conventions

## Roadmap

- Module roadmap: [clientx roadmap](./roadmap)
- Global roadmap: [ArcGo roadmap](../roadmap)

## Current Implementation Snapshot

- `clientx/http`: resty-based HTTP client wrapper with retry/TLS/header options
- `clientx/tcp`: dial + timeout-wrapped connection with optional TLS
- `clientx/udp`: UDP dial/listen baseline with timeout-wrapped connections
- `clientx`: shared typed error model (`Error`, `ErrorKind`, `WrapError`) used in `http/tcp/udp` transport paths
- `clientx`: lightweight hooks (`Hook`, `HookFuncs`) for dial and I/O lifecycle events
- constructors now return interfaces (`http.Client`, `tcp.Client`, `udp.Client`) to keep internal implementation replaceable

## Usage

### HTTP Client (`clientx/http`)

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	clienthttp "github.com/DaiYuANg/arcgo/clientx/http"
)

func main() {
	c := clienthttp.New(clienthttp.Config{
		BaseURL: "https://api.example.com",
		Timeout: 2 * time.Second,
		Retry: clientx.RetryConfig{
			Enabled:    true,
			MaxRetries: 2,
			WaitMin:    100 * time.Millisecond,
			WaitMax:    500 * time.Millisecond,
		},
	})

	resp, err := c.Execute(nil, http.MethodGet, "/health")
	if err != nil {
		if clientx.IsKind(err, clientx.ErrorKindTimeout) {
			fmt.Println("http timeout")
		}
		panic(err)
	}
	fmt.Println(resp.StatusCode())
}
```

### TCP Client (`clientx/tcp`)

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	"github.com/DaiYuANg/arcgo/clientx/tcp"
)

func main() {
	c := tcp.New(tcp.Config{
		Address:      "127.0.0.1:9000",
		DialTimeout:  time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	conn, err := c.Dial(context.Background())
	if err != nil {
		if clientx.IsKind(err, clientx.ErrorKindConnRefused) {
			fmt.Println("tcp conn refused")
		}
		panic(err)
	}
	defer conn.Close()
}
```

### UDP Client (`clientx/udp`)

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	"github.com/DaiYuANg/arcgo/clientx/udp"
)

func main() {
	c := udp.New(udp.Config{
		Address:      "127.0.0.1:9001",
		DialTimeout:  time.Second,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})

	conn, err := c.Dial(context.Background())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte("ping"))
	if err != nil && clientx.IsKind(err, clientx.ErrorKindTimeout) {
		fmt.Println("udp write timeout")
	}
}
```

### Codec Layer (TCP/UDP Only)

`clientx` supports optional codec composition for `tcp` and `udp`.
`http` is still handled by HTTP semantics (`Content-Type`, request body, resty behaviors), so no mandatory codec layer is introduced there.

Built-in codecs:

- `codec.JSON`
- `codec.Text`
- `codec.Bytes`

Custom codec example:

```go
type ReverseCodec struct{}

func (c ReverseCodec) Name() string { return "reverse" }
func (c ReverseCodec) Marshal(v any) ([]byte, error)   { /* ... */ return nil, nil }
func (c ReverseCodec) Unmarshal(data []byte, v any) error { /* ... */ return nil }
```

Register/get by name:

```go
_ = codec.Register(ReverseCodec{})
c := codec.Must("reverse")
_ = c
```

TCP + codec + framer:

```go
cc, err := tcpClient.DialCodec(ctx, codec.JSON, codec.NewLengthPrefixed(1024*1024))
if err != nil {
	panic(err)
}
defer cc.Close()

_ = cc.WriteValue(map[string]string{"message": "ping"})
var out map[string]string
_ = cc.ReadValue(&out)
```

UDP + codec:

```go
uc, err := udpClient.DialCodec(ctx, codec.JSON)
if err != nil {
	panic(err)
}
defer uc.Close()

_ = uc.WriteValue(map[string]string{"message": "ping"})
var out map[string]string
_ = uc.ReadValue(&out)
```

### Hooks (Dial/IO Lifecycle)

`clientx` provides protocol-agnostic hooks:

- `OnDial` for dial/listen lifecycle
- `OnIO` for read/write/request lifecycle

```go
h := clientx.HookFuncs{
	OnDialFunc: func(e clientx.DialEvent) {
		// protocol/op/addr/duration/err
	},
	OnIOFunc: func(e clientx.IOEvent) {
		// protocol/op/bytes/duration/err
	},
}

httpClient := clienthttp.New(cfg, clienthttp.WithHooks(h))
tcpClient := tcp.New(cfg, tcp.WithHooks(h))
udpClient := udp.New(cfg, udp.WithHooks(h))

_, _, _ = httpClient, tcpClient, udpClient
```

observabilityx adapter:

```go
obsHook := clientx.NewObservabilityHook(
	obs,
	clientx.WithHookMetricPrefix("clientx"),
	clientx.WithHookAddressAttribute(false), // default false, avoid high-cardinality addr labels
)

tcpClient := tcp.New(cfg, tcp.WithHooks(obsHook))
_ = tcpClient
```

## Error Handling Conventions

- All transport-level errors are wrapped as `*clientx.Error`.
- Use `clientx.KindOf(err)` or `clientx.IsKind(err, kind)` for category checks.
- Wrapped errors keep `Unwrap()` behavior (`errors.Is`/`errors.As` still works).
- Wrapped timeout errors still satisfy `net.Error` timeout checks.

## Notes

- `clientx` is currently experimental and under active iteration.
- Inter-package dependencies are allowed; current implementation already reuses shared config and `collectionx`.
- Prefer programming against `http.Client` / `tcp.Client` / `udp.Client` instead of concrete structs.
