// @Description 定时任务结构
// @Author Zero - 2024/3/14 16:17:06

package timewheel

import "time"

// timeTask 定时任务，注册到时间轮上的每一个任务节点
type timeTask struct {
	// 定时任务的唯一标识
	key string
	// 闭包函数，任务的具体执行逻辑
	task func() error
	// 该任务在时间轮 slots 环形数组中的索引位置
	pos int
	// 定时任务的延迟轮次，指的是时间轮整体轮询几次才可以满足执行条件
	// 延迟时长越久，该数值可能会越大，只有取值为0 时才表示任务延迟时间已到.
	delayCycle int
	// 执行次数
	// 	 -1: 表示为定时任务，不停的重复执行.
	// 	  1: 表示为延迟任务，只执行一次.
	//	  n: 表示该任务共执行n次，执行完后移除该任务.
	times int
	// 任务的延迟时长
	delay time.Duration
}

// HTTPTimeTask 分布式定时任务
// 分布式定时任务以 HTTP 接口形式存储，时间到达时调度该接口即可.
type HTTPTimeTask struct {
	// 唯一标识
	Key string `json:"key"`
	// 任务回调地址
	CallbackUrl string `json:"callback_url"`
	// 请求类型
	Method string `json:"method"`
	// 请求 query 参数
	Query map[string]string `json:"query"`
	// 请求体数据
	RequestBody map[string]any `json:"request_body"`
	// 请求头数据
	Headers map[string]string `json:"headers"`

	// 执行次数
	Times int `json:"times"`
	// 每次执行的延迟时间
	delay time.Duration `json:"delay"`
}

func NewHTTPTimeTask(key, callback, method string, times int) *HTTPTimeTask {
	return &HTTPTimeTask{
		Key:         key,
		CallbackUrl: callback,
		Method:      method,
		Times:       times,
		Query:       nil,
		RequestBody: nil,
		Headers:     nil,
	}
}
