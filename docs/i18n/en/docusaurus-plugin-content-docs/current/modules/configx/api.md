---
sidebar_position: 4
---

# API Reference

This page lists the key APIs of `configx`.

## Functions

### `Load`

```go
func Load(cfg any, opts ...Option) error
```

Loads configuration into the target struct pointer.

### `LoadConfig`

```go
func LoadConfig(opts ...Option) (*Config, error)
```

Loads configuration and returns a `*Config` object.

### `NewConfig`

```go
func NewConfig(opts ...Option) (*Config, error)
```

Alias of `LoadConfig`.

## Options

### `WithDotenv`

```go
func WithDotenv(files ...string) Option
```

Enable dotenv loading.

### `WithFiles`

```go
func WithFiles(files ...string) Option
```

Set config file paths.

### `WithEnvPrefix`

```go
func WithEnvPrefix(prefix string) Option
```

Set a single env prefix.

### `WithEnvPrefixs`

```go
func WithEnvPrefixs(prefixes ...string) Option
```

Set multiple env prefixes.

### `WithPriority`

```go
func WithPriority(p ...Source) Option
```

Set source priority. Later sources override earlier ones.

Source types:
- `SourceDotenv`
- `SourceFile`
- `SourceEnv`
- `SourceDefault`

### `WithDefaults`

```go
func WithDefaults(m map[string]any) Option
```

Set default key-value pairs.

### `WithValidateLevel`

```go
func WithValidateLevel(level ValidateLevel) Option
```

Set validation level.

Validation levels:
- `ValidateLevelNone`
- `ValidateLevelStruct`
- `ValidateLevelRequired`

### `WithValidator`

```go
func WithValidator(v *validator.Validate) Option
```

Inject a custom validator.

## `Config` methods

```go
func (c *Config) GetString(path string) string
func (c *Config) GetInt(path string) int
func (c *Config) GetInt64(path string) int64
func (c *Config) GetFloat64(path string) float64
func (c *Config) GetBool(path string) bool
func (c *Config) GetDuration(path string) time.Duration
func (c *Config) GetStringSlice(path string) []string
func (c *Config) GetIntSlice(path string) []int
func (c *Config) Get(path string) any
func (c *Config) Exists(path string) bool
func (c *Config) All() map[string]any
func (c *Config) Unmarshal(path string, out any) error
func (c *Config) Cut(path string) *Config
func (c *Config) MarshalJSON() ([]byte, error)
```

## Error handling

Potential errors include:
- `os.ErrNotExist`
- `validator.ValidationErrors`
- parser/decode errors

Example:

```go
var cfg Config
err := configx.Load(&cfg,
    configx.WithFiles("config.yaml"),
    configx.WithValidateLevel(configx.ValidateLevelRequired),
)
if err != nil {
    // handle file/validation/parse errors
}
```
