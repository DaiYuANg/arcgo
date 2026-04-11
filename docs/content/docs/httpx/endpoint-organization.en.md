---
title: 'httpx Endpoint Organization'
linkTitle: 'endpoint-organization'
description: 'Organize routes with Endpoint, Registrar, and EndpointSpec'
weight: 5
---

## Endpoint organization

`httpx` keeps typed route registration explicit, but it also provides a narrower endpoint pattern for larger services.

Use it when you want to:

- group related routes into modules
- apply one prefix to a whole endpoint
- apply default tags, security, parameters, descriptions, or summary prefixes once
- keep service bootstrap code shorter

## Preferred contracts

- `Endpoint`: implement `Register(registrar)` and keep endpoint code scoped to registration concerns.
- `Registrar`: a narrow registration scope. Use `Scope()` for the current endpoint group and `Group(prefix)` for nested groups.
- `EndpointSpecProvider`: implement `EndpointSpec()` when you want endpoint-level defaults like a shared prefix or tags.

## Minimal example

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

This keeps endpoint registration explicit while letting `httpx` apply endpoint defaults before routes are added.

## Why package-level route helpers stay

The preferred endpoint shape is narrower, but the typed route helpers remain package-level functions:

- `httpx.MustGroupGet`
- `httpx.MustGroupPost`
- `httpx.MustGroupPut`
- `httpx.MustAuto`

This is intentional. Go still does not support type-parameterized methods, so moving these helpers onto `Registrar` would weaken the current strong typing.

## Auto route naming

`httpx.Auto(...)` is intentionally limited. It infers the HTTP method and scoped path from the handler method name:

- `List` -> `GET ""`
- `GetByID` -> `GET "/{id}"`
- `Create` -> `POST ""`
- `UpdateByID` -> `PUT "/{id}"`
- `DeleteByID` -> `DELETE "/{id}"`
- `ListProfiles` -> `GET "/profiles"`
- `GetProfileByID` -> `GET "/profile/{id}"`

This is meant as a small syntax sugar for common endpoint shapes, not a general-purpose controller reflection system. When the name no longer reads cleanly, prefer explicit route registration.

## What `EndpointSpec` can define

- `Prefix`
- `Tags`
- `Security`
- `Parameters`
- `SummaryPrefix`
- `Description`
- `ExternalDocs`
- `Extensions`

These are applied through the existing group-level defaults. The runtime model stays the same.

## Compatibility

- New code should implement `Endpoint`.
- Legacy `RegisterRoutes(server)` and `RegisterGroupRoutes(group)` endpoints still work.
- `BaseEndpoint` stays available for legacy code, but new endpoints should not need it.
- If an endpoint implements `EndpointSpec()` but not a supported registration contract, the spec is ignored.

## Example

- Runnable example: [examples/httpx/endpoint](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/endpoint)
