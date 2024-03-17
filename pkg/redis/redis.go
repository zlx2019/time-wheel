// @Author Zero - 2024/3/15 18:35:11


package redis

import (
	"context"
	"github.com/gomodule/redigo/redis"
	"time"
)

// Client 自定义 Redis 客户端
type Client struct {
	// Redis 客户端配置
	*clientOptions
	// Redis 连接池
	pool *redis.Pool
}

// NewClient 创建 Redis 客户端
func NewClient(host, password string, opts ...ClientOption) *Client {
	// 初始化客户端配置
	options := &clientOptions{
		host:     host,
		password: password,
	}
	for _, opt := range opts {
		opt(options)
	}
	initOptions(options)

	// 初始化客户端和连接池
	client := &Client{
		options,
		nil,
	}
	client.initPool()
	return client
}


// GetConn 从连接池中获取一个连接
func (client *Client) GetConn() redis.Conn {
	return client.pool.Get()
}

// GetConnCtx 从连接池中获取一个连接，带有上下文控制
func (client *Client) GetConnCtx(ctx context.Context) (redis.Conn,error) {
	return client.pool.GetContext(ctx)
}

// Close 关闭连接池
func (client *Client) Close()error {
	return client.pool.Close()
}


// 初始化 Redis 客户端连接池
func (client *Client) initPool(){
	// 设置连接池配置
	client.pool = &redis.Pool{
		Dial:         client.dial,
		DialContext:  client.dialWithContext,
		TestOnBorrow: client.detectionConn,
		MaxIdle:      client.maxIdleConn,
		MaxActive:    client.maxActiveConn,
		IdleTimeout:  time.Duration(client.idleTimeout) * time.Second,
		Wait:         client.wait,
	}
}

// 连接池获取新的连接，与Redis建立新的连接，并且返回连接放入池中.
func (client *Client) dial() (redis.Conn, error) {
	return client.dialWithContext(context.Background())
}

// 与Redis建立连接，并且返回连接放入池中.(With Context)
func (client *Client) dialWithContext(ctx context.Context) (redis.Conn, error) {
	if client.host == "" {
		panic("Cannot get redis address from config")
	}
	var dialOpts []redis.DialOption
	if len(client.password) > 0 {
		// 设置连接密码
		dialOpts = append(dialOpts, redis.DialPassword(client.password))
	}
	// 连接Redis
	conn, err := redis.DialContext(ctx, "tcp", client.host, dialOpts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// 连接池调度空闲连接前执行此函数，探测连接是否还可用，返回 error 则该连接关闭
func (client *Client) detectionConn(conn redis.Conn, lastUsed time.Time) error {
	_, err := conn.Do("PING")
	return err
}
