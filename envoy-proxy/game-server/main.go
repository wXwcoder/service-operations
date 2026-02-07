package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	consulapi "github.com/hashicorp/consul/api"
)

// ConsulRegistry Consul服务注册器
type ConsulRegistry struct {
	Client *consulapi.Client
}

// NewConsulRegistry 创建Consul注册器
func NewConsulRegistry(consulAddr string) (*ConsulRegistry, error) {
	config := consulapi.DefaultConfig()
	config.Address = consulAddr

	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建Consul客户端失败: %v", err)
	}

	return &ConsulRegistry{Client: client}, nil
}

// RegisterGameServer 注册游戏服务器到Consul
func (cr *ConsulRegistry) RegisterGameServer(serverID string, serverIP string, serverPort int, externalPort int) error {
	healthPort := serverPort + 1000

	registration := &consulapi.AgentServiceRegistration{
		ID:      serverID,
		Name:    "game-server", // 修改服务名为game-server
		Tags:    []string{"udp", "game"},
		Address: serverIP,
		Port:    serverPort,
		Meta: map[string]string{
			"protocol":            "udp",
			"server_type":         "game",
			"registered_at":       time.Now().Format("2006-01-02 15:04:05"),
			"envoy_external_port": fmt.Sprintf("%d", externalPort), // 为Envoy动态端口转发指定外部端口
		},
		Check: &consulapi.AgentServiceCheck{
			DeregisterCriticalServiceAfter: "5m",
			HTTP:                           fmt.Sprintf("http://%s:%d/health", serverIP, healthPort),
			Interval:                       "10s",
			Timeout:                        "2s",
		},
	}

	err := cr.Client.Agent().ServiceRegister(registration)
	if err != nil {
		return fmt.Errorf("注册服务到Consul失败: %v", err)
	}

	log.Printf("✅ 游戏服务器 %s 已成功注册到Consul: %s:%d (外部端口: %d)", serverID, serverIP, serverPort, externalPort)
	return nil
}

// DeregisterGameServer 从Consul注销游戏服务器
func (cr *ConsulRegistry) DeregisterGameServer(serverID string) error {
	err := cr.Client.Agent().ServiceDeregister(serverID)
	if err != nil {
		return fmt.Errorf("从Consul注销服务失败: %v", err)
	}

	log.Printf("✅ 游戏服务器 %s 已从Consul注销", serverID)
	return nil
}

// GameServer 游戏服务器结构体
type GameServer struct {
	ServerID     string
	ListenPort   int
	ExternalPort int // 对应的外部UDP端口
	Conn         *net.UDPConn
	Registry     *ConsulRegistry
}

// NewGameServer 创建新的游戏服务器实例
func NewGameServer(serverID string, port int, externalPort int, consulAddr string) (*GameServer, error) {
	registry, err := NewConsulRegistry(consulAddr)
	if err != nil {
		return nil, fmt.Errorf("创建Consul注册器失败: %v", err)
	}

	return &GameServer{
		ServerID:     serverID,
		ListenPort:   port,
		ExternalPort: externalPort,
		Registry:     registry,
	}, nil
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

// RegisterToConsul 注册到Consul
func (gs *GameServer) RegisterToConsul() error {
	if gs.Registry == nil {
		return fmt.Errorf("Consul注册器未初始化")
	}

	// 获取容器IP地址
	serverIP := os.Getenv("CONTAINER_IP")
	if serverIP == "" {
		serverIP = gs.ServerID // 使用服务名作为IP（在Docker网络中可用）
	}

	err := gs.Registry.RegisterGameServer(gs.ServerID, serverIP, gs.ListenPort, gs.ExternalPort)
	if err != nil {
		return fmt.Errorf("注册到Consul失败: %v", err)
	}

	return nil
}

// DeregisterFromConsul 从Consul注销
func (gs *GameServer) DeregisterFromConsul() error {
	if gs.Registry == nil {
		return fmt.Errorf("Consul注册器未初始化")
	}

	err := gs.Registry.DeregisterGameServer(gs.ServerID)
	if err != nil {
		return fmt.Errorf("从Consul注销失败: %v", err)
	}

	return nil
}

// registerWithRetry 带重试的 Consul 注册，应对 Consul 未就绪或重启
func registerWithRetry(gs *GameServer, totalWait time.Duration, interval time.Duration) {
	deadline := time.Now().Add(totalWait)
	for time.Now().Before(deadline) {
		if err := gs.RegisterToConsul(); err == nil {
			return
		}
		log.Printf("⚠️ 注册到Consul失败，%v 后重试...", interval)
		time.Sleep(interval)
	}
	log.Printf("⚠️ 游戏服务器将继续运行，但服务发现可能不可用（Consul 注册超时）")
}

// Stop 停止服务器
func (gs *GameServer) Stop() {
	// 从Consul注销
	if err := gs.DeregisterFromConsul(); err != nil {
		log.Printf("⚠️ 从Consul注销失败: %v", err)
	}

	// 关闭UDP连接
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

	// 从环境变量获取外部端口配置
	externalPortStr := os.Getenv("EXTERNAL_PORT")
	externalPort := port + 2000 // 默认外部端口为内部端口+2000
	if externalPortStr != "" {
		if ep, err := strconv.Atoi(externalPortStr); err == nil {
			externalPort = ep
		}
	}

	// Consul地址配置（支持 http://host:port，客户端会去掉 scheme）
	consulAddr := os.Getenv("CONSUL_URL")
	if consulAddr == "" {
		consulAddr = "consul-server:8500"
	}
	consulAddr = strings.TrimPrefix(strings.TrimPrefix(consulAddr, "http://"), "https://")

	// 创建并启动游戏服务器
	gameServer, err := NewGameServer(serverID, port, externalPort, consulAddr)
	if err != nil {
		log.Fatalf("创建游戏服务器失败: %v", err)
	}

	// 启动HTTP健康检查服务器
	go startHealthCheckServer(port + 1000)

	// 启动UDP服务器
	if err := gameServer.Start(); err != nil {
		log.Fatalf("启动游戏服务器失败: %v", err)
	}

	// 自动注册到Consul（带重试，应对 Consul 未就绪或重启）
	log.Printf("正在注册到Consul...")
	registerWithRetry(gameServer, 30*time.Second, 2*time.Second)

	log.Printf("🎮 游戏服务器 %s 准备就绪，监听端口: %d (外部端口: %d)", serverID, port, externalPort)

	// 设置信号处理，优雅关闭
	setupSignalHandling(gameServer)

	// 等待中断信号
	select {}
}

// setupSignalHandling 设置信号处理，实现优雅关闭
func setupSignalHandling(gameServer *GameServer) {
	// 创建信号通道
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动信号处理协程
	go func() {
		sig := <-sigChan
		log.Printf("收到信号: %v，正在优雅关闭...", sig)

		// 停止游戏服务器
		gameServer.Stop()

		log.Printf("游戏服务器已优雅关闭")
		os.Exit(0)
	}()
}
