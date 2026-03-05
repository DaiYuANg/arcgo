---
slug: /
title: arcgo - 一套简洁高效的 Go 工具库
sidebar_label: 首页
---

# arcgo

<div style={{textAlign: 'center', fontSize: '1.2rem', marginBottom: '2rem'}}>
一套简洁高效的 Go 工具库
</div>

<div style={{textAlign: 'center', marginBottom: '3rem'}}>
  <a href="/docs/intro" style={{marginRight: '1rem'}} className="button button--primary button--lg">快速开始</a>
  <a href="https://github.com/DaiYuANg/arcgo" className="button button--secondary button--lg">GitHub</a>
</div>

## 📦 模块介绍

### configx - 配置加载

基于 [koanf](https://github.com/knadh/koanf) 和 [validator](https://github.com/go-playground/validator) 的配置加载库。

- ✅ 支持 `.env` 文件加载
- ✅ 支持配置文件 (YAML/JSON/TOML)
- ✅ 支持环境变量
- ✅ 可配置加载优先级
- ✅ 支持默认值
- ✅ 基于 validator 的结构体验证

```bash
go get github.com/DaiYuANg/arcgo/configx
```

[了解更多 →](/docs/modules/configx/overview)

---

### httpx - HTTP 框架适配器

灵活的 HTTP 框架适配器层，支持多种流行的 Go Web 框架。

- **按需引入** - 每个适配器都是独立的子包
- **原生中间件支持** - 直接使用框架原生的中间件生态
- **统一接口** - 所有适配器实现相同的接口
- **Huma OpenAPI 支持** - 支持 OpenAPI 文档生成

支持的框架：
- Gin
- Fiber
- Echo
- 标准库 (基于 chi)

```bash
go get github.com/DaiYuANg/arcgo/httpx/adapter/gin
```

[了解更多 →](/docs/modules/httpx/overview)

---

### logx - 日志记录器

基于 [zerolog](https://github.com/rs/zerolog) 的日志记录器，支持文件轮转和 [oops](https://github.com/samber/oops) 错误追踪集成。

- ✅ 支持控制台和文件输出
- ✅ 支持日志轮转（基于 lumberjack）
- ✅ 支持错误堆栈追踪
- ✅ 支持开发/生产环境预设
- ✅ 简洁易用的 API

```bash
go get github.com/DaiYuANg/arcgo/logx
```

[了解更多 →](/docs/modules/logx/overview)

---

## 🚀 快速开始

```bash
# 安装配置模块
go get github.com/DaiYuANg/arcgo/configx

# 安装日志模块
go get github.com/DaiYuANg/arcgo/logx

# 安装 HTTP 模块（以 Gin 为例）
go get github.com/DaiYuANg/arcgo/httpx/adapter/gin
```

---

## 📖 文档导航

- [快速开始](/docs/quick-start) - 快速了解 arcgo
- [configx 文档](/docs/modules/configx/overview) - 配置加载详解
- [httpx 文档](/docs/modules/httpx/overview) - HTTP 框架适配器详解
- [logx 文档](/docs/modules/logx/overview) - 日志记录器详解

---

## 🔗 相关链接

- [GitHub 仓库](https://github.com/DaiYuANg/arcgo)
- [Issues](https://github.com/DaiYuANg/arcgo/issues)
- [Discussions](https://github.com/DaiYuANg/arcgo/discussions)

---

## 📄 License

MIT License
