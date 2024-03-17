// @Description 封装一些基础的Redis调用库
// @Author Zero - 2024/3/15 20:46:26

package redis

import (
	"context"
	"github.com/gomodule/redigo/redis"
)

// SAdd 指令: 向一个 ZSet 有序表中添加数据
func (client *Client) SAdd(ctx context.Context, key, val string) (int,error) {
	conn, err := client.GetConnCtx(ctx)
	if err != nil {
		return -1, err
	}
	defer conn.Close()
	return redis.Int(conn.Do("SADD", key, val))
}

// Eval 执行Lua脚本
// script: 要执行的脚步
// keys: key的数量
// params: 参数列表(由key-value组成的列表)
func (client *Client) Eval(ctx context.Context, script string, keys int, params []any) (any,error) {
	// 构建脚本参数
	args := make([]any, 2 + len(params))
	// 设置脚步内容和参数数量
	args[0] = script
	args[1] = keys
	// 设置参数
	copy(args[2:], params)
	// 从连接池中获取一个连接
	conn, err := client.pool.GetContext(ctx)
	if err != nil {
		return -1, err
	}
	defer conn.Close()
	// 执行 lua 脚步
	return conn.Do("EVAL", args...)
}

