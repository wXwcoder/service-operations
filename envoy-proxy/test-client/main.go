package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// 游戏服务器映射表
var serverPorts = map[string]int{
	"1": 10000, // game-server-1
	"2": 10001, // game-server-2
	"3": 10002, // game-server-3
}

// UDPClient UDP客户端结构体
type UDPClient struct {
	ServerHost   string
	ServerPort   int
	TargetServer string
	Conn         *net.UDPConn
}

// NewUDPClient 创建新的UDP客户端
func NewUDPClient(host string, port int, targetServer string) *UDPClient {
	return &UDPClient{
		ServerHost:   host,
		ServerPort:   port,
		TargetServer: targetServer,
	}
}

// Connect 连接到UDP服务器
func (c *UDPClient) Connect() error {
	serverAddr := fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return fmt.Errorf("解析服务器地址失败: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("连接服务器失败: %v", err)
	}

	c.Conn = conn
	log.Printf("✅ 已连接到Envoy代理: %s", serverAddr)
	log.Printf("🎯 目标游戏服务器: game-server-%s", c.TargetServer)
	return nil
}

// SendMessage 发送消息到服务器
func (c *UDPClient) SendMessage(message string) (string, error) {
	if c.Conn == nil {
		return "", fmt.Errorf("未连接到服务器")
	}

	// 在消息中添加目标服务器信息
	enhancedMessage := fmt.Sprintf("[SERVER:%s] %s", c.TargetServer, message)

	// 发送消息
	_, err := c.Conn.Write([]byte(enhancedMessage))
	if err != nil {
		return "", fmt.Errorf("发送消息失败: %v", err)
	}

	// 设置读取超时
	c.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// 接收响应
	buffer := make([]byte, 1024)
	n, _, err := c.Conn.ReadFromUDP(buffer)
	if err != nil {
		return "", fmt.Errorf("接收响应失败: %v", err)
	}

	response := string(buffer[:n])
	return response, nil
}

// TestConnection 测试连接
func (c *UDPClient) TestConnection() error {
	response, err := c.SendMessage("PING")
	if err != nil {
		return err
	}

	log.Printf("📡 连接测试响应: %s", response)
	return nil
}

// TestBattleMessage 测试战斗消息
func (c *UDPClient) TestBattleMessage() error {
	response, err := c.SendMessage("BATTLE attack enemy-123")
	if err != nil {
		return err
	}

	log.Printf("⚔️ 战斗消息响应: %s", response)
	return nil
}

// TestStatusMessage 测试状态消息
func (c *UDPClient) TestStatusMessage() error {
	response, err := c.SendMessage("STATUS")
	if err != nil {
		return err
	}

	log.Printf("📊 状态消息响应: %s", response)
	return nil
}

// Close 关闭连接
func (c *UDPClient) Close() {
	if c.Conn != nil {
		c.Conn.Close()
		log.Printf("🔌 连接已关闭")
	}
}

func main() {
	// 从环境变量获取配置
	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "envoy-proxy"
	}

	// 显示服务器选择菜单
	fmt.Println("🚀 Envoy UDP代理测试客户端")
	fmt.Println("================================")
	fmt.Println("请选择要连接的游戏服务器:")
	fmt.Println("1. game-server-1 (端口: 10000)")
	fmt.Println("2. game-server-2 (端口: 10001)")
	fmt.Println("3. game-server-3 (端口: 10002)")
	fmt.Print("请输入选择 (1-3): ")

	var serverChoice string
	fmt.Scanln(&serverChoice)

	// 验证服务器选择
	port, exists := serverPorts[serverChoice]
	if !exists {
		log.Fatalf("❌ 无效的服务器选择: %s", serverChoice)
	}

	// 创建UDP客户端
	client := NewUDPClient(host, port, serverChoice)

	// 连接到服务器
	if err := client.Connect(); err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer client.Close()

	log.Printf("🚀 UDP客户端启动成功")
	log.Printf("📡 Envoy代理地址: %s:%d", host, port)
	log.Printf("🎯 目标游戏服务器: game-server-%s", serverChoice)
	log.Printf("💡 支持的命令: PING, BATTLE, STATUS, 或任意消息")
	log.Printf("⏹️  输入 'quit' 或 'exit' 退出\n")

	// 交互式测试循环
	for {
		fmt.Print("请输入消息: ")
		var input string
		fmt.Scanln(&input)

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if input == "quit" || input == "exit" {
			break
		}

		// 发送消息并接收响应
		response, err := client.SendMessage(input)
		if err != nil {
			log.Printf("❌ 发送消息失败: %v", err)
			continue
		}

		log.Printf("📨 服务器响应: %s", response)
	}

	log.Printf("👋 客户端退出")
}
