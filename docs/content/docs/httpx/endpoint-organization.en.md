---
title: 'httpx Endpoint Organization'
linkTitle: 'endpoint-organization'
description: 'Organize routes with Endpoint, GroupEndpoint, and EndpointSpec'
weight: 5
---

## Endpoint organization

`httpx` keeps route registration simple at the bottom level, but it also provides an optional endpoint pattern for larger services.

Use it when you want to:

- group related routes into modules
- apply one prefix to a whole endpoint
- apply default tags, security, parameters, descriptions, or summary prefixes once
- keep service bootstrap code shorter

## Three levels

- `Endpoint`: the original minimal interface. Implement `RegisterRoutes(server)` and register routes however you like.
- `GroupEndpoint`: an optional interface. Implement `RegisterGroupRoutes(group)` to receive a scoped `httpx.Group`.
- `EndpointSpecProvider`: an optional metadata interface. Implement `EndpointSpec()` to define endpoint-level defaults.

## Minimal example

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

This produces one grouped registration point and lets `httpx` apply endpoint defaults before routes are added.

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

- Existing `Endpoint` implementations keep working.
- If an endpoint only implements `RegisterRoutes(server)`, nothing changes.
- If an endpoint implements `EndpointSpec()` but not `GroupEndpoint`, the spec is ignored and registration falls back to the old path.

## Example

- Runnable example: [examples/httpx/endpoint](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/endpoint)
