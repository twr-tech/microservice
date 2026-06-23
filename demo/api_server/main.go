package main

import (
	"microservice/http_server"
	"net/http"

	"microservice"
)

func main() {
	app := microservice.New(microservice.DefaultConfig(":8080"))

	// 全局中间件
	app.Use(http_server.Recovery(), http_server.Logger(), http_server.CORS())

	// 公开路由
	app.Router.GET("/health", func(c *http_server.Context) error {
		return c.OK(map[string]string{"status": "ok"})
	})

	// 路由组 + 鉴权中间件
	api := app.Group("/api/v1", http_server.Auth("secret-token"))
	{
		api.GET("/users/:id", getUser)
		api.POST("/users", createUser)
	}

	// 嵌套路由组
	admin := api.Group("/admin")
	{
		admin.GET("/stats", func(c *http_server.Context) error {
			return c.OK(map[string]int{"users": 100, "orders": 500})
		})
	}

	if err := app.Run(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func getUser(c *http_server.Context) error {
	id := c.Param("id")
	return c.OK(map[string]string{
		"id":   id,
		"name": "Alice",
	})
}

type createUserReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func createUser(c *http_server.Context) error {
	var req createUserReq
	if err := c.BindJSON(&req); err != nil {
		return c.Error(http.StatusBadRequest, "invalid json body")
	}
	return c.OK(map[string]any{
		"id":    "u-001",
		"name":  req.Name,
		"email": req.Email,
	})
}
