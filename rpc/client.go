package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ClientConfig RPC 客户端配置。
type ClientConfig struct {
	// BaseURL 服务根地址，如 http://127.0.0.1:9090；会自动拼接 /rpc。
	BaseURL string
	// Endpoint 是 RPC 完整地址；若设置则优先于 BaseURL。
	Endpoint       string
	Timeout        time.Duration
	HTTPClient     *http.Client
	DefaultHeaders map[string]string
}

// Client RPC 客户端，基于 net/http 调用远程服务。
type Client struct {
	endpoint string
	client   *http.Client
	headers  map[string]string
}

// NewClient 创建 RPC 客户端。
func NewClient(cfg ClientConfig) *Client {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		base := strings.TrimRight(cfg.BaseURL, "/")
		endpoint = base + "/rpc"
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	return &Client{
		endpoint: endpoint,
		client:   httpClient,
		headers:  cfg.DefaultHeaders,
	}
}

// Call 调用远程 RPC 方法，将结果解码到 result。
func (c *Client) Call(ctx context.Context, service, method string, params, result any) error {
	var rawParams json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params: %w", err)
		}
		rawParams = data
	}

	reqBody := Request{
		Service: service,
		Method:  method,
		Params:  rawParams,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rpc http error: status=%d body=%s", resp.StatusCode, string(respData))
	}

	var rpcResp Response
	if err := json.Unmarshal(respData, &rpcResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	if rpcResp.Code != 0 {
		return fmt.Errorf("rpc error: code=%d message=%s", rpcResp.Code, rpcResp.Message)
	}

	if result != nil && len(rpcResp.Result) > 0 && string(rpcResp.Result) != "null" {
		if err := json.Unmarshal(rpcResp.Result, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}
	return nil
}
