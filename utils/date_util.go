package utils

import (
	"fmt"
	"time"
)

// FormatTime 格式化时间字符串显示
func FormatTime(timestamp int64) string {
	t := time.UnixMilli(timestamp)
	timeStr := t.Format("2006-01-02 15:04:05")
	return timeStr
}

// GpsTimeFormat 格式化时间字符串显示
func GpsTimeFormat(timestamp int64) string {
	var utcTime = time.UnixMilli(timestamp).UTC()
	return utcTime.Format("2006/01/02 15:04:05")
}

// FormatTimeNo 格式化时间字符串显示
func FormatTimeNo(timestamp int64) string {
	t := time.UnixMilli(timestamp)
	timeStr := t.Format("20060102150405")
	return timeStr
}

// ParseGpsTime 将字符串日期转成时间
func ParseGpsTime(dateStr string) (time.Time, error) {
	return time.ParseInLocation("2006/01/02 15:04:05", dateStr, time.Local)
}

// ParseGpsTimeUnixMilli 将字符串日期转成时间戳
func ParseGpsTimeUnixMilli(dateStr string) int64 {
	t, err := ParseGpsTime(dateStr)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}

// GetEndWithMinute0or5 将给定时间向下取整到最近的5分钟刻度（0, 5, 10, 15, ...），并清零秒和毫秒
func GetEndWithMinute0or5(baseTime time.Time) time.Time {
	minute := baseTime.Minute()
	remainder := minute % 5
	truncatedTime := baseTime.Add(-time.Duration(remainder) * time.Minute)
	result := time.Date(
		truncatedTime.Year(),
		truncatedTime.Month(),
		truncatedTime.Day(),
		truncatedTime.Hour(),
		truncatedTime.Minute(),
		0,
		0,
		truncatedTime.Location(),
	)
	return result
}

func GetHourTime() int64 {
	now := time.Now()
	aligned := time.Date(
		now.Year(), now.Month(), now.Day(),
		now.Hour(), 0, 0, 0,
		now.Location(),
	)
	timestampMillis := aligned.UnixMilli()
	return timestampMillis
}

func LocalTimeUtilGetUtcTimeYearFileName(t time.Time) string {
	return t.UTC().Format("2006")
}

func LocalTimeUtilGetDayOfYear(t time.Time) string {
	day := t.UTC().YearDay()
	return fmt.Sprintf("%03d", day)
}

func LocalTimeUtilGetHourOfDay(t time.Time) string {
	return fmt.Sprintf("%02d", t.UTC().Hour())
}

func SecToReadableTimeInt(sec int) string {
	if sec < 60 {
		return fmt.Sprintf("%d秒", sec)
	} else if sec < 3600 {
		return fmt.Sprintf("%d分钟", sec/60)
	} else {
		return fmt.Sprintf("%d小时", sec/3600)
	}
}

// SecToCron 将秒转换成 cron 表达式（Linux标准格式）
func SecToCron(sec int) string {
	if sec < 60 {
		return fmt.Sprintf("*/%d * * * * *", sec)
	} else if sec < 3600 {
		return fmt.Sprintf("0 */%d * * * *", sec/60)
	} else {
		return fmt.Sprintf("0 0 */%d * * *", sec/3600)
	}
}

func SecToEvery(sec int) string {
	return fmt.Sprintf("@every %ds", sec)
}
