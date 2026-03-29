package utils

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type IpApiResp struct {
	Country string `json:"country"`
	Region  string `json:"regionName"`
	City    string `json:"city"`
	Query   string `json:"query"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func IsMobileDevice(ua string) bool {
	mobileKeywords := []string{
		"Mobile", "iPhone", "Android", "iPad", "iPod",
		"Windows Phone", "BlackBerry", "Opera Mini",
	}
	for _, keyword := range mobileKeywords {
		if strings.Contains(strings.ToLower(ua), strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func GetDeviceType(c *gin.Context) string {
	ua := c.Request.UserAgent()
	if IsMobileDevice(ua) {
		return "mobile"
	}
	return "pc"
}

func IsPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	// 判断是否为 loopback、multicast、link-local 或 private
	if ip.IsLoopback() || ip.IsMulticast() || ip.IsLinkLocalUnicast() {
		return true
	}
	// 判断是否属于私有网络范围（RFC 1918）
	privateNets := []*net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},     // 10.0.0.0/8
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},  // 172.16.0.0/12
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)}, // 192.168.0.0/16
		{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)}, // 169.254.0.0/16 (link-local)
	}
	for _, privNet := range privateNets {
		if privNet.Contains(ip) {
			return true
		}
	}
	if ip.To4() == nil && ip.IsPrivate() {
		return true
	}
	return false
}

func GetIpLocation(ip string) (string, error) {
	if IsPrivateIP(ip) {
		return "内网", nil
	}
	resp, err := http.Get("http://ip-api.com/json/" + ip + "?lang=zh-CN")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result IpApiResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Status != "success" {
		return "", fmt.Errorf("query failed: %s", result.Message)
	}
	return result.Country + "|" + result.Region + "|" + result.City, nil
}
