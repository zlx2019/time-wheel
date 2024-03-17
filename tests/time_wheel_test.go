// @Title time_wheel_test.go
// @Description $END$
// @Author Zero - 2024/3/14 20:58:07

package tests

import (
	"errors"
	"fmt"
	"log"
	"testing"
	"time"
	"timewheel"
)

// 测试延迟任务
func TestDelayTask(t *testing.T) {
	wheel := newTimeWheel()
	defer wheel.Shutdown()

	task := func()error {
		fmt.Println("Hello World!")
		return nil
	}
	// 添加一个延迟 3 秒的任务函数
	wheel.AddDelayTask("test1",task, time.Now().Add(time.Second * 3))
	time.Sleep(time.Second * 5)
}

// 测试重复定时任务
func TestLoopTask(t *testing.T) {
	wheel := newTimeWheel()
	defer wheel.Shutdown()

	task := func()error {
		fmt.Println("Hello World!")
		return nil
	}
	// 注册一个定时任务函数，每隔 3 秒执行一次.
	wheel.AddLoopTask("test2", task, time.Second * 3)
	select {

	}
}

// 测试执行指定次数的定时任务
func TestTask(t *testing.T) {
	wheel := newTimeWheel()
	defer wheel.Shutdown()

	task := func()error {
		fmt.Println("Hello World!")
		return nil
	}
	wheel.AddTask("test3", task, time.Second * 3, 3)
	time.Sleep(time.Second * 10)
}

// 测试多个定时任务执行
func TestMultipleTask(t *testing.T) {
	wheel := newTimeWheel()
	defer wheel.Shutdown()

	wheel.AddTask("task1", func() error {
		fmt.Println("我是task1")
		return nil
	}, time.Second * 1, 10)

	wheel.AddTask("task2", func() error {
		fmt.Println("我是task2")
		return nil
	}, time.Second * 2, 5)

	wheel.AddTask("task3", func() error {
		fmt.Println("我是task3")
		return nil
	}, time.Second * 5, 2)

	time.Sleep(time.Second * 11)
}

// 测试定时任务出现错误
func TestTaskError(t *testing.T) {
	wheel := newTimeWheel()
	defer wheel.Shutdown()

	wheel.AddDelayTask("task", func() error {
		return errors.New("一个不简单的错误")
	}, time.Now().Add(time.Second * 3))

	time.Sleep(time.Second * 4)
}

// 测试时间轮-以秒为调度单元，注册分钟级别定时任务
func TestBigTimeTask(t *testing.T) {
	wheel := newTimeWheel()
	defer wheel.Shutdown()
	log.Println("start time.")
	wheel.AddTask("task1", func() error {
		fmt.Println("我开始执行了哦~")
		return nil
	}, time.Minute, 3)
	time.Sleep(time.Second * 181)
}

// 测试 定时任务的覆盖
func TestTaskCover(t *testing.T) {
	wheel := newTimeWheel()
	defer wheel.Shutdown()

	log.Println("原任务注册.")
	wheel.AddTask("mytask", func() error {
		fmt.Println("我是原任务~")
		return nil
	}, time.Second * 10, 1)

	time.Sleep(time.Second)

	log.Println("新任务注册.")
	wheel.AddTask("mytask", func() error {
		fmt.Println("我是新任务~")
		return nil
	}, time.Second * 3, 1)

	time.Sleep(time.Second * 20)
}

func newTimeWheel() *timewheel.TimeWheel {
	return timewheel.NewTimeWheel(60, time.Second)
}