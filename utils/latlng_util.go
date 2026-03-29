package utils

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// 将经纬度转换为度分秒格式
func latLng2Dfm(du float64) string {
	du1 := int(du)
	tp := (du - float64(du1)) * 60
	fen := int(tp)
	miao := math.Abs((tp - float64(fen)) * 60)
	return fmt.Sprintf("%d°%d'%.2f\"", du1, fen, miao)
}

// 度分秒转经纬度
func dfm2LatLng(dms string) float64 {
	if dms == "" {
		return 0.0
	}
	dms = strings.ReplaceAll(dms, " ", "")
	re := regexp.MustCompile(`(\d+)°(\d+)'([\d\.]+)"`)
	matches := re.FindStringSubmatch(dms)
	if len(matches) < 4 {
		return 0.0
	}

	d, _ := strconv.Atoi(matches[1])
	f, _ := strconv.Atoi(matches[2])
	m, _ := strconv.ParseFloat(matches[3], 64)

	fen := float64(f) + m/60
	du := math.Abs(float64(d)) + fen/60
	if d < 0 {
		du = -du
	}
	return math.Round(du*1e7) / 1e7 // 保留7位小数
}

// 将经纬度转换为度分格式
func latLng2Df(du float64) string {
	du1 := int(du)
	tp := (du - float64(du1)) * 60
	fen := int(tp)
	return fmt.Sprintf("%d°%d'", du1, fen)
}

// 度分转经纬度
func df2LatLng(dm string) float64 {
	if dm == "" {
		return 0.0
	}
	dm = strings.ReplaceAll(dm, " ", "")

	// 找到小数点前两位作为度
	idx := strings.Index(dm, ".")
	if idx < 3 {
		return 0.0
	}
	dStr := dm[:idx-2]
	d, err := strconv.Atoi(dStr)
	if err != nil {
		return 0.0
	}

	fen, err := strconv.ParseFloat(dm[len(dStr):], 64)
	if err != nil {
		return 0.0
	}

	lat := float64(d) + fen/60
	if lat < 0 {
		lat = -lat
	}
	return math.Round(lat*1e7) / 1e7
}

// DMS 转换为经纬度（特殊格式）
func dfm2Du(dms float64) float64 {
	degree := int(dms / 360000)
	minute := int((dms - float64(degree)*360000) / 6000)
	second := (dms - float64(degree)*360000 - float64(minute)*6000) / 100.0

	digitalDegree := float64(degree) + float64(minute)/60 + second/(60*60)
	return math.Round(digitalDegree*1e7) / 1e7
}

// 解析字符串为整数（空则返回0）
func parseInteger(str string) int {
	if str == "" {
		return 0
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return i
}

// GetN 计算N值
func GetN(lng float64) int {
	return int((lng + 1.5) / 3)
}

// GetCenterMeridian 获取中央子午线
func GetCenterMeridian(lng float64) int {
	return 3 * GetN(lng)
}

// Get3Degree 3度带编号
func Get3Degree(lng float64) int {
	return GetN(lng) - 25 + 4534
}

// GetCenterMeridianFromDegree 根据 degree 获取 Meridian
func GetCenterMeridianFromDegree(degree int) int {
	lng := float64(degree-4534+25)*3 - 1.5
	return 3 * GetN(lng)
}
