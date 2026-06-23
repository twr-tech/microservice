package microservice

import (
	"microservice/http_server"
	"microservice/rpc"
)

// Service 表示一个微服务模块，统一封装 HTTP 路由与 RPC 注册。
type Service struct {
	Name string
	app  *App
	rpc  *rpc.Server
}

// NewService 创建并绑定到 App 的微服务模块。
func (a *App) NewService(name string) *Service {
	return &Service{
		Name: name,
		app:  a,
		rpc:  rpc.NewServer(),
	}
}

// RPC 返回模块内的 RPC 服务端，用于手动注册方法。
func (s *Service) RPC() *rpc.Server {
	return s.rpc
}

// Register 通过结构体注册 RPC 方法，服务名默认为模块 Name。
func (s *Service) Register(receiver any) error {
	return s.rpc.RegisterService(s.Name, receiver)
}

// MountRPC 将模块 RPC 挂载到 HTTP 路由，默认入口 POST {prefix}。
func (s *Service) MountRPC(prefix string) {
	s.app.MountRPC(prefix, s.rpc)
}

// Group 创建该模块的 HTTP 路由组。
func (s *Service) Group(prefix string, mws ...http_server.Middleware) *http_server.Router {
	return s.app.Group(prefix, mws...)
}
