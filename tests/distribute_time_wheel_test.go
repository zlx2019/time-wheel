// @Author Zero - 2024/3/15 20:18:38

package tests

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
	"timewheel"
	"timewheel/pkg/http"
	"timewheel/pkg/redis"
	"timewheel/pkg/utils"
)

const (
	host     = "127.0.0.1:6379"
	password = "root1234"
)

// 测试时间轮正常运行，并且执行定时任务
func TestDistributeTimeWheel(t *testing.T) {
	wheel := newDistributedTimeWheel()
	defer wheel.Shutdown()

	// 构建一个任务
	task := timewheel.NewHTTPTimeTask("testTask", "http://localhost:8080/timeTask", "GET", 1)
	// 添加定时任务
	log.Println("添加任务.")
	err := wheel.AddTask(context.Background(), task, time.Second*3)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 4)
	// 添加定时任务
	log.Println("添加任务.")
	err = wheel.AddDelayTask(context.Background(), task, time.Now().Add(time.Second*5))
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second * 20)
}

// 测试循环定时任务
func TestDistributeLoopTask(t *testing.T) {
	wheel := newDistributedTimeWheel()
	defer wheel.Shutdown()
	task := timewheel.NewHTTPTimeTask("testTask", "http://localhost:8080/timeTask", "GET", -1)
	log.Println("添加任务.")
	err := wheel.AddTask(context.Background(), task, time.Second*3)
	if err != nil {
		panic(err)
	}
	select {}
}

// 测试Redis是否可连接
func TestRedisConn(t *testing.T) {
	client := redis.NewClient(host, password)
	defer client.Close()
	conn, err := client.GetConnCtx(context.Background())
	if err != nil {
		panic(err)
	}
	reply, err := conn.Do("get", "k1")
	if err != nil {
		panic(err)
	}
	fmt.Println(reply)
}

// 测试向Redis中添加定时任务
func TestTaskAdd(t *testing.T) {
	wheel := newDistributedTimeWheel()
	task := timewheel.NewHTTPTimeTask("testTask", "http://127.0.0.1", "GET", 1)
	err := wheel.AddDelayTask(context.Background(), task, time.Now().Add(time.Second*5))
	if err != nil {
		panic(err)
	}
	task2 := timewheel.NewHTTPTimeTask("testTask2", "http://127.0.0.1", "POST", -1)
	err = wheel.AddTask(context.Background(), task2, time.Second*3)
	if err != nil {
		panic(err)
	}
}

// 测试从Redis中删除定时任务
func TestTaskRemove(t *testing.T) {
	wheel := newDistributedTimeWheel()
	execAt, _ := utils.TimeParse("2024-03-16 18:50:00")
	fmt.Println(execAt)
	err := wheel.RemoveTask(context.Background(), "testTask", execAt)
	if err != nil {
		panic(err)
	}
}

func newDistributedTimeWheel() *timewheel.DistributedTimeWheel {
	return timewheel.NewDistributedTimeWheel(newRedisClient(), http.NewClient())
}

func newRedisClient() *redis.Client {
	return redis.NewClient(host, password)
}

func TestName(t *testing.T) {
	t1 := time.Now().Add(time.Second * 3)
	fmt.Println(t1.Sub(time.Now()))
}
