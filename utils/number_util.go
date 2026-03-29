package utils

import (
	"math"
	"strconv"
)

func HexToInt(s string) int {
	n, _ := strconv.ParseInt(s, 16, 32)
	return int(n)
}

func HexToInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 16, 64)
	return n
}

func Scale(val float64, precision int) float64 {
	factor := math.Pow(10, float64(precision))
	return math.Round(val*factor) / factor
}

// Ztd2Zhd 将 ZTD 转换为 ZHD（静态对流层延迟）
// pos[0]：纬度（单位：度）
// pos[2]：高度（单位：米）
// pres：气压（单位：hPa）
func Ztd2Zhd(pos []float64, pres float64) float64 {
	if len(pos) < 3 || pos[2] < -100.0 || pos[2] > 1e4 {
		return 0.0
	}
	hgt := math.Max(pos[2], 0.0)
	latRad := 2.0 * pos[0] * math.Pi / 180.0
	return 0.0022768 * pres / (1.0 - 0.00266*math.Cos(latRad) - 0.00028*hgt/1000.0)
}

// ConvertWindUV 风向角度转换（风来自方向变为风去的方向）
func ConvertWindUV(speed float64, dir int) (u, v float64) {
	if dir >= 180 {
		dir = dir - 180
	} else {
		dir = dir + 180
	}
	rad := float64(dir) * math.Pi / 180.0
	u = Scale(speed*math.Sin(rad), 6)
	v = Scale(speed*math.Cos(rad), 6)
	return
}

// CalculateDewPoint 计算露点温度（单位：摄氏度）
// 参数：temperatureCelsius：温度（°C），relativeHumidityPercent：相对湿度（0~100）
func CalculateDewPoint(temperatureCelsius, relativeHumidityPercent float64) float64 {
	a := 6.112
	b := 17.67
	lnRH := math.Log(relativeHumidityPercent / 100.0)
	gamma := lnRH + (a*temperatureCelsius)/(235+temperatureCelsius)
	value := (b * gamma) / (a - gamma)
	return Scale(value, 6)
}

// GetLevel determines the level based on the given value, returning an integer that represents the level.
func GetLevel(value float64) int {
	if value < 0.1/12 {
		return 0
	} else if value >= 5.0/12 && value <= 15.0/12 {
		return 1
	} else if value > 15.0/12 && value <= 30.0/12 {
		return 2
	} else if value > 30.0/12 && value <= 70.0/12 {
		return 3
	} else {
		return 4
	}
}
