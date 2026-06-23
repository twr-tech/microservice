package rpc

import (
	"encoding/json"
	"net/http"
)

// Request RPC 请求体。
type Request struct {
	Service string          `json:"service"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id,omitempty"`
}

// Response RPC 响应体。
type Response struct {
	Code    int             `json:"code"`
	Message string          `json:"message,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	ID      any             `json:"id,omitempty"`
}

func writeResponse(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
