package http_server

import (
	"net/http"
	"strings"
)

type routeNode struct {
	method      string
	path        string
	segments    []string
	handler     HandlerFunc
	middlewares []Middleware
}

type prefixRoute struct {
	prefix      string
	handler     HandlerFunc
	middlewares []Middleware
}

type routeEngine struct {
	routes       []routeNode
	prefixRoutes []prefixRoute
}

// Router 路由管理器，支持路径参数与路由分组。
type Router struct {
	engine      *routeEngine
	middlewares []Middleware
	prefix      string
}

// NewRouter 创建路由器。
func NewRouter() *Router {
	return &Router{
		engine: &routeEngine{},
	}
}

// Group 创建带公共前缀和中间件的路由组。
func (r *Router) Group(prefix string, mws ...Middleware) *Router {
	return &Router{
		engine:      r.engine,
		middlewares: append(append([]Middleware{}, r.middlewares...), mws...),
		prefix:      r.prefix + prefix,
	}
}

// Use 注册全局中间件。
func (r *Router) Use(mws ...Middleware) {
	r.middlewares = append(r.middlewares, mws...)
}

func (r *Router) addRoute(method, path string, handler HandlerFunc) {
	fullPath := r.prefix + path
	if !strings.HasPrefix(fullPath, "/") {
		fullPath = "/" + fullPath
	}
	segments := splitPath(fullPath)
	r.engine.routes = append(r.engine.routes, routeNode{
		method:      method,
		path:        fullPath,
		segments:    segments,
		handler:     handler,
		middlewares: append([]Middleware{}, r.middlewares...),
	})
}

// GET 注册 GET 路由。
func (r *Router) GET(path string, handler HandlerFunc) {
	r.addRoute(http.MethodGet, path, handler)
}

// POST 注册 POST 路由。
func (r *Router) POST(path string, handler HandlerFunc) {
	r.addRoute(http.MethodPost, path, handler)
}

// PUT 注册 PUT 路由。
func (r *Router) PUT(path string, handler HandlerFunc) {
	r.addRoute(http.MethodPut, path, handler)
}

// DELETE 注册 DELETE 路由。
func (r *Router) DELETE(path string, handler HandlerFunc) {
	r.addRoute(http.MethodDelete, path, handler)
}

// PATCH 注册 PATCH 路由。
func (r *Router) PATCH(path string, handler HandlerFunc) {
	r.addRoute(http.MethodPatch, path, handler)
}

// Mount 将标准 http.Handler 挂载到路径前缀，匹配该前缀下所有子路径。
func (r *Router) Mount(path string, handler http.Handler) {
	fullPath := r.prefix + path
	if !strings.HasPrefix(fullPath, "/") {
		fullPath = "/" + fullPath
	}
	fullPath = strings.TrimRight(fullPath, "/")

	h := func(c *Context) error {
		handler.ServeHTTP(c.Writer, c.Request)
		return nil
	}
	r.engine.prefixRoutes = append(r.engine.prefixRoutes, prefixRoute{
		prefix:      fullPath,
		handler:     h,
		middlewares: append([]Middleware{}, r.middlewares...),
	})
}

// Handle 注册任意 HTTP 方法路由。
func (r *Router) Handle(method, path string, handler HandlerFunc) {
	r.addRoute(method, path, handler)
}

func (r *Router) matchPrefix(method, path string) (*prefixRoute, bool) {
	var matched *prefixRoute
	for i := range r.engine.prefixRoutes {
		pr := &r.engine.prefixRoutes[i]
		if !strings.HasPrefix(path, pr.prefix) {
			continue
		}
		if matched == nil || len(pr.prefix) > len(matched.prefix) {
			matched = pr
		}
	}
	if matched == nil {
		return nil, false
	}
	return matched, true
}

// match 匹配路由并提取路径参数。
func (r *Router) match(method, path string) (*routeNode, map[string]string, bool) {
	segments := splitPath(path)
	for i := range r.engine.routes {
		route := &r.engine.routes[i]
		if route.method != method {
			continue
		}
		params, ok := matchSegments(route.segments, segments)
		if ok {
			return route, params, true
		}
	}
	return nil, nil, false
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func matchSegments(pattern, actual []string) (map[string]string, bool) {
	if len(pattern) != len(actual) {
		return nil, false
	}
	params := make(map[string]string)
	for i, p := range pattern {
		if strings.HasPrefix(p, ":") {
			params[p[1:]] = actual[i]
			continue
		}
		if p != actual[i] {
			return nil, false
		}
	}
	return params, true
}

// buildHandlerChain 构建中间件 + 业务 handler 的执行链。
func buildHandlerChain(mws []Middleware, handler HandlerFunc) []HandlerFunc {
	wrapped := handler
	for i := len(mws) - 1; i >= 0; i-- {
		wrapped = mws[i](wrapped)
	}
	return []HandlerFunc{wrapped}
}

// ServeHTTP 实现 http.Handler，作为 API 接入入口。
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if route, params, ok := r.match(req.Method, req.URL.Path); ok {
		r.serveChain(w, req, route.middlewares, route.handler, params)
		return
	}

	if pr, ok := r.matchPrefix(req.Method, req.URL.Path); ok {
		r.serveChain(w, req, pr.middlewares, pr.handler, nil)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"error":"not found"}`))
}

func (r *Router) serveChain(w http.ResponseWriter, req *http.Request, mws []Middleware, handler HandlerFunc, params map[string]string) {
	chain := buildHandlerChain(mws, handler)
	ctx := newContext(w, req, chain, params)
	if err := chain[0](ctx); err != nil {
		if ctx.index < len(chain) {
			_ = ctx.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
	}
}
