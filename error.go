// @Description 自定义错误类型，便于调试
// @Author Zero - 2024/3/14 20:06:58

package timewheel

import "fmt"

// taskRuntimeError 任务运行错误
type taskRuntimeError struct {
	// 出现错误的任务的标识
	key string
	// 错误信息
	message string
}
func (e *taskRuntimeError) Error() string {
	return fmt.Sprintf("key: %s of timetask error: %s \n", e.key,e.message)
}

func newTaskRuntimeError(key, message string) *taskRuntimeError {
	return &taskRuntimeError{key, message}
}