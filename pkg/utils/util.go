// @Description 工具库
// @Author Zero - 2024/3/16 15:04:54

package utils

import "time"

const TimeLayout = "2006-01-02 15:04:05"
const MinuteLayout = "2006-01-02:15:04"


// TimeFormatOfMinute 将日期时间，格式化为以分钟为单位的字符串
func TimeFormatOfMinute(date time.Time) string {
	return date.Format(MinuteLayout)
}

// TimeParse 时间格式化为字符串
func TimeParse(date string) (time.Time,error) {
	return time.ParseInLocation(TimeLayout, date, time.Local)
}

// TimeFormat 时间格式化
func TimeFormat(date time.Time) string {
	return date.Format(TimeLayout)
}

func GetTimeSecond(date time.Time) time.Time {
	return time.Date(date.Year(),date.Month(), date.Day(), date.Hour(), date.Minute(), date.Second(), 0, time.Local)
}