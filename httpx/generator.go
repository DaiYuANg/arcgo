package httpx

import (
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/samber/lo"
	"github.com/samber/mo"
)

// RouterGenerator 路由生成器
type RouterGenerator struct {
	opts GeneratorOptions
}

// GeneratorOptions 生成器配置
type GeneratorOptions struct {
	// BasePath 基础路径前缀
	BasePath string
	// UseComment 是否使用注释解析路由
	UseComment bool
	// UseTag 是否使用标签解析路由
	UseTag bool
	// UseNaming 是否使用函数命名解析路由
	UseNaming bool
	// TagKey 标签键名，默认为 "route"
	TagKey string
	// MethodPrefixes 方法前缀映射，默认为 Get/List/Create/Update/Delete
	MethodPrefixes map[string]string
}

// DefaultGeneratorOptions 默认配置
func DefaultGeneratorOptions() GeneratorOptions {
	return GeneratorOptions{
		BasePath:   "",
		UseComment: false,
		UseTag:     true,
		UseNaming:  true,
		TagKey:     "route",
		MethodPrefixes: map[string]string{
			"Get":    MethodGet,
			"List":   MethodGet,
			"Create": MethodPost,
			"Update": MethodPut,
			"Patch":  MethodPatch,
			"Delete": MethodDelete,
		},
	}
}

// NewRouterGenerator 创建路由生成器
func NewRouterGenerator(opts ...GeneratorOptions) *RouterGenerator {
	options := lo.FirstOr(opts, DefaultGeneratorOptions())

	// 确保至少启用一种解析方式
	if !options.UseComment && !options.UseTag && !options.UseNaming {
		options.UseNaming = true
	}

	return &RouterGenerator{opts: options}
}

// Generate 从 endpoint struct 生成路由
// endpoint 必须是指针类型的 struct
func (g *RouterGenerator) Generate(endpoint interface{}) mo.Result[[]RouteInfo] {
	v := reflect.ValueOf(endpoint)
	if v.Kind() != reflect.Ptr {
		return mo.Err[[]RouteInfo](ErrInvalidEndpoint)
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return mo.Err[[]RouteInfo](ErrInvalidEndpoint)
	}

	// 使用指针类型获取所有方法（包括指针接收器的方法）
	ptrType := reflect.TypeOf(endpoint)

	// 排除的辅助方法
	excludedMethods := map[string]struct{}{
		"JSON": {}, "Error": {}, "Success": {},
		"GetHeader": {}, "GetQuery": {}, "GetQueryOrDefault": {},
	}

	// 遍历所有方法（包括指针接收器的方法）
	methodRoutes := lo.FilterMap(lo.Range(ptrType.NumMethod()), func(i int, _ int) (RouteInfo, bool) {
		methodType := ptrType.Method(i)
		methodName := methodType.Name

		// 检查是否是需要排除的方法
		if _, excluded := excludedMethods[methodName]; excluded {
			return RouteInfo{}, false
		}

		// 检查方法是否可以作为 handler
		if !g.isHandlerMethod(methodName) {
			return RouteInfo{}, false
		}

		// 解析路由信息
		route := g.parseRouteInfo(methodName)
		if route != nil {
			route.HandlerName = methodName
			return *route, true
		}
		return RouteInfo{}, false
	})

	// 遍历字段（支持字段方法 + 标签方式）
	fieldRoutes := lo.FilterMap(lo.Range(v.NumField()), func(i int, _ int) (RouteInfo, bool) {
		field := v.Type().Field(i)
		fieldValue := v.Field(i)

		// 检查字段是否是函数类型且有标签
		if field.Type.Kind() == reflect.Func && g.opts.UseTag {
			route := g.parseRouteInfoFromField(field, fieldValue)
			if route != nil {
				return *route, true
			}
		}
		return RouteInfo{}, false
	})

	return mo.Ok(append(methodRoutes, fieldRoutes...))
}

// GenerateOrDie 从 endpoint struct 生成路由，失败则 panic
func (g *RouterGenerator) GenerateOrDie(endpoint interface{}) []RouteInfo {
	return g.Generate(endpoint).MustGet()
}

// isHandlerMethod 检查方法是否为有效的 handler 方法
func (g *RouterGenerator) isHandlerMethod(methodName string) bool {
	// 方法名必须以大写字母开头
	if len(methodName) == 0 || !unicode.IsUpper(rune(methodName[0])) {
		return false
	}

	// 检查方法名是否符合命名规范
	if g.opts.UseNaming {
		return lo.SomeBy(lo.Keys(g.opts.MethodPrefixes), func(prefix string) bool {
			return strings.HasPrefix(methodName, prefix)
		})
	}

	return false
}

// parseRouteInfo 解析方法的路由信息
func (g *RouterGenerator) parseRouteInfo(methodName string) *RouteInfo {
	// 从方法名解析
	if g.opts.UseNaming {
		if route := g.parseFromNaming(methodName); route != nil {
			return route
		}
	}

	return nil
}

// parseRouteInfoFromField 从字段解析路由信息
func (g *RouterGenerator) parseRouteInfoFromField(field reflect.StructField, fieldValue reflect.Value) *RouteInfo {
	// 从标签解析
	if route := g.parseFromTag(field); route != nil {
		route.HandlerName = field.Name
		return route
	}

	return nil
}

// parseFromTag 从标签解析路由
func (g *RouterGenerator) parseFromTag(field reflect.StructField) *RouteInfo {
	// 尝试 http 标签
	httpTag, httpOk := field.Tag.Lookup("http")
	if httpOk {
		return g.parseHTTPTag(httpTag, field.Name)
	}

	// 尝试 route 标签
	routeTag, routeOk := field.Tag.Lookup(g.opts.TagKey)
	if routeOk {
		return g.parseRouteTag(routeTag, field.Name)
	}

	return nil
}

// parseHTTPTag 解析 http 标签，格式："GET /user/list" 或 "/user/list"
func (g *RouterGenerator) parseHTTPTag(tag, fieldName string) *RouteInfo {
	parts := strings.Fields(tag)
	if len(parts) == 0 {
		return nil
	}

	var method, path string
	if len(parts) == 1 {
		method = MethodGet
		path = parts[0]
	} else {
		method = strings.ToUpper(parts[0])
		path = parts[1]
	}

	return &RouteInfo{
		Method:      method,
		Path:        joinRoutePath(g.opts.BasePath, path),
		HandlerName: fieldName,
	}
}

// parseRouteTag 解析 route 标签，格式："GET /user/list" 或 "/user/list"
func (g *RouterGenerator) parseRouteTag(tag, fieldName string) *RouteInfo {
	return g.parseHTTPTag(tag, fieldName)
}

// parseFromNaming 从方法名解析路由
func (g *RouterGenerator) parseFromNaming(methodName string) *RouteInfo {
	// 查找匹配的方法前缀
	var found bool
	var prefix string
	var httpMethod string

	prefixes := lo.Keys(g.opts.MethodPrefixes)
	sort.Slice(prefixes, func(i, j int) bool {
		if len(prefixes[i]) == len(prefixes[j]) {
			return prefixes[i] < prefixes[j]
		}
		return len(prefixes[i]) > len(prefixes[j])
	})

	for _, p := range prefixes {
		method := g.opts.MethodPrefixes[p]
		if strings.HasPrefix(methodName, p) {
			prefix = p
			httpMethod = method
			found = true
			break
		}
	}

	if !found {
		return nil
	}

	remainingName := methodName[len(prefix):]
	path := g.camelToPath(remainingName)

	return &RouteInfo{
		Method:      httpMethod,
		Path:        joinRoutePath(g.opts.BasePath, path),
		HandlerName: methodName,
	}
}

// camelToPath 将驼峰命名转换为路径
func (g *RouterGenerator) camelToPath(name string) string {
	if name == "" {
		return "/"
	}

	var result strings.Builder
	result.WriteByte('/')

	runes := []rune(name)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prevUpper := unicode.IsUpper(runes[i-1])
				nextLower := (i+1 < len(runes)) && unicode.IsLower(runes[i+1])

				if !prevUpper || (prevUpper && nextLower) {
					result.WriteByte('/')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// LoadFromPackage 从包路径加载所有 endpoint 并生成路由（暂未实现）
func (g *RouterGenerator) LoadFromPackage(pkgPath string) mo.Result[[]RouteInfo] {
	// 此功能需要 AST 分析，暂不实现
	return mo.Ok([]RouteInfo{})
}

func joinRoutePath(basePath, path string) string {
	base := normalizeRoutePrefix(basePath)

	if path == "" {
		if base == "" {
			return "/"
		}
		return base
	}

	cleanPath := path
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	if base == "" {
		return cleanPath
	}

	if cleanPath == "/" {
		return base
	}

	return base + cleanPath
}

func normalizeRoutePrefix(prefix string) string {
	clean := strings.TrimSpace(prefix)
	if clean == "" || clean == "/" {
		return ""
	}
	return "/" + strings.Trim(clean, "/")
}
