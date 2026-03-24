# arcgo

`arcgo` 是模块化的 Go 后端基础设施工具集，按包组织、可按需引入，并允许包间依赖组合。

[English](./README.md) | Chinese

> **文档**: 访问 [Hugo 文档站点](./docs/) 获得统一的文档体验。

## 包概览

| 包 | 描述 |
| --- | --- |
| `authx` | 基于 Authboss + Casbin 的 Opinionated 安全抽象层 |
| `collectionx` | 泛型集合与并发安全结构 |
| `configx` | 分层配置加载与校验 |
| `dbx` | 基于 `database/sql` 的 schema-first / generic-first ORM 核心 |
| `dix` | 基于 `do` 的强类型模块化应用框架 |
| `eventx` | 进程内强类型事件总线 |
| `clientx` | 面向协议的客户端包集合（HTTP/TCP/UDP） |
| `httpx` | 多框架统一强类型 HTTP 路由（可选子模块：`httpx/middleware`、`httpx/websocket`） |
| `kvx` | Redis / Valkey 的统一强类型访问框架 |
| `logx` | 结构化日志与 `slog` 互通 |
| `observabilityx` | 可选可观测性抽象（OTel/Prometheus） |

## 安装

快速上手（`@latest`）：

```bash
go get github.com/DaiYuANg/arcgo/dix@latest
go get github.com/DaiYuANg/arcgo/httpx@latest
go get github.com/DaiYuANg/arcgo/dbx@latest
go get github.com/DaiYuANg/arcgo/kvx@latest
```

固定版本（生产/CI 推荐）：

```bash
go get github.com/DaiYuANg/arcgo/authx@v0.0.1
go get github.com/DaiYuANg/arcgo/clientx@v0.0.1
go get github.com/DaiYuANg/arcgo/collectionx@v0.0.1
go get github.com/DaiYuANg/arcgo/configx@v0.0.1
go get github.com/DaiYuANg/arcgo/dbx@v0.0.1
go get github.com/DaiYuANg/arcgo/dix@v0.0.1
go get github.com/DaiYuANg/arcgo/eventx@v0.0.1
go get github.com/DaiYuANg/arcgo/httpx@v0.0.1
go get github.com/DaiYuANg/arcgo/kvx@v0.0.1
go get github.com/DaiYuANg/arcgo/logx@v0.0.1
go get github.com/DaiYuANg/arcgo/observabilityx@v0.0.1
```

可选子模块：

```bash
go get github.com/DaiYuANg/arcgo/dbx/migrate@v0.0.1
go get github.com/DaiYuANg/arcgo/httpx/middleware@v0.0.1
go get github.com/DaiYuANg/arcgo/httpx/websocket@v0.0.1
```

## 快速选择

- 需要容器/数据工具：从 `collectionx` 开始
- 需要基于 Authboss + Casbin 的认证/授权抽象：从 `authx` 开始
- 需要从 `.env` + 文件 + 环境变量加载配置：从 `configx` 开始
- 需要 ORM（schema 建模、查询 DSL、迁移/计划）：从 `dbx` 开始
- 需要纯 SQL 模板（可选解析器校验）：从 `dbx/sqltmplx` 开始（作为 `dbx` 下的子包能力）
- 需要强类型模块化应用框架：从 `dix` 开始
- 需要进程内带类型的事件总线：从 `eventx` 开始
- 需要协议化客户端（HTTP/TCP/UDP 的重试/TLS/hooks 等约定）：从 `clientx` 开始
- 需要跨框架的统一 HTTP 路由：从 `httpx` 开始
- 需要 Redis/Valkey 的强类型对象映射与 repository-style 访问：从 `kvx` 开始
- 需要结构化日志和日志轮转：从 `logx` 开始
- 需要可选的遥测抽象（OTel/Prometheus）：从 `observabilityx` 开始

## 常见组合

- API 服务：`httpx + configx + logx`
- 模块化应用：`dix + configx + logx`
- 单体内事件驱动：`eventx + logx`
- 数据结构/工具层：`collectionx + configx`
- 数据层（ORM + 纯 SQL helper）：`dbx`（包含 `dbx/sqltmplx`）
- Redis/Valkey 集成：`kvx + configx + logx`

## 常用命令

```bash
go tool task fmt
go tool task lint
go tool task test
go tool task check
```

## 文档脚本（跨平台）

```bash
go run ./scripts/deploy-docs help
go run ./scripts/deploy-docs sync
go run ./scripts/deploy-docs build
go run ./scripts/deploy-docs serve
go run ./scripts/deploy-docs deploy
# 可选：
# DOCS_REMOTE=origin DOCS_BRANCH=gh-pages go run ./scripts/deploy-docs deploy
```

## clientx 示例

```bash
go run ./clientx/examples/edge_http
go run ./clientx/examples/internal_rpc_tcp
go run ./clientx/examples/low_latency_udp
```

## Git 提交前 Hook

仓库使用 `lefthook`（通过 `go tool` 管理）。

每个 clone 只需执行一次安装：

```bash
go tool task git:hooks:install
```

手动执行 hook：

```bash
go tool task git:hooks:run
```

`pre-commit` 会在提交前执行：

- `go tool task fmt`
- `go tool task lint`
```

## 说明

- 代码注释统一为英文。
- 所有文档统一在 Hugo 文档站点维护。
- 发布声明：在 Go 的 "generic method" 提案正式发布/落地之前，本库不会进行正式发布；一旦该提案实现，可能会发生大范围破坏性更新。目前阶段不建议在生产环境中使用。
