package ip2geo

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/service"

	"github.com/wfu-work/nav-common-go-lib/global"
)

//go:embed ip2region_v4.xdb
var ipV4DataFile embed.FS

//go:embed ip2region_v6.xdb
var ipV6DataFile embed.FS

var (
	ipSearcher *service.Ip2Region
	searcherMu sync.RWMutex
)

// ExtractIPRegionDB 从嵌入的文件系统中提取 IP2Region 数据库
func ExtractIPRegionDB() (string, string, error) {
	if _, err := os.Stat(global.Config().Local.Ip2geoPath); os.IsNotExist(err) {
		if err := os.MkdirAll(global.Config().Local.Ip2geoPath, 0755); err != nil {
			return "", "", err
		}
	}
	dbv4Path := filepath.Join(global.Config().Local.Ip2geoPath, "ip2region_v4.xdb")
	dbv6Path := filepath.Join(global.Config().Local.Ip2geoPath, "ip2region_v6.xdb")
	if _, err := os.Stat(dbv4Path); err == nil {
		if _, err := os.Stat(dbv6Path); err == nil {
			log.Println("IP2Region 数据库已存在，跳过提取")
			return dbv4Path, dbv6Path, nil
		}
	}
	data, err := fs.ReadFile(ipV4DataFile, "ip2region_v4.xdb")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(dbv4Path, data, 0644); err != nil {
		return "", "", err
	}
	data, err = fs.ReadFile(ipV6DataFile, "ip2region_v6.xdb")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(dbv6Path, data, 0644); err != nil {
		return "", "", err
	}
	log.Println("IP2Region 数据库已成功提取")
	return dbv4Path, dbv6Path, nil
}

// InitIpGeoLocation 初始化 IP 地理位置查询
func InitIpGeoLocation() error {
	ipV4Path, ipV6Path, err := ExtractIPRegionDB()
	if err != nil {
		return fmt.Errorf("提取 ip2region 数据库失败: %v", err)
	}
	v4Config, err := service.NewV4Config(service.VIndexCache, ipV4Path, 20)
	if err != nil {
		return fmt.Errorf("failed to create v4 config: %s", err)
	}
	v6Config, err := service.NewV6Config(service.VIndexCache, ipV6Path, 20)
	if err != nil {
		return fmt.Errorf("failed to create v6 config: %s", err)
	}
	ip2region, err := service.NewIp2Region(v4Config, v6Config)
	if err != nil {
		return fmt.Errorf("failed to create ip2region service: %s", err)
	}
	searcherMu.Lock()
	if ipSearcher != nil {
		ipSearcher.Close()
	}
	ipSearcher = ip2region
	searcherMu.Unlock()
	log.Println("ip2region 初始化成功")
	return nil
}

// CloseIpGeoLocation 关闭 IP 地理位置查询
func CloseIpGeoLocation() {
	searcherMu.Lock()
	defer searcherMu.Unlock()
	if ipSearcher != nil {
		ipSearcher.Close()
		ipSearcher = nil
	}
}

// GetIpLocation 获取 IP 的地理位置信息
func GetIpLocation(ip string) (string, string, string, error) {
	if ip == "" || ip == "localhost" || ip == "127.0.0.1" {
		return "本地", "本地", "本地", nil
	}
	if isPrivateIP(net.ParseIP(ip)) {
		return "内网", "本地网络", "本地网络", nil
	}
	domestic, provinces, glo, err := queryIpLocation(ip)
	if err != nil {
		return "未知", "未知", "未知", err
	}
	return domestic, provinces, glo, nil
}

// 查询 IP 地理位置
func queryIpLocation(ip string) (string, string, string, error) {
	searcherMu.RLock()
	defer searcherMu.RUnlock()
	if ipSearcher == nil {
		return "未知", "未知", "未知", fmt.Errorf("ip2region 未初始化")
	}
	region, err := ipSearcher.Search(ip)
	if err != nil {
		return "未知", "未知", "未知", err
	}
	return parseIpRegion(region)
}

// 解析 ip2region 返回的地区信息
func parseIpRegion(region string) (string, string, string, error) {
	parts := splitRegion(region)
	var domestic, provinces, glo string
	// 国内
	if parts[0] == "中国" {
		if parts[1] != "" && parts[1] != "0" {
			provinces = removeSuffixes(parts[1])
		}
		if parts[2] != "" && parts[2] != "0" {
			domestic = removeSuffixes(parts[2])
		} else if parts[3] != "" && parts[3] != "0" {
			domestic = parts[3]
		} else {
			domestic = "中国"
		}
	} else if parts[0] == "0" || parts[0] == "" {
		domestic = "未知"
	} else {
		domestic = "国外"
	}
	// 全球
	if parts[0] != "0" && parts[0] != "" {
		glo = parts[0]
	} else {
		glo = "未知"
	}

	return domestic, provinces, glo, nil
}

// 解析 ip2region
func splitRegion(region string) []string {
	parts := make([]string, 5)
	fields := bytes.Split([]byte(region), []byte("|"))
	for i := 0; i < len(fields) && i < 5; i++ {
		parts[i] = string(fields[i])
	}
	return parts
}

// 是否是内网 IP
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	privateIPRanges := []struct {
		start net.IP
		end   net.IP
	}{
		{net.ParseIP("10.0.0.0"), net.ParseIP("10.255.255.255")},
		{net.ParseIP("172.16.0.0"), net.ParseIP("172.31.255.255")},
		{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.255.255")},
	}
	for _, r := range privateIPRanges {
		if bytes.Compare(ip, r.start) >= 0 && bytes.Compare(ip, r.end) <= 0 {
			return true
		}
	}
	return false
}

func removeSuffixes(name string) string {
	suffixes := []string{"省", "自治区", "维吾尔自治区", "壮族自治区", "回族自治区", "特别行政区"}
	for _, suffix := range suffixes {
		if len(name) > len(suffix) && name[len(name)-len(suffix):] == suffix {
			return name[:len(name)-len(suffix)]
		}
	}
	return name
}
