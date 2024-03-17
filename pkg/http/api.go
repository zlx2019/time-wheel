// @Description HTTP 请求函数库
// @Author Zero - 2024/3/16 12:34:17

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type headers map[string]string
type query map[string]string
type body map[string]any

// Get 发起 HTTP-GET 请求
func (client *Client) Get(ctx context.Context, url string, reply any, query query, headers headers) error {
	return client.Request(ctx, getCompleteUrl(url, query), http.MethodGet, reply, nil, headers)
}

// Post 发起 HTTP POST 请求
func (client *Client) Post(ctx context.Context, url string, reply any, query query, body body, headers headers) error {
	return client.Request(ctx, getCompleteUrl(url, query), http.MethodPost, reply, body, headers)
}

// Request 发起 HTTP 请求
func (client *Client) Request(ctx context.Context, url, method string, reply any, requestBody body, headers headers) error {
	// 读取请求体
	var reader io.Reader
	if requestBody != nil {
		body, _ := json.Marshal(requestBody)
		reader = bytes.NewReader(body)
	}

	// 创建 请求
	request, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
	}
	// 设置请求头
	if request.Header == nil {
		request.Header = make(http.Header)
	}
	for k, v := range headers {
		request.Header.Add(k, v)
	}
	request.Header.Add("Content-Type","application/json")
	// 发起请求
	response, err := client.Do(request)

	// 校验请求是否成功(返回200)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status: %d", response.StatusCode)
	}
	defer response.Body.Close()
	if response == nil {
		return nil
	}

	// 读取响应数据
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	// 将响应数据写入到 reply 结构中
	return json.Unmarshal(responseBody, reply)
}

// 将 query 查询条件与url进行拼接处理
func getCompleteUrl(baseUrl string, params query) string{
	if len(params) == 0{
		return baseUrl
	}
	// 将 query 中的每一组条件，拼接成: key1=val1&key2=val2&...
	values := url.Values{}
	for k, v := range params {
		values.Add(k,v)
	}
	queryStr, _ := url.QueryUnescape(values.Encode())
	return fmt.Sprintf("%s?%s", baseUrl, queryStr)
}