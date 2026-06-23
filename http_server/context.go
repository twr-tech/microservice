package http_server

import (
	"context"
	"encoding/json"
	"net/http"
)

// HandlerFunc 是业务处理函数。
type HandlerFunc func(*Context) error

// Context 封装 HTTP 请求与响应，提供统一的 API 接入能力。
type Context struct {
	Request  *http.Request
	Writer   http.ResponseWriter
	params   map[string]string
	index    int
	handlers []HandlerFunc
	keys     map[string]any
}

func newContext(w http.ResponseWriter, r *http.Request, handlers []HandlerFunc, params map[string]string) *Context {
	return &Context{
		Request:  r,
		Writer:   w,
		params:   params,
		handlers: handlers,
		keys:     make(map[string]any),
	}
}

// Next 执行下一个 handler（中间件链核心）。
func (c *Context) Next() error {
	c.index++
	if c.index < len(c.handlers) {
		return c.handlers[c.index](c)
	}
	return nil
}

// Param 获取路由路径参数，如 /users/:id 中的 id。
func (c *Context) Param(key string) string {
	if c.params == nil {
		return ""
	}
	return c.params[key]
}

// Query 获取查询参数。
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// Set 在context设置数据。
func (c *Context) Set(key string, value any) {
	c.keys[key] = value
}

// Get 获取context里设置的数据。
func (c *Context) Get(key string) (any, bool) {
	v, ok := c.keys[key]
	return v, ok
}

// MustGet 获取中间件设置的数据，不存在则 panic。
func (c *Context) MustGet(key string) any {
	v, ok := c.Get(key)
	if !ok {
		panic("key " + key + " not found in context")
	}
	return v
}

// BindJSON 解析 JSON 请求体到目标结构体。
func (c *Context) BindJSON(v any) error {
	defer c.Request.Body.Close()
	return json.NewDecoder(c.Request.Body).Decode(v)
}

// JSON 返回 JSON 响应。
func (c *Context) JSON(code int, data any) error {
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Writer.WriteHeader(code)
	return json.NewEncoder(c.Writer).Encode(data)
}

// String 返回纯文本响应。
func (c *Context) String(code int, body string) error {
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(code)
	_, err := c.Writer.Write([]byte(body))
	return err
}

// Status 设置 HTTP 状态码。
func (c *Context) Status(code int) {
	c.Writer.WriteHeader(code)
}

// Context 返回请求的 context.Context。
func (c *Context) Context() context.Context {
	return c.Request.Context()
}
