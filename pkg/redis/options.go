// @Author Zero - 2024/3/15 18:37:09

package redis

const (
	// 连接最大空闲超时时间
	defaultIdleTimeout = 10
	// 连接池最大活跃连接数
	defaultMaxActiveConn = 100
	// 连接池最大空闲连接数
	defaultMaxIdleConn = 20
)

// clientOptions Redis 客户端配置参数
type clientOptions struct {
	// Redis服务地址
	host string
	// 认证密码
	password string

	// 连接池最大空闲连接数
	maxIdleConn int
	// 连接池最大连接数
	maxActiveConn int
	// 空闲超时时间(秒)
	idleTimeout int
	// 是否为阻塞模式
	wait bool
}

type ClientOption func(opt *clientOptions)

func WithMaxIdleConn(maxIdleConn int) ClientOption {
	return func(opt *clientOptions) {
		opt.maxIdleConn = maxIdleConn
	}
}

func WithMaxActiveConn(maxActiveConn int) ClientOption {
	return func(opt *clientOptions) {
		opt.maxActiveConn = maxActiveConn
	}
}

func WithIdleTimeout(idleTimeout int) ClientOption {
	return func(opt *clientOptions) {
		opt.idleTimeout = idleTimeout
	}
}

func WithWaitMode() ClientOption {
	return func(opt *clientOptions) {
		opt.wait = true
	}
}

func initOptions(opt *clientOptions) {
	if opt.maxIdleConn < 0 {
		opt.maxIdleConn = defaultMaxIdleConn
	}
	if opt.maxActiveConn < 0 {
		opt.maxActiveConn = defaultMaxActiveConn
	}
	if opt.idleTimeout < 0 {
		opt.idleTimeout = defaultIdleTimeout
	}
}