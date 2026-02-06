package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// GameServer 游戏服务器结构体
type GameServer struct {
	ServerID   string
	ListenPort int
	Conn       *net.UDPConn
}

// NewGameServer 创建新的游戏服务器实例
func NewGameServer(serverID string, port int) *GameServer {
	return &GameServer{
		ServerID:   serverID,
		ListenPort: port,
	}
}

// Start 启动UDP服务器
func (gs *GameServer) Start() error {
	addr := fmt.Sprintf("0.0.0.0:%d", gs.ListenPort)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("解析UDP地址失败: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("监听UDP端口失败: %v", err)
	}

	gs.Conn = conn
	log.Printf("游戏服务器 %s 启动成功，监听端口: %d", gs.ServerID, gs.ListenPort)

	// 启动消息处理循环
	go gs.handleMessages()

	return nil
}

// handleMessages 处理接收到的UDP消息
func (gs *GameServer) handleMessages() {
	buffer := make([]byte, 1024)

	for {
		n, remoteAddr, err := gs.Conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("读取UDP数据失败: %v", err)
			continue
		}

		message := string(buffer[:n])
		log.Printf("服务器 %s 收到来自 %s 的消息: %s", gs.ServerID, remoteAddr.String(), message)

		// 处理不同类型的消息
		response := gs.processMessage(message, remoteAddr)

		// 发送响应
		if response != "" {
			_, err = gs.Conn.WriteToUDP([]byte(response), remoteAddr)
			if err != nil {
				log.Printf("发送响应失败: %v", err)
			}
		}
	}
}

// processMessage 处理不同类型的消息
func (gs *GameServer) processMessage(message string, remoteAddr *net.UDPAddr) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	if strings.HasPrefix(message, "PING") {
		return fmt.Sprintf("PONG from server %s at %s", gs.ServerID, timestamp)
	}

	if strings.HasPrefix(message, "BATTLE") {
		return fmt.Sprintf("BATTLE_RESPONSE from server %s: 战斗数据已处理 at %s", gs.ServerID, timestamp)
	}

	if strings.HasPrefix(message, "STATUS") {
		return fmt.Sprintf("STATUS_RESPONSE from server %s: 运行正常，端口 %d at %s",
			gs.ServerID, gs.ListenPort, timestamp)
	}

	return fmt.Sprintf("ECHO from server %s: %s at %s", gs.ServerID, message, timestamp)
}

// GetServerInfo 获取服务器信息
func (gs *GameServer) GetServerInfo() map[string]interface{} {
	return map[string]interface{}{
		"server_id":  gs.ServerID,
		"port":       gs.ListenPort,
		"protocol":   "udp",
		"status":     "running",
		"started_at": time.Now().Format(time.RFC3339),
	}
}

// startHealthCheckServer 启动HTTP健康检查服务器
func startHealthCheckServer(port int) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "healthy", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ready", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
	})

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("健康检查服务器启动，监听端口: %d", port)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Printf("健康检查服务器启动失败: %v", err)
	}
}

// Stop 停止服务器
func (gs *GameServer) Stop() {
	if gs.Conn != nil {
		gs.Conn.Close()
		log.Printf("游戏服务器 %s 已停止", gs.ServerID)
	}
}

func main() {
	// 从环境变量获取服务器配置
	serverID := os.Getenv("SERVER_ID")
	if serverID == "" {
		serverID = "game-server-1"
	}

	portStr := os.Getenv("SERVER_PORT")
	port := 8080
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	// 创建并启动游戏服务器
	gameServer := NewGameServer(serverID, port)

	if err := gameServer.Start(); err != nil {
		log.Fatalf("启动游戏服务器失败: %v", err)
	}

	// 启动HTTP健康检查服务器
	go startHealthCheckServer(port + 1000)

	// 注册到Consul（在实际部署中实现）
	log.Printf("游戏服务器 %s 准备就绪，等待连接...", serverID)

	// 等待中断信号
	select {}
}
