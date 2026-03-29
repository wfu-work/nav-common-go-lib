package utils

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// PrintHex 格式化打印十六进制数据
func PrintHex(data []byte) {
	for _, b := range data {
		fmt.Printf(" %02X", b)
	}
	fmt.Println()
}

// HexStringToStrSlice 转成16进制字符串数组
func HexStringToStrSlice(hexStr string) []string {
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	var result []string
	for i := 0; i < len(hexStr); i += 2 {
		result = append(result, strings.ToUpper(hexStr[i:i+2]))
	}
	return result
}

// BytesToHexString 将字节切片转换为其十六进制字符串表示，每个字节之间用空格分隔
func BytesToHexString(data []byte, tag bool) string {
	var sb strings.Builder
	for i, b := range data {
		if tag && i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(fmt.Sprintf("%02X", b))
	}
	return sb.String()
}

// SwapHexByteOrder 高低位互换
func SwapHexByteOrder(hexStr string) (string, error) {
	if len(hexStr)%2 != 0 {
		return "", fmt.Errorf("hex string must have even length")
	}
	var result strings.Builder
	for i := len(hexStr); i > 0; i -= 2 {
		result.WriteString(hexStr[i-2 : i])
	}
	return result.String(), nil
}

// ReverseJoinHexStrings 高低位互换
func ReverseJoinHexStrings(arr []string) string {
	var sb strings.Builder
	for i := len(arr) - 1; i >= 0; i-- {
		sb.WriteString(arr[i])
	}
	return sb.String()
}

// ReverseHexStr 高低位互换
func ReverseHexStr(str string) string {
	if len(str)%2 != 0 {
		str = "0" + str
	}
	var groups []string
	for i := 0; i < len(str); i += 2 {
		groups = append(groups, str[i:i+2])
	}
	for i, j := 0, len(groups)-1; i < j; i, j = i+1, j-1 {
		groups[i], groups[j] = groups[j], groups[i]
	}
	return strings.Join(groups, "")
}

// HexStringsToBytes 16进制字符串转16进制字节
func HexStringsToBytes(hexStrs []string) ([]byte, error) {
	result := make([]byte, len(hexStrs))
	for i, s := range hexStrs {
		b, err := hex.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("hex decode error at index %d: %v", i, err)
		}
		result[i] = b[0]
	}
	return result, nil
}

// HexToFloat converts a 16-bit hex float string to float32 (IEEE 754)
func HexToFloat(hexStr string) float32 {
	val, err := strconv.ParseUint(hexStr, 16, 16)
	if err != nil {
		fmt.Println("invalid hex:", hexStr)
		return 0
	}
	intVal := uint16(val)
	exponent := uint32((intVal>>10)&0x1F) + (127 - 15)
	mantissa := uint32(intVal&0x3FF) << 13
	floatBits := (exponent << 23) | mantissa
	result := math.Float32frombits(floatBits)
	return float32(math.Round(float64(result)*1e4) / 1e4)
}

// CRC16Modbus 计算 CRC16 (Modbus 协议常用的实现)
func CRC16Modbus(data []byte) uint16 {
	var crc uint16 = 0xFFFF

	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if (crc & 0x0001) != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc = crc >> 1
			}
		}
	}
	return crc
}

// CalcNMEAChecksum 计算 NMEA 校验和
func CalcNMEAChecksum(sentence string) string {
	if sentence[0] == '$' {
		sentence = sentence[1:]
	}
	if i := strings.Index(sentence, "*"); i != -1 {
		sentence = sentence[:i]
	}
	var checksum byte
	for i := 0; i < len(sentence); i++ {
		checksum ^= sentence[i]
	}
	return fmt.Sprintf("%02X", checksum)
}
