package http_server

import "net/http"

// Response 统一 API 响应结构。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// OK 返回成功响应。
func (c *Context) OK(data any) error {
	return c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

// Fail 返回业务失败响应。
func (c *Context) Fail(code int, message string) error {
	return c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// Error 返回 HTTP 错误响应。
func (c *Context) Error(status int, message string) error {
	return c.JSON(status, Response{
		Code:    status,
		Message: message,
	})
}
