package microservice

import (
	"context"
	"errors"
	"fmt"
	"log"
	"microservice/http_server"
	"net/http"
	"time"
)

// Config 应用配置。
type Config struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultConfig 返回默认配置。
func DefaultConfig(addr string) Config {
	return Config{
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// App 微服务应用入口，整合路由与 HTTP 服务。
type App struct {
	Router *http_server.Router
	server *http.Server
	config Config
}

// New 创建应用实例。
func New(cfg Config) *App {
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	return &App{
		Router: http_server.NewRouter(),
		config: cfg,
	}
}

// Use 注册全局中间件。
func (a *App) Use(mws ...http_server.Middleware) {
	a.Router.Use(mws...)
}

// Group 创建路由组。
func (a *App) Group(prefix string, mws ...http_server.Middleware) *http_server.Router {
	return a.Router.Group(prefix, mws...)
}

// Run 启动 HTTP 服务（阻塞）。
func (a *App) Run() error {
	a.server = &http.Server{
		Addr:         a.config.Addr,
		Handler:      a.Router,
		ReadTimeout:  a.config.ReadTimeout,
		WriteTimeout: a.config.WriteTimeout,
		IdleTimeout:  a.config.IdleTimeout,
	}
	log.Printf("microframe server listening on %s", a.config.Addr)
	return a.server.ListenAndServe()
}

// Shutdown 优雅关闭服务。
func (a *App) Shutdown(ctx context.Context) error {
	if a.server == nil {
		return errors.New("server not started")
	}
	return a.server.Shutdown(ctx)
}

// Handler 返回 http.Handler，便于测试或嵌入其他服务。
func (a *App) Handler() http.Handler {
	return a.Router
}

// String 返回服务地址描述。
func (a *App) String() string {
	return fmt.Sprintf("microframe@%s", a.config.Addr)
}
