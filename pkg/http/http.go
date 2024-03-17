// @Author Zero - 2024/3/15 18:49:07

package http

import "net/http"

// Client 自定义HTTP客户端，用于调度定时任务
type Client struct {
	 *http.Client
}

// NewClient 创建HTTP客户端
func NewClient() *Client {
	return &Client{
		http.DefaultClient,
	}
}

