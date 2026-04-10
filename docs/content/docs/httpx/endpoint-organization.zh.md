---
title: 'httpx Endpoint Organization'
linkTitle: 'endpoint-organization'
description: '使用 Endpoint、GroupEndpoint、EndpointSpec 组织路由'
weight: 5
---

## Endpoint 组织

`httpx` 底层的路由注册保持简单，但对更大的服务也提供了一套可选的 endpoint 模式。

适合在这些场景使用：

- 把相关路由组织成模块
- 给整个 endpoint 统一加前缀
- 一次性应用默认 tags、security、parameters、description、summary prefix
- 让服务启动代码更短

## 三层接口

- `Endpoint`：最原始的最小接口。实现 `RegisterRoutes(server)`，自己决定怎么注册。
- `GroupEndpoint`：可选接口。实现 `RegisterGroupRoutes(group)`，由 `httpx` 传入一个 scoped `httpx.Group`。
- `EndpointSpecProvider`：可选元信息接口。实现 `EndpointSpec()`，声明 endpoint 级默认值。

## 最小示例

```go
type UserEndpoint struct {
	httpx.BaseEndpoint
}

func (e *UserEndpoint) EndpointSpec() httpx.EndpointSpec {
	return httpx.EndpointSpec{
		Prefix:        "/api/v1/users",
		Tags:          []string{"users", "v1"},
		SummaryPrefix: "Users",
		Description:   "User management endpoints",
	}
}

func (e *UserEndpoint) RegisterGroupRoutes(group *httpx.Group) {
	httpx.MustGroupGet(group, "", func(ctx context.Context, _ *struct{}) (*listUsersOutput, error) {
		out := &listUsersOutput{}
		out.Body.Users = []string{"Alice", "Bob"}
		return out, nil
	}, func(op *huma.Operation) {
		op.Summary = "List"
	})
}

func main() {
	server := httpx.New(...)
	server.RegisterOnly(&UserEndpoint{})
}
```

这种写法会把 endpoint 变成一个明确的分组注册点，`httpx` 会先应用 endpoint 默认值，再注册具体路由。

## `EndpointSpec` 能定义什么

- `Prefix`
- `Tags`
- `Security`
- `Parameters`
- `SummaryPrefix`
- `Description`
- `ExternalDocs`
- `Extensions`

这些值最终都通过已有的 group-level default 机制生效，运行时模型没有变。

## 兼容性

- 现有 `Endpoint` 实现不需要修改。
- 如果 endpoint 只实现 `RegisterRoutes(server)`，行为完全不变。
- 如果 endpoint 实现了 `EndpointSpec()` 但没有实现 `GroupEndpoint`，spec 会被忽略，注册仍然走旧路径。

## 示例

- 可运行示例：[examples/httpx/endpoint](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/endpoint)
