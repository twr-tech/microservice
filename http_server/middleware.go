package http_server

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// Middleware 中间件函数签名。
type Middleware func(HandlerFunc) HandlerFunc

// Use 将中间件包装为 HandlerFunc，便于在路由组中使用。
func Use(mw Middleware, handler HandlerFunc) HandlerFunc {
	return mw(handler)
}

// Chain 将多个中间件合并为一个。
func Chain(mws ...Middleware) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

// Logger 记录请求日志。
func Logger() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			start := time.Now()
			err := next(c)
			log.Printf("[%s] %s %s - %v", c.Request.Method, c.Request.URL.Path, time.Since(start), err)
			return err
		}
	}
}

// Recovery 捕获 panic 并返回 500。
func Recovery() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic recovered: %v\n%s", r, debug.Stack())
					_ = c.JSON(http.StatusInternalServerError, map[string]string{
						"error": "internal server error",
					})
				}
			}()
			return next(c)
		}
	}
}

// CORS 跨域中间件。
func CORS() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if c.Request.Method == http.MethodOptions {
				c.Status(http.StatusNoContent)
				return nil
			}
			return next(c)
		}
	}
}

// Auth 简单 Bearer Token 鉴权示例中间件。
func Auth(validToken string) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			token := c.Request.Header.Get("Authorization")
			if token != "Bearer "+validToken {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "unauthorized",
				})
			}
			c.Set("authenticated", true)
			return next(c)
		}
	}
}
