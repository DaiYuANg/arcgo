---
title: 'clientx'
linkTitle: 'clientx'
description: '协议导向客户端包集（HTTP/TCP/UDP）与共享工程约束'
weight: 8
---

## clientx

`clientx` 是面向常见网络协议的协议导向客户端包集。

当前方向：

- 首批协议：`http`、`tcp`、`udp`
- 共享配置原语（`RetryConfig`、`TLSConfig`）
- 保持协议 API 显式与可组合，并共享工程约束

## 路线图

- 模块路线图：[clientx roadmap](./roadmap)
- 全局路线图：[ArcGo roadmap](../roadmap)

## 当前实现快照

- `clientx/http`：基于 resty 的 HTTP client 封装（重试/TLS/header 选项）
- `clientx/tcp`：带超时封装连接与可选 TLS 的拨号能力
- `clientx/udp`：已提供 UDP dial/listen 基线能力与超时封装连接
- `clientx`：已提供共享 typed error 模型（`Error`、`ErrorKind`、`WrapError`），用于 `http/tcp/udp` 传输错误路径
- `clientx`：已提供轻量 hooks（`Hook`、`HookFuncs`），覆盖 dial 与 I/O 生命周期事件
- 构造函数已返回接口（`http.Client`、`tcp.Client`、`udp.Client`），以保证内部实现可替换

## 使用方式

### HTTP 客户端（`clientx/http`）

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

### TCP 客户端（`clientx/tcp`）

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

### UDP 客户端（`clientx/udp`）

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

### Codec 层（仅 TCP/UDP）

`clientx` 在 `tcp` 与 `udp` 上提供可选 codec 组合能力。  
`http` 仍按 HTTP 语义处理（`Content-Type`、请求体、resty 行为），不强制引入 codec 层。

内置 codec：

- `codec.JSON`
- `codec.Text`
- `codec.Bytes`

自定义 codec 示例：

```go
type ReverseCodec struct{}

func (c ReverseCodec) Name() string { return "reverse" }
func (c ReverseCodec) Marshal(v any) ([]byte, error)   { /* ... */ return nil, nil }
func (c ReverseCodec) Unmarshal(data []byte, v any) error { /* ... */ return nil }
```

注册并按名称获取：

```go
_ = codec.Register(ReverseCodec{})
c := codec.Must("reverse")
_ = c
```

TCP + codec + framer：

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

UDP + codec：

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

### Hooks（Dial/IO 生命周期）

`clientx` 提供协议无关的 hooks：

- `OnDial`：拨号/监听生命周期
- `OnIO`：读写/请求生命周期

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

observabilityx 适配器：

```go
obsHook := clientx.NewObservabilityHook(
	obs,
	clientx.WithHookMetricPrefix("clientx"),
	clientx.WithHookAddressAttribute(false), // 默认 false，避免高基数 addr 标签
)

tcpClient := tcp.New(cfg, tcp.WithHooks(obsHook))
_ = tcpClient
```

## 错误处理约定

- 传输层错误统一封装为 `*clientx.Error`。
- 使用 `clientx.KindOf(err)` 或 `clientx.IsKind(err, kind)` 进行类别判断。
- 封装后仍保留 `Unwrap()` 语义（`errors.Is`/`errors.As` 仍可用）。
- 超时错误封装后仍满足 `net.Error` 的超时判断语义。

## 说明

- `clientx` 当前处于实验阶段，仍在快速迭代。
- 包间依赖允许；当前实现已复用共享配置与 `collectionx`。
- 建议业务侧优先依赖 `http.Client` / `tcp.Client` / `udp.Client` 接口，而不是具体结构体。
