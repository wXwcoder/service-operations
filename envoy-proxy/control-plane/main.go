package main

import (
	"context"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	udpproxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/udp/udp_proxy/v3"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

const (
	grpcKeepaliveTime        = 30 * time.Second
	grpcKeepaliveTimeout     = 5 * time.Second
	grpcKeepaliveMinTime     = 30 * time.Second
	grpcMaxConcurrentStreams = 1000000
)

// ControlPlane 控制平面结构体
type ControlPlane struct {
	cache   cache.SnapshotCache
	server  server.Server
	consul  *consulapi.Client
	ctx     context.Context
	cancel  context.CancelFunc
	xdsPort uint
}

// NewControlPlane 创建新的控制平面实例
func NewControlPlane(consulAddr string, xdsPort uint) (*ControlPlane, error) {
	// 初始化Consul客户端
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = consulAddr
	consulClient, err := consulapi.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("创建Consul客户端失败: %v", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建缓存 - 完全禁用一致性检查
	// UDP代理不需要标准的HTTP路由配置，因此禁用一致性检查
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)

	// 创建服务器
	callbacks := &test.Callbacks{}
	xdsserver := server.NewServer(ctx, snapshotCache, callbacks)

	controlPlane := &ControlPlane{
		cache:   snapshotCache,
		server:  xdsserver,
		consul:  consulClient,
		ctx:     ctx,
		cancel:  cancel,
		xdsPort: xdsPort,
	}

	return controlPlane, nil
}

// Start 启动控制平面
func (cp *ControlPlane) Start() error {
	// 启动Consul监听器
	go cp.watchConsulServices()

	// 启动xDS服务器
	cp.runXdsServer()

	return nil
}

// watchConsulServices 监听Consul服务变化
func (cp *ControlPlane) watchConsulServices() {
	log.Println("🔄 开始监听Consul服务变化...")

	// 初始加载服务
	cp.updateEnvoyConfig()

	// 定期轮询Consul服务变化
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cp.ctx.Done():
			log.Println("⏹️ 控制平面停止监听")
			return
		case <-ticker.C:
			cp.updateEnvoyConfig()
		}
	}
}

// updateEnvoyConfig 更新Envoy配置
func (cp *ControlPlane) updateEnvoyConfig() {
	log.Println("🔄 更新Envoy配置...")

	// 查询所有game-server服务
	services, _, err := cp.consul.Health().Service("game-server", "", true, nil)
	if err != nil {
		log.Printf("❌ 查询Consul服务失败: %v", err)
		return
	}

	log.Printf("📊 发现 %d 个game-server服务实例", len(services))

	// 构建新的快照
	snapshot, err := cp.buildSnapshot(services)
	if err != nil {
		log.Printf("❌ 构建快照失败: %v", err)
		return
	}

	// Envoy 拉取配置时使用的 node.id 必须与 SetSnapshot 的 node 一致。go-control-plane 用 request.Node 的 hash 作为 key。
	// 为 bootstrap 中的 id (envoy_instance_01) 与 docker-compose --service-node (proxy-1) 都设置快照，避免不一致导致 listeners 为空
	nodeIDs := []string{"proxy-1"}
	if custom := os.Getenv("ENVOY_NODE_ID"); custom != "" {
		nodeIDs = []string{custom}
	}
	// #region agent log
	// debugLogPath := os.Getenv("DEBUG_LOG_PATH")
	// if debugLogPath == "" {
	// 	debugLogPath = "e:\\xcode\\service-operations\\envoy-proxy\\.cursor\\debug.log"
	// }
	// if f, e := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); e == nil {
	// 	b, _ := json.Marshal(map[string]interface{}{"sessionId": "debug-session", "hypothesisId": "H2", "location": "main.go:SetSnapshot", "message": "before SetSnapshot", "data": map[string]interface{}{"snapshotVersion": snapshot.GetVersion(resource.ListenerType), "nodeIDs": nodeIDs}, "timestamp": time.Now().UnixMilli()})
	// 	f.Write(append(b, '\n'))
	// 	f.Close()
	// }
	// #endregion
	for _, nodeID := range nodeIDs {
		if err := cp.cache.SetSnapshot(cp.ctx, nodeID, snapshot); err != nil {
			// #region agent log
			// dp := os.Getenv("DEBUG_LOG_PATH")
			// if dp == "" {
			// 	dp = "e:\\xcode\\service-operations\\envoy-proxy\\.cursor\\debug.log"
			// }
			// if f, e := os.OpenFile(dp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); e == nil {
			// 	b, _ := json.Marshal(map[string]interface{}{"sessionId": "debug-session", "hypothesisId": "H1", "location": "main.go:SetSnapshot", "message": "SetSnapshot failed", "data": map[string]interface{}{"error": err.Error(), "nodeID": nodeID}, "timestamp": time.Now().UnixMilli()})
			// 	f.Write(append(b, '\n'))
			// 	f.Close()
			// }
			// #endregion
			log.Printf("❌ 设置快照失败 (node=%s): %v", nodeID, err)
			return
		}
	}
	// #region agent log
	// if f, e := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); e == nil {
	// 	b, _ := json.Marshal(map[string]interface{}{"sessionId": "debug-session", "runId": "post-fix", "hypothesisId": "H3", "location": "main.go:SetSnapshot", "message": "SetSnapshot success", "data": map[string]interface{}{"version": snapshot.GetVersion(resource.ListenerType), "nodeIDs": nodeIDs}, "timestamp": time.Now().UnixMilli()})
	// 	f.Write(append(b, '\n'))
	// 	f.Close()
	// }
	// #endregion

	log.Println("✅ Envoy配置更新完成")
}

// buildSnapshot 构建配置快照
func (cp *ControlPlane) buildSnapshot(services []*consulapi.ServiceEntry) (*cache.Snapshot, error) {
	var clusters []cache_types.Resource
	var listeners []cache_types.Resource

	// 为每个服务创建集群和监听器
	for _, service := range services {
		servicePort := service.Service.Port
		serviceAddress := service.Service.Address

		// 从元数据中获取外部端口
		externalPortStr, ok := service.Service.Meta["envoy_external_port"]
		if !ok {
			log.Printf("⚠️ 服务 %s 未指定envoy_external_port元数据，跳过", service.Service.ID)
			continue
		}

		externalPort, err := strconv.Atoi(externalPortStr)
		if err != nil {
			log.Printf("⚠️ 服务 %s 的外部端口格式错误: %s，跳过", service.Service.ID, externalPortStr)
			continue
		}

		// 检查协议是否为UDP
		protocol, ok := service.Service.Meta["protocol"]
		if !ok || strings.ToLower(protocol) != "udp" {
			log.Printf("⚠️ 服务 %s 协议非UDP，跳过", service.Service.ID)
			continue
		}

		clusterName := fmt.Sprintf("cluster_%s_%d", service.Service.ID, externalPort)
		listenerName := fmt.Sprintf("listener_%d", externalPort)

		// 创建集群
		clusterResource := cp.createCluster(clusterName, serviceAddress, servicePort)
		clusters = append(clusters, clusterResource)

		// 创建UDP监听器
		listenerResource, err := cp.createUDPListener(listenerName, uint32(externalPort), clusterName)
		if err != nil {
			log.Printf("⚠️ 创建UDP监听器失败: %v", err)
			continue
		}
		listeners = append(listeners, listenerResource)

		log.Printf("📝 为服务 %s 创建配置: 外部端口 %d -> 内部 %s:%d",
			service.Service.ID, externalPort, serviceAddress, servicePort)
	}

	// 构建快照 - 仅包含集群与监听器。UDP 代理不需要 RouteConfiguration；
	// 若提供 Route 但无 listener 引用，go-control-plane 一致性检查会报错：referenced 0 != resources 1
	// #region agent log
	// dp := os.Getenv("DEBUG_LOG_PATH")
	// if dp == "" {
	// 	dp = "e:\\xcode\\service-operations\\envoy-proxy\\.cursor\\debug.log"
	// }
	// if f, e := os.OpenFile(dp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); e == nil {
	// 	b, _ := json.Marshal(map[string]interface{}{"sessionId": "debug-session", "runId": "post-fix", "hypothesisId": "H3", "location": "main.go:buildSnapshot", "message": "snapshot without RouteType", "data": map[string]interface{}{"clusterCount": len(clusters), "listenerCount": len(listeners), "hasRouteType": false}, "timestamp": time.Now().UnixMilli()})
	// 	f.Write(append(b, '\n'))
	// 	f.Close()
	// }
	// #endregion

	snapshot, err := cache.NewSnapshot(
		time.Now().Format("2006-01-02T15:04:05"), // 版本号使用当前时间
		map[resource.Type][]cache_types.Resource{
			resource.ClusterType:  clusters,
			resource.ListenerType: listeners,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("创建快照失败: %v", err)
	}

	return snapshot, nil
}

// isIP 判断是否为 IP 地址（否则视为主机名，需用 STRICT_DNS）
func isIP(s string) bool {
	return net.ParseIP(s) != nil
}

// createCluster 创建集群资源。主机名（如 game-server-1）用 STRICT_DNS，IP 用 STATIC
func (cp *ControlPlane) createCluster(name, address string, port int) *cluster.Cluster {
	typ := cluster.Cluster_STATIC
	if !isIP(address) {
		typ = cluster.Cluster_STRICT_DNS
	}
	return &cluster.Cluster{
		Name:                 name,
		ConnectTimeout:       durationpb.New(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: typ},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
		LoadAssignment: &endpoint.ClusterLoadAssignment{
			ClusterName: name,
			Endpoints: []*endpoint.LocalityLbEndpoints{{
				LbEndpoints: []*endpoint.LbEndpoint{{
					HostIdentifier: &endpoint.LbEndpoint_Endpoint{
						Endpoint: &endpoint.Endpoint{
							Address: &core.Address{
								Address: &core.Address_SocketAddress{
									SocketAddress: &core.SocketAddress{
										Protocol: core.SocketAddress_UDP,
										Address:  address,
										PortSpecifier: &core.SocketAddress_PortValue{
											PortValue: uint32(port),
										},
									},
								},
							},
						},
					},
				}},
			}},
		},
	}
}

// createUDPListener 创建UDP监听器
func (cp *ControlPlane) createUDPListener(name string, port uint32, clusterName string) (*listener.Listener, error) {
	// 创建UDP代理过滤器
	udpFilter := &udpproxy.UdpProxyConfig{
		StatPrefix: fmt.Sprintf("udp_stats_%d", port),
		RouteSpecifier: &udpproxy.UdpProxyConfig_Cluster{
			Cluster: clusterName,
		},
		IdleTimeout: durationpb.New(60 * time.Second), // 游戏场景的合理超时
	}

	anyFilter, err := anypb.New(udpFilter)
	if err != nil {
		return nil, fmt.Errorf("创建UDP过滤器失败: %v", err)
	}

	// UDP 无连接监听器必须用 ListenerFilters 配置 udp_proxy，不能使用 FilterChains（会报 connection-less UDP listener）
	return &listener.Listener{
		Name: name,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_UDP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},
		ListenerFilters: []*listener.ListenerFilter{{
			Name: "envoy.filters.udp_listener.udp_proxy",
			ConfigType: &listener.ListenerFilter_TypedConfig{
				TypedConfig: anyFilter,
			},
		}},
	}, nil
}

// runXdsServer 运行xDS服务器
func (cp *ControlPlane) runXdsServer() {
	// gRPC服务器选项
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions,
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    grpcKeepaliveTime,
			Timeout: grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcKeepaliveMinTime,
			PermitWithoutStream: true,
		}),
	)

	grpcServer := grpc.NewServer(grpcOptions...)

	// 注册xDS服务
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, cp.server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, cp.server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, cp.server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, cp.server)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, cp.server)

	// 监听端口
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cp.xdsPort))
	if err != nil {
		log.Fatalf("❌ 无法监听端口 %d: %v", cp.xdsPort, err)
	}

	log.Printf("🚀 控制平面启动，监听xDS端口: %d", cp.xdsPort)

	if err = grpcServer.Serve(lis); err != nil {
		log.Printf("❌ gRPC服务器错误: %v", err)
	}
}

// Stop 停止控制平面
func (cp *ControlPlane) Stop() {
	log.Println("🛑 正在停止控制平面...")
	cp.cancel()
}

// HealthHandler 健康检查处理器
func (cp *ControlPlane) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status": "healthy", "component": "control-plane", "timestamp": "`+time.Now().Format(time.RFC3339)+`"}`)
}

func main() {
	// 从环境变量获取配置
	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr == "" {
		consulAddr = "consul-server:8500"
	}

	xdsPortStr := os.Getenv("XDS_PORT")
	xdsPort := uint(18000)
	if xdsPortStr != "" {
		if port, err := strconv.Atoi(xdsPortStr); err == nil {
			xdsPort = uint(port)
		}
	}

	healthPortStr := os.Getenv("HEALTH_PORT")
	healthPort := 8080
	if healthPortStr != "" {
		if port, err := strconv.Atoi(healthPortStr); err == nil {
			healthPort = port
		}
	}

	log.Printf("🎮 启动游戏服务器动态UDP代理控制平面")
	log.Printf("📍 Consul地址: %s", consulAddr)
	log.Printf("📍 xDS端口: %d", xdsPort)
	log.Printf("📍 健康检查端口: %d", healthPort)

	// 创建控制平面实例
	controlPlane, err := NewControlPlane(consulAddr, xdsPort)
	if err != nil {
		log.Fatalf("❌ 创建控制平面失败: %v", err)
	}

	// 启动健康检查服务器
	go func() {
		http.HandleFunc("/health", controlPlane.HealthHandler)
		http.HandleFunc("/ready", controlPlane.HealthHandler)

		addr := fmt.Sprintf("0.0.0.0:%d", healthPort)
		log.Printf("🏥 健康检查服务器启动，监听端口: %d", healthPort)

		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Printf("⚠️ 健康检查服务器错误: %v", err)
		}
	}()

	// 启动控制平面
	go func() {
		if err := controlPlane.Start(); err != nil {
			log.Printf("❌ 控制平面启动失败: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("🛑 收到中断信号，正在关闭...")

	// 停止控制平面
	controlPlane.Stop()

	log.Println("✅ 控制平面已关闭")
}
