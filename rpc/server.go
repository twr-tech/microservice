package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"
)

// Handler RPC 方法处理函数。
type Handler func(ctx context.Context, params json.RawMessage) (any, error)

// Server RPC 服务端，注册并暴露远程方法。
type Server struct {
	mu       sync.RWMutex
	services map[string]map[string]Handler
}

// NewServer 创建 RPC 服务端。
func NewServer() *Server {
	return &Server{
		services: make(map[string]map[string]Handler),
	}
}

// Register 注册 RPC 方法。
func (s *Server) Register(service, method string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.services[service] == nil {
		s.services[service] = make(map[string]Handler)
	}
	s.services[service][method] = handler
	log.Printf("rpc registered: %s.%s", service, method)
}

// RegisterService 通过结构体注册 RPC 服务，方法签名为 func(ctx, *Req) (*Resp, error)。
func (s *Server) RegisterService(service string, receiver any) error {
	rv := reflect.ValueOf(receiver)
	rt := rv.Type()
	if rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("receiver must be a pointer to struct")
	}

	for i := 0; i < rt.NumMethod(); i++ {
		method := rt.Method(i)
		if !method.IsExported() {
			continue
		}
		mtype := method.Type
		if mtype.NumIn() != 3 || mtype.NumOut() != 2 {
			continue
		}
		if mtype.In(1) != reflect.TypeOf((*context.Context)(nil)).Elem() {
			continue
		}
		if mtype.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}

		methodName := method.Name
		handler := makeMethodHandler(rv, method)
		s.Register(service, methodName, handler)
	}
	return nil
}

func makeMethodHandler(receiver reflect.Value, method reflect.Method) Handler {
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		mtype := method.Type
		argType := mtype.In(2)
		arg := reflect.New(argType.Elem())

		if len(params) > 0 && string(params) != "null" {
			if err := json.Unmarshal(params, arg.Interface()); err != nil {
				return nil, fmt.Errorf("unmarshal params: %w", err)
			}
		}

		results := method.Func.Call([]reflect.Value{receiver, reflect.ValueOf(ctx), arg})
		if errVal := results[1].Interface(); errVal != nil {
			return nil, errVal.(error)
		}
		return results[0].Interface(), nil
	}
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeResponse(w, http.StatusMethodNotAllowed, Response{Code: 405, Message: "method not allowed"})
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, Response{Code: 400, Message: "invalid request body"})
		return
	}
	defer r.Body.Close()

	s.mu.RLock()
	methods, ok := s.services[req.Service]
	if !ok {
		s.mu.RUnlock()
		writeResponse(w, http.StatusOK, Response{Code: 404, Message: "service not found", ID: req.ID})
		return
	}
	handler, ok := methods[req.Method]
	s.mu.RUnlock()
	if !ok {
		writeResponse(w, http.StatusOK, Response{Code: 404, Message: "method not found", ID: req.ID})
		return
	}

	result, err := handler(r.Context(), req.Params)
	if err != nil {
		writeResponse(w, http.StatusOK, Response{Code: 500, Message: err.Error(), ID: req.ID})
		return
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		writeResponse(w, http.StatusOK, Response{Code: 500, Message: "marshal result failed", ID: req.ID})
		return
	}

	writeResponse(w, http.StatusOK, Response{Code: 0, Message: "ok", Result: resultData, ID: req.ID})
}

// RPCHandler 返回 RPC 入口 Handler，供框架路由挂载。
func (s *Server) RPCHandler() http.Handler {
	return http.HandlerFunc(s.handleRPC)
}

// HealthHandler 返回健康检查 Handler。
func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}

// Handler 返回独立部署用的 http.Handler，默认暴露 /rpc 与 /health。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.Mount(mux, "")
	return mux
}

// Mount 将 RPC 服务挂载到指定路径前缀下的 /rpc 与 /health。
func (s *Server) Mount(mux *http.ServeMux, prefix string) {
	mux.Handle(prefix+"/rpc", s.RPCHandler())
	mux.Handle(prefix+"/health", HealthHandler())
}

// ServeHTTP 实现 http.Handler，按路径分发 RPC 与健康检查（兼容独立部署）。
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/rpc":
		s.handleRPC(w, r)
	case "/health":
		HealthHandler().ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}
