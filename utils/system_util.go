package utils

import (
	"fmt"
	"net"
	"os/exec"
	"time"
)

// RestartService 重启指定的 systemd 服务
func RestartService(serviceName string) error {
	cmd := exec.Command("systemctl", "restart", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌重启服务失败: %v, 输出: %s", err, string(output))
		return fmt.Errorf("重启服务失败: %v, 输出: %s", err, string(output))
	}
	fmt.Printf("✅服务 [%s] 已成功重启\n", serviceName)
	return nil
}

// CheckLocalPort 检测本地端口是否可用
func CheckLocalPort(port int) bool {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// CheckPort 检测某个主机的端口是否可连接
func CheckPort(host string, port int) bool {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
