// @Description 分布式时间轮
// @Author Zero - 2024/3/15 17:58:16

package timewheel

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/demdxx/gocast"
	"log"
	"strings"
	"sync"
	"time"
	"timewheel/pkg/http"
	"timewheel/pkg/redis"
	"timewheel/pkg/utils"
)

// DistributedTimeWheel 分布式时间轮结构
type DistributedTimeWheel struct {
	// 单例工具
	once sync.Once
	// Redis 客户端
	redisClient *redis.Client
	// HTTP 请求客户端，执行定时任务使用
	httpClient *http.Client
	// 定时器
	ticker *time.Ticker
	// 关闭时间轮信号通道
	stopCh chan struct{}
}

// NewDistributedTimeWheel 创建分布式时间轮
func NewDistributedTimeWheel(rClient *redis.Client, hClient *http.Client) *DistributedTimeWheel {
	// 创建分布式时间轮实例
	dtw := DistributedTimeWheel{
		redisClient: rClient,
		httpClient:  hClient,
		ticker:      time.NewTicker(time.Second),
		stopCh: make(chan struct{}),
	}

	// 开启异步协程,启动时间轮
	go dtw.run()
	return &dtw
}

// RemoveTask 将一个定时任务，从Redis中移除
// key: 要删除的任务的唯一标识
// execAt: 该任务的执行时间，用来确定删除时放入哪个Set集合
func (dtw *DistributedTimeWheel) RemoveTask(ctx context.Context, key string, execAt time.Time) error {
	_, err := dtw.redisClient.Eval(ctx, RemoveTaskScript, 1, []any{
		dtw.getTaskOfDelSetKey(execAt),
		key,
	})
	return err
}

// Shutdown 关闭时间轮
func (dtw *DistributedTimeWheel) Shutdown()  {
	dtw.once.Do(func() {
		close(dtw.stopCh)
		dtw.ticker.Stop()
		dtw.redisClient.Close()
	})
}

// AddDelayTask 添加延迟任务
func (dtw *DistributedTimeWheel) AddDelayTask(ctx context.Context, task *HTTPTimeTask, execAt time.Time) error {
	task.delay = execAt.Sub(time.Now())
	return dtw.addTask(ctx, task)
}

// AddTask 添加定时任务
func (dtw *DistributedTimeWheel) AddTask(ctx context.Context, task *HTTPTimeTask, delay time.Duration) error {
	task.delay = delay
	return dtw.addTask(ctx, task)
}

// 运行时间轮
func (dtw *DistributedTimeWheel) run() {
	for  {
		select {
		case <- dtw.stopCh:
			// 时间轮被关闭
			return
		case <-dtw.ticker.C:
			// 接收到定时器信号，调度执行定时任务
			dtw.dispatchReadyTasks()
		}
	}
}

// 定时器触发，开始调度处理定时任务
func (dtw *DistributedTimeWheel) dispatchReadyTasks()  {
	defer func() {
		if err:= recover(); err != nil{
			panic(err)
		}
	}()

	// 超时控制，保证30s内完成该批次的任务调度
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	// 根据当前时间的时间戳(秒)为条件，取出这一秒需要执行的定时任务列表
	tasks, err := dtw.getCurrentReadyTasks(ctx)
	if err != nil {
		// TODO LOG
		return
	}
	// 没有任务需要处理
	if len(tasks) == 0 {
		return
	}
	dtw.doBatchTask(ctx,tasks)
}

// 批量执行定时任务
func (dtw *DistributedTimeWheel) doBatchTask (ctx context.Context,tasks []*HTTPTimeTask) {
	// 并发执行定时任务
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		task := task
		go func() {
			defer func() {
				if err := recover();err != nil{
					// 捕获错误
					log.Printf("[ERROR] dispatch task [%s] error: %v",task.Key, err)
				}
				wg.Done()
			}()

			// 对定时任务发起HTTP请求
			if err := dtw.doTask(ctx, task); err != nil{
				panic(err)
			}
		}()
	}
	// 等待所有任务执行完成
	wg.Wait()
}

// 执行定时任务，发起HTTP请求
func (dtw *DistributedTimeWheel) doTask(ctx context.Context, task *HTTPTimeTask) error {
	return dtw.httpClient.Request(ctx, task.CallbackUrl, task.Method, nil, task.RequestBody, task.Headers)
}

// 获取当前就绪的定时任务
func (dtw *DistributedTimeWheel) getCurrentReadyTasks(ctx context.Context) ([]*HTTPTimeTask,error) {
	// 获取当前这一秒对应的定时任务列表的Key
	now := time.Now()
	setKey := dtw.getTaskOfZSetKey(now)
	delSetKey := dtw.getTaskOfDelSetKey(now)

	// 获取当前时间这一秒，至下一秒之间作为 score 的左右边界，进行条件检索
	// 将当前时间精确到秒
	nowSecond := utils.GetTimeSecond(now)
	// 当前秒的起始时间戳
	scoreStart := nowSecond.Unix()
	// 获取当前秒的截止时间戳(最后一毫秒)
	// 例如: 12:30:05.000 ~ 12:30:05.999
	scoreEnd := nowSecond.Add(time.Second - time.Millisecond).Unix()

	// 查询Redis，获取当前这一秒可以被执行的定时任务，以及是否有任务被删除的标识集合
	reply, err := dtw.redisClient.Eval(ctx, GetTaskScript, 2, []any{
		setKey, delSetKey,
		scoreStart, scoreEnd,
	})
	if err != nil {
		return nil, err
	}
	// 将返回结果转换为一个任意类型的切片
	replySlice := gocast.ToInterfaceSlice(reply)
	if len(replySlice) == 0 {
		return nil, fmt.Errorf("invalid reply: %v", replySlice)
	}
	// 获取已经取消的任务标识集合
	delTasks := gocast.ToStringSlice(replySlice[0])
	delTaskKeyMap := make(map[string]struct{}, len(delTasks))
	for _, taskKey := range delTasks {
		delTaskKeyMap[taskKey] = struct{}{}
	}

	// 获取要执行调度的定时任务
	tasks := make([]*HTTPTimeTask, 0, len(replySlice)-1)
	for i := 1; i < len(replySlice); i++ {
		var task HTTPTimeTask
		// 将定时任务Json字符串反序列化
		if err := json.Unmarshal([]byte(gocast.ToString(replySlice[i])), &task); err != nil {
			continue
		}
		// 该任务的唯一标识存在于 删除标识集合中，表示在此之前该任务已经被取消了，不需要执行
		if _, ok := delTaskKeyMap[task.Key]; ok {
			continue
		}
		tasks = append(tasks, &task)
	}
	return tasks,nil
}

// AddTask 向时间轮中添加任务
// task: 要添加的定时任务
// delay: 任务延迟时长
func (dtw *DistributedTimeWheel) addTask(ctx context.Context, task *HTTPTimeTask) error {
	// 校验定时任务基础参数
	if err := dtw.taskPreCheck(task);err != nil {
		return err
	}
	// 将定时任务序列化为 Json 格式, 作为存储类型
	taskJson, _ := json.Marshal(task)
	// 获取任务的执行时间
	execAt := time.Now().Add(task.delay)
	// 准备Lua脚本参数
	args := []any{
		dtw.getTaskOfZSetKey(execAt), // 设置 KEYS[1]，即任务要存储的ZSet的Key
		dtw.getTaskOfDelSetKey(execAt), // 设置 KEYS[2], 即该任务删除时，要标记的Set结构的Key
		execAt.Unix(), // 设置 ARGV[1], 即在ZSet中的score. 将任务执行时间转换为以秒为单位的时间戳，作为排序的score
		string(taskJson), // 设置 ARGV[2]，即具体要设置的Value
		task.Key,			 // 设置 ARGV[3]，该任务的唯一标识符
	}
	// 通过执行 Lua 脚本，将定时任务添加到 Redis中
	_, err := dtw.redisClient.Eval(ctx, ZAddTaskScript, 2, args)
	return err
}





// 定时任务检验
func (dtw *DistributedTimeWheel) taskPreCheck(task *HTTPTimeTask) error {
	switch task.Method {
	case "GET":
	case "POST":
	case "PUT":
	case "DELETE":
	default:
		return fmt.Errorf("invalid method: %s", task.Method)
	}
	if !strings.HasPrefix(task.CallbackUrl,"http://") && !strings.HasPrefix(task.CallbackUrl,"https://") {
		return fmt.Errorf("invalid url: %s", task.CallbackUrl)
	}
	return nil
}




// 根据一个任务的执行时间，获取它要存储的ZSet的Key，将时间格式化为分钟为单位作为ZSet的Key
func (dtw *DistributedTimeWheel) getTaskOfZSetKey(taskExecAt time.Time) string {
	// 将执行时间格式化为以分钟为单位的字符串，作为hash tag
	taskHashTag := utils.TimeFormatOfMinute(taskExecAt)
	return fmt.Sprintf("zero9501_timewheel_task:{%s}", taskHashTag)
}

// 根据一个任务的执行时间，获取标识它的删除集合Key
func (dtw *DistributedTimeWheel) getTaskOfDelSetKey(taskExecAt time.Time) string {
	taskHashTag := utils.TimeFormatOfMinute(taskExecAt)
	return fmt.Sprintf("zero9501_timewheel_del_task:{%s}", taskHashTag)
}