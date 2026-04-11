---
title: 'httpx Endpoint Organization'
linkTitle: 'endpoint-organization'
description: '使用 Endpoint、Registrar、EndpointSpec 组织路由'
weight: 5
---

## Endpoint 组织

`httpx` 底层的强类型路由注册仍然保持显式，但对更大的服务也提供了一套更收敛的 endpoint 模式。

适合在这些场景使用：

- 把相关路由组织成模块
- 给整个 endpoint 统一加前缀
- 一次性应用默认 tags、security、parameters、description、summary prefix
- 让服务启动代码更短

## 推荐接口

- `Endpoint`：实现 `Register(registrar)`，把 endpoint 代码限制在“注册路由”这个职责里。
- `Registrar`：窄化后的注册作用域。`Scope()` 取当前 endpoint group，`Group(prefix)` 继续创建子分组。
- `EndpointSpecProvider`：可选元信息接口。实现 `EndpointSpec()`，声明 endpoint 级默认值。

## 最小示例

```go
type UserEndpoint struct{}

func (e *UserEndpoint) EndpointSpec() httpx.EndpointSpec {
	return httpx.EndpointSpec{
		Prefix:        "/api/v1/users",
		Tags:          httpx.Tags("users", "v1"),
		SummaryPrefix: "Users",
		Description:   "User management endpoints",
	}
}

func (e *UserEndpoint) Register(registrar httpx.Registrar) {
	httpx.MustAuto(registrar,
		httpx.Auto(e.List),
		httpx.Auto(e.GetByID),
		httpx.Auto(e.Create),
	)
}

func (e *UserEndpoint) List(ctx context.Context, _ *struct{}) (*listUsersOutput, error) {
	out := &listUsersOutput{}
	out.Body.Users = []string{"Alice", "Bob"}
	return out, nil
}

func (e *UserEndpoint) GetByID(ctx context.Context, input *getUserInput) (*getUserOutput, error) {
	out := &getUserOutput{}
	out.Body.ID = input.ID
	return out, nil
}

func (e *UserEndpoint) Create(ctx context.Context, input *createUserInput) (*createUserOutput, error) {
	out := &createUserOutput{}
	out.Body = input.Body
	return out, nil
}

func main() {
	server := httpx.New(...)
	server.RegisterOnly(&UserEndpoint{})
}
```

这种写法会把 endpoint 变成一个明确的注册点，`httpx` 会先应用 endpoint 默认值，再注册具体路由。

## 为什么保留包级别路由函数

新的 endpoint 入口更窄了，但强类型路由注册仍然保留包级别函数：

- `httpx.MustGroupGet`
- `httpx.MustGroupPost`
- `httpx.MustGroupPut`
- `httpx.MustAuto`

这是有意为之。Go 目前仍然不支持带类型参数的方法，如果强行把这些能力放到 `Registrar` 上，反而会削弱现在的静态类型能力。

## Auto 命名规则

`httpx.Auto(...)` 是一个受限语法糖，只根据 handler 方法名推导 HTTP method 和当前作用域下的 path：

- `List` -> `GET ""`
- `GetByID` -> `GET "/{id}"`
- `Create` -> `POST ""`
- `UpdateByID` -> `PUT "/{id}"`
- `DeleteByID` -> `DELETE "/{id}"`
- `ListProfiles` -> `GET "/profiles"`
- `GetProfileByID` -> `GET "/profile/{id}"`

它的定位只是给常见 endpoint 形态减一点样板代码，不是一个通用 controller 反射系统。命名一旦开始变得别扭，就应该回到显式路由注册。

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

- 新代码应优先实现 `Endpoint`。
- 旧的 `RegisterRoutes(server)` 和 `RegisterGroupRoutes(group)` 仍然兼容。
- `BaseEndpoint` 继续保留给旧代码使用，但新 endpoint 一般不再需要它。
- 如果 endpoint 实现了 `EndpointSpec()` 但没有实现受支持的注册接口，spec 会被忽略。

## 示例

- 可运行示例：[examples/httpx/endpoint](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/endpoint)
