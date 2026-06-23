package microservice

import (
	"microservice/http_server"
	"net/http"
	"strings"

	"microservice/rpc"
)

// NewRPCServer 创建 RPC 服务端并挂载到 App，返回服务端实例便于注册方法。
func (a *App) NewRPCServer(prefix string) *rpc.Server {
	server := rpc.NewServer()
	a.MountRPC(prefix, server)
	return server
}

// MountRPC 将 RPC 服务端挂载到 HTTP 路由。
// prefix 为 RPC 入口路径，如 "/rpc" 对应 POST /rpc。
func (a *App) MountRPC(prefix string, server *rpc.Server) {
	prefix = normalizePath(prefix)
	a.Router.POST(prefix, adaptHandler(server.RPCHandler()))
	a.Router.GET(prefix+"/health", func(c *http_server.Context) error {
		return c.OK(map[string]string{"status": "ok", "service": "rpc"})
	})
}

// MountRPCStandalone 挂载完整 RPC 服务（含 /rpc 与 /health 子路径），适合独立 RPC 进程接入网关。
func (a *App) MountRPCStandalone(prefix string, server *rpc.Server) {
	prefix = normalizePath(prefix)
	a.Router.POST(prefix+"/rpc", adaptHandler(server.RPCHandler()))
	a.Router.GET(prefix+"/health", adaptHandler(rpc.HealthHandler()))
}

func adaptHandler(h http.Handler) http_server.HandlerFunc {
	return func(c *http_server.Context) error {
		h.ServeHTTP(c.Writer, c.Request)
		return nil
	}
}

func normalizePath(path string) string {
	if path == "" {
		return "/rpc"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/")
}
