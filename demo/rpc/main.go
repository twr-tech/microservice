package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"microservice/rpc"
)

// UserService 示例 RPC 服务。
type UserService struct{}

type GetUserReq struct {
	ID string `json:"id"`
}

type GetUserResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *UserService) GetUser(ctx context.Context, req *GetUserReq) (*GetUserResp, error) {
	return &GetUserResp{
		ID:   req.ID,
		Name: "test2",
	}, nil
}

type AddReq struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddResp struct {
	Sum int `json:"sum"`
}

func main() {
	// ---- RPC 服务端 ----
	rpcServer := rpc.NewServer()
	if err := rpcServer.RegisterService("UserService", &UserService{}); err != nil {
		log.Fatal(err)
	}
	rpcServer.Register("MathService", "Add", func(ctx context.Context, params json.RawMessage) (any, error) {
		var req AddReq
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, err
		}
		return &AddResp{Sum: req.A + req.B}, nil
	})

	go func() {
		log.Println("RPC server listening on :9090")
		if err := http.ListenAndServe(":9090", rpcServer); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(200 * time.Millisecond)

	// ---- RPC 客户端 ----
	client := rpc.NewClient(rpc.ClientConfig{
		Endpoint: "http://127.0.0.1:9090/rpc",
		Timeout:  3 * time.Second,
	})

	ctx := context.Background()

	var user GetUserResp
	if err := client.Call(ctx, "UserService", "GetUser", &GetUserReq{ID: "42"}, &user); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("GetUser: %+v\n", user)

	var sum AddResp
	if err := client.Call(ctx, "MathService", "Add", &AddReq{A: 3, B: 5}, &sum); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Add: %+v\n", sum)
}
