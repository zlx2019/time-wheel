// @Description 基于Go实现时间轮
// @Author Zero - 2024/3/14 14:14:49

package timewheel

import (
	"container/list"
	"log"
	"sync"
	"time"
)

// TimeWheel 时间轮核心结构
type TimeWheel struct {
	// 单例工具
	once sync.Once
	// 定时器的调度间隔，也就是时间轮的轮询精度.
	interval time.Duration
	// 调度定时器，每隔 interval 时长调度一次.
	ticker *time.Ticker
	// 用于关闭时间轮
	stopCh chan struct{}
	// 是否正在运行
	isRun bool

	// 时间轮数组，每个元素就表示一个时间刻度，指向一个定时任务链表.
	slots []* list.List
	// 当前所在的时间刻度位置，也就是 slots 的索引位.
	curSlot int
	// 映射表，用于记录每个定时任务在哪个任务链表中的哪个节点上，主要是方便后续的删除.
	taskMapping map[string]*list.Element

	// 添加定时任务通道
	addTaskCh chan *timeTask
	// 删除定时任务通道
	removeTaskCh chan string
}

// NewTimeWheel 创建时间轮
// scale: 时间轮的时间刻度数量，也就是 slots 数组的长度.
// interval: 时间轮调度间隔的时长, slotNum * interval = 时间轮一轮所需的时长
func NewTimeWheel(scale int, interval time.Duration) *TimeWheel {
	// 刻度默认为30,将时间轮分为30个时间格
	if scale <= 0 {
		scale = 30
	}
	// 间隔时长默认为1s
	if interval <= 0 {
		interval = time.Second
	}

	// 初始化时间轮
	tw := TimeWheel{
		interval:     interval,
		ticker:       time.NewTicker(interval),
		isRun:        false,
		slots:        make([]*list.List, 0,scale),
		taskMapping:  make(map[string]*list.Element),
		stopCh:       make(chan struct{}),
		addTaskCh:    make(chan *timeTask),
		removeTaskCh: make(chan string),
	}

	// 初始化所有的定时任务链表
	for i := 0; i < scale; i++ {
		tw.slots = append(tw.slots, list.New())
	}

	// 开启异步协程,启动时间轮
	tw.isRun = true
	go tw.run()
	return &tw
}

// 时间轮运行，开始轮询调度
func (tw *TimeWheel) run() {
	defer func() {
		if err := recover(); err != nil {
			// 时间轮启动期间发生错误，直接panic
			panic(err)
		}
	}()
	for  {
		select {
		case <-tw.stopCh:
			// 时间轮停止
			tw.isRun = false
			return
		case <-tw.ticker.C:
			//  开始批量执行定时任务
			tw.dispatchReadyTasks()
		case task := <- tw.addTaskCh:
			//  注册定时任务
			tw.registerTask(task)
		case taskKey := <-tw.removeTaskCh:
			// 删除定时任务
			tw.RemoveTask(taskKey)
		}
	}
}


// Shutdown 关闭时间轮
func (tw *TimeWheel) Shutdown() {
	// 单例执行，确保 channel 只会被 close 一次.
	tw.once.Do(func() {
		// 关闭定时器
		tw.ticker.Stop()
		// 关闭时间轮工作协程
		close(tw.stopCh)
	})
}

// AddDelayTask 添加延迟任务
func (tw *TimeWheel) AddDelayTask(key string, task func() error, delay time.Time) {
	tw.AddTask(key,task, time.Until(delay), 1)
}


// AddLoopTask 添加定时任务
func (tw *TimeWheel) AddLoopTask(key string, task func() error, delay time.Duration) {
	tw.AddTask(key, task, delay, -1)
}

// AddTask 添加任务
func (tw *TimeWheel) AddTask(key string, task func() error, delay time.Duration, times int) {
	tw.addTask(key,task, delay, times)
}

// RemoveTask 删除定时任务
func (tw *TimeWheel) RemoveTask(key string)  {
	tw.cancelTask(key)
}

// 定时器触发，开始调度处理定时任务
func (tw *TimeWheel) dispatchReadyTasks() {
	// 去除当前刻度的任务链表
	tasks := tw.slots[tw.curSlot]
	// 移动刻度指针到下一个刻度
	defer tw.incrCurSlot()
	// 批量执行就绪的任务
	tw.batchDoTask(tasks)
}

// 批量执行任务，每次处理一个任务链表
// tasks: 要处理的定时任务链表
func (tw *TimeWheel) batchDoTask(tasks *list.List)  {
	// 从链表头结点开始遍历
	for elem := tasks.Front(); elem != nil; {
		// 获取任务
		task,_ := elem.Value.(*timeTask)
		// 判断任务是否是属于本轮次可执行
		if task.delayCycle > 0 {
			// 表示该任务还未到所属的 执行轮次，对存次进行递减即可
			task.delayCycle--
			elem = elem.Next()
			continue
		}

		// 开启协程执行对应的任务函数
		go func() {
			defer func() {
				if err := recover(); err != nil {
					if e, ok := err.(*taskRuntimeError); ok {
						// TODO 输出异常栈信息
						log.Printf("[ERROR] dispatch task [%s] error: %s",e.key, e.message)
					}
				}
			}()

			// 执行任务函数
			err := task.task()
			if err != nil {
				// 任务函数出现错误，抛出自定义错误
				panic(newTaskRuntimeError(task.key, err.Error()))
			}
			log.Printf("[LOG] dispatch task [%s] success\n", task.key)
			task.times--
			//  判断是否还有执行次数，或者是一个重复调度的定时任务，如果是就将任务继续注册到时间轮
			if task.times > 0 || task.times < -1{
				tw.addTask(task.key, task.task, task.delay, task.times)
			}
		}()
		next := elem.Next()
		// 从链表中删除该任务
		tasks.Remove(elem)
		// 从映射表中删除
		delete(tw.taskMapping, task.key)
		// 继续处理下一个节点
		elem = next
	}
}

// 时间轮指针移动到下一个刻度
func (tw *TimeWheel) incrCurSlot()  {
	// 到达时间轮尾部后，重新回到头部
	tw.curSlot = (tw.curSlot + 1) % len(tw.slots)
}

// 向时间轮中添加定时任务
// key: 任务的唯一标识
// task: 任务函数
// delay: 延迟的时长(相对时间)
// times: 执行的次数
func (tw *TimeWheel) addTask(key string, task func()error, delay time.Duration, times int) {
	// 延迟的时长 -1 毫秒，是为了算 pos 的位置更精准
	pos, delayCycle := tw.getTaskPosAndCycle(delay - time.Millisecond)
	tw.addTaskCh <- &timeTask{
		task:  task,
		pos:   pos,
		delayCycle: delayCycle,
		times: times,
		delay: delay,
		key:   key,
	}
}


// 根据定时任务的执行时间，计算出该放入哪个刻度的任务链表中，以及需要延迟的轮次
func (tw *TimeWheel) getTaskPosAndCycle(delayTime time.Duration) (int,int){
	// 将延迟的时长转为毫秒
	delay := int(delayTime)
	// 获取时间轮轮询一轮所需要的总时长
	totalInterval := len(tw.slots) * int(tw.interval)
	// 需要延迟的轮次 = 延迟时长 \ 总时长
	delayCycle := delay / totalInterval
	// 该放入的任务链表索引
	pos := (tw.curSlot + delay / int(tw.interval)) % len(tw.slots)
	return pos, delayCycle
}

// 将定时任务注册到对应的任务链表中
func (tw *TimeWheel) registerTask(task *timeTask) {
	// 获取从属的任务链表
	taskList := tw.slots[task.pos]
	// 如果定时任务的key已存在，则删除之前的任务
	if _, ok := tw.taskMapping[task.key]; ok{
		tw.RemoveTask(task.key)
	}
	// 将定时任务添加到链表尾部，并且获取存储的节点
	element := taskList.PushBack(task)
	// 建立映射关系
	tw.taskMapping[task.key] = element
}

// 取消定时任务
func (tw *TimeWheel) cancelTask(key string)  {
	// 从映射表中获取定时任务对应的链表节点
	element, ok := tw.taskMapping[key]
	if  !ok {
		return
	}
	// 强转为自定义的任务类型
	task,ok := element.Value.(*timeTask)
	if ok {
		tw.slots[task.pos].Remove(element)
	}
}