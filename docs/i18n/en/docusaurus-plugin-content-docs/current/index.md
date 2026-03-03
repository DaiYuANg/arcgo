---
slug: /
title: toolkit4go - A concise and efficient Go toolkit
sidebar_label: Home
---

# toolkit4go

<div style={{textAlign: 'center', fontSize: '1.2rem', marginBottom: '2rem'}}>
A concise and efficient Go toolkit
</div>

<div style={{textAlign: 'center', marginBottom: '3rem'}}>
  <a href="/docs/intro" style={{marginRight: '1rem'}} className="button button--primary button--lg">Quick Start</a>
  <a href="https://github.com/DaiYuANg/toolkit4go" className="button button--secondary button--lg">GitHub</a>
</div>

## Modules

### configx - Configuration loading

A configuration loader built on top of [koanf](https://github.com/knadh/koanf) and [validator](https://github.com/go-playground/validator).

- Supports `.env` files
- Supports config files (YAML/JSON/TOML)
- Supports environment variables
- Supports custom source priority
- Supports defaults
- Supports struct validation via validator

```bash
go get github.com/DaiYuANg/toolkit4go/configx
```

[Learn more ->](/docs/modules/configx/overview)

---

### httpx - HTTP framework adapters

A flexible adapter layer for popular Go web frameworks.

- `on-demand imports`: every adapter is an independent subpackage
- `native middleware`: use each framework's middleware ecosystem directly
- `unified abstractions`: shared adapter interface
- `Huma OpenAPI integration`: built-in OpenAPI documentation support

Supported frameworks:
- Gin
- Fiber
- Echo
- Standard library (based on chi)

```bash
go get github.com/DaiYuANg/toolkit4go/httpx/adapter/gin
```

[Learn more ->](/docs/modules/httpx/overview)

---

### logx - Logger

A logger based on [zerolog](https://github.com/rs/zerolog), with file rotation and [oops](https://github.com/samber/oops) error tracking integration.

- Console and file output
- Log rotation (powered by lumberjack)
- Error stack tracing
- Development/production presets
- Simple API

```bash
go get github.com/DaiYuANg/toolkit4go/logx
```

[Learn more ->](/docs/modules/logx/overview)

---

## Quick Start

```bash
# Install config module
go get github.com/DaiYuANg/toolkit4go/configx

# Install logging module
go get github.com/DaiYuANg/toolkit4go/logx

# Install HTTP module (Gin example)
go get github.com/DaiYuANg/toolkit4go/httpx/adapter/gin
```

---

## Documentation

- [Quick Start](/docs/quick-start)
- [configx docs](/docs/modules/configx/overview)
- [httpx docs](/docs/modules/httpx/overview)
- [logx docs](/docs/modules/logx/overview)

---

## Links

- [GitHub Repository](https://github.com/DaiYuANg/toolkit4go)
- [Issues](https://github.com/DaiYuANg/toolkit4go/issues)
- [Discussions](https://github.com/DaiYuANg/toolkit4go/discussions)

---

## License

MIT License
