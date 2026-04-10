---
title: 'authx 快速开始'
linkTitle: 'getting-started'
description: '用最小示例跑通 Engine.Check 与 Engine.Can'
weight: 2
---

## 快速开始

本页演示一个不依赖 Web 框架的 **`authx` 核心示例**：自定义凭证类型、`ProviderManager`、`Engine.Check` 与 `Engine.Can`。

`authx` **不提供**内置的密码 / JWT / OTP 等具体凭证类型；由业务定义 struct，并实现对应的 `AuthenticationProvider`。

## 1）安装

```bash
go get github.com/DaiYuANg/arcgo/authx@latest
```

## 2）创建 `main.go`

示例定义 `usernamePassword` 凭证、注册一个泛型 provider，并演示一个「全部放行」的 `Authorizer`（仅用于跑通流程）。

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DaiYuANg/arcgo/authx"
)

// usernamePassword 由业务自定义；authx 核心保持机制无关。
type usernamePassword struct {
	Username string
	Password string
}

func main() {
	ctx := context.Background()

	engine := authx.NewEngine(
		authx.WithAuthenticationManager(
			authx.NewProviderManager(
				authx.NewAuthenticationProviderFunc(func(
					_ context.Context,
					in usernamePassword,
				) (authx.AuthenticationResult, error) {
					if in.Username != "alice" || in.Password != "secret" {
						return authx.AuthenticationResult{}, fmt.Errorf("invalid credentials")
					}
					return authx.AuthenticationResult{
						Principal: authx.Principal{ID: in.Username},
					}, nil
				}),
			),
		),
		authx.WithAuthorizer(authx.AuthorizerFunc(func(
			_ context.Context,
			_ authx.AuthorizationModel,
		) (authx.Decision, error) {
			return authx.Decision{Allowed: true}, nil
		})),
	)

	result, err := engine.Check(ctx, usernamePassword{Username: "alice", Password: "secret"})
	if err != nil {
		log.Fatal(err)
	}

	decision, err := engine.Can(ctx, authx.AuthorizationModel{
		Principal: result.Principal,
		Action:    "query",
		Resource:  "order",
	})
	if err != nil {
		log.Fatal(err)
	}
	if !decision.Allowed {
		log.Fatal("authorization denied")
	}

	log.Println("ok", result.Principal)
}
```

## 3）运行

```bash
go mod init example.com/authx-minimal
go get github.com/DaiYuANg/arcgo/authx@latest
go run .
```

## 下一步

- HTTP `Guard` 与 std adapter（`chi + net/http`）：[HTTP 集成](./http-integration)
- 各框架适配与更多示例：见 [authx 文档首页](../) 的包布局与仓库示例链接
