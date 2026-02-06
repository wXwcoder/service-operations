# Envoy Proxy + Consul UDP代理方案

## 项目概述

本项目实现了一个基于Envoy Proxy和Consul的UDP代理方案，专门解决游戏战斗服务器的UDP通信需求。客户端可以根据指定的IP+PORT连接到特定的游戏服务器，实现精确的路由。

### 核心特性

- ✅ **精确路由**: 基于目标端口实现精确的服务器路由
- ✅ **动态服务发现**: 使用Consul实现游戏服务器的动态注册和发现
- ✅ **负载均衡**: Envoy提供多种负载均衡策略
- ✅ **高可用性**: 支持服务器健康检查和自动故障转移
- ✅ **易于扩展**: 支持400+游戏服务器的水平扩展

## 系统架构

```
客户端 (UDP) → Envoy Proxy (端口10000) → 游戏服务器集群 (端口8080-8479)
                     ↓
                Consul (服务发现)
```

## 快速开始

### 1. 环境要求

- Docker & Docker Compose
- 4GB+ 内存
- Windows/Linux/macOS

### 2. 启动系统

```bash
# 克隆项目
git clone <repository-url>
cd envoy-proxy

# 启动所有服务
docker-compose up -d
```

### 3. 验证部署

```bash
# 检查服务状态
docker-compose ps

# 查看Envoy日志
docker logs envoy-proxy

# 查看Consul Web UI (http://localhost:8500)
```

### 4. 测试连接

```bash
# 使用测试客户端
docker exec -it test-client /app/test-client

# 或使用Python测试脚本
python scripts/test-proxy.py
```

## 组件说明

### 1. Envoy Proxy

**配置文件**: `envoy/envoy-advanced.yaml`

- **监听端口**: 10000 (UDP)
- **管理端口**: 9901 (HTTP)
- **健康检查**: 8001 (HTTP)

**关键配置**:
- UDP代理过滤器 (`envoy.filters.udp.udp_proxy`)
- 基于端口的哈希路由
- Consul服务发现集成

### 2. Consul 服务发现

**配置文件**: `consul/config.json`

- **Web UI**: http://localhost:8500
- **服务注册**: 自动注册游戏服务器
- **健康检查**: TCP端口检查

### 3. 游戏服务器

**语言**: Golang
**协议**: UDP
**端口范围**: 8080-8479 (可扩展)

每个游戏服务器启动时自动注册到Consul，包含以下元数据：
- 服务器ID
- 监听端口
- 协议类型
- 健康状态

### 4. 服务注册器

自动将游戏服务器注册到Consul，支持：
- 批量注册
- 健康检查
- 服务注销

## 使用示例

### 客户端连接

客户端连接到Envoy代理端口(10000)，Envoy根据目标端口路由到对应的游戏服务器：

```go
// 连接到Envoy代理
client := NewUDPClient("localhost", 10000)
client.Connect()

// 发送消息
response, err := client.SendMessage("PING")
```

### 消息类型

- `PING` - 连接测试
- `BATTLE <action>` - 战斗消息
- `STATUS` - 服务器状态查询
- 任意消息 - 回声测试

## 扩展配置

### 增加游戏服务器

1. 在 `docker-compose.yml` 中添加新的服务定义
2. 更新端口映射
3. 在服务注册脚本中添加服务器信息

```yaml
game-server-4:
  build:
    context: ./game-server
    dockerfile: Dockerfile
  container_name: game-server-4
  ports:
    - "8083:8083/udp"
  environment:
    - SERVER_ID=game-server-4
    - SERVER_PORT=8083
```

### 自定义路由规则

修改 `envoy/envoy-advanced.yaml` 中的路由配置：

```yaml
matcher:
  on_no_match:
    action:
      name: route
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.filters.udp.udp_proxy.v3.Route
        cluster: dynamic_game_servers
```

## 监控和管理

### Envoy管理界面

访问 http://localhost:9901 查看Envoy统计信息和配置

### Consul Web UI

访问 http://localhost:8500 查看服务注册状态和健康检查

### 健康检查

```bash
# Envoy健康检查
curl http://localhost:8001/healthz

# 游戏服务器健康检查
curl http://localhost:8500/v1/health/service/game-server
```

## 故障排除

### 常见问题

1. **连接超时**
   - 检查Docker网络配置
   - 验证端口映射是否正确

2. **服务注册失败**
   - 检查Consul服务状态
   - 验证网络连通性

3. **路由错误**
   - 检查Envoy配置
   - 验证Consul服务发现

### 日志查看

```bash
# Envoy日志
docker logs envoy-proxy

# Consul日志
docker logs consul-server

# 游戏服务器日志
docker logs game-server-1
```

## 性能优化

### UDP会话超时

调整 `idle_timeout` 配置以适应游戏会话需求：

```yaml
idle_timeout: 60s  # 根据游戏会话长度调整
```

### 负载均衡策略

支持多种负载均衡算法：
- `ROUND_ROBIN` - 轮询
- `LEAST_REQUEST` - 最少请求
- `MAGLEV` - 一致性哈希

## 安全考虑

- 使用网络隔离（Docker network）
- 限制外部访问端口
- 定期更新容器镜像
- 监控异常流量

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建Pull Request

## 许可证

MIT License