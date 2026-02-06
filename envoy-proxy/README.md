# Envoy Proxy + Consul UDP代理方案

## 项目概述

本项目实现了一个基于Envoy Proxy和Consul的UDP代理方案，专门解决游戏战斗服务器的UDP通信需求。通过Envoy的精确路由功能，客户端可以根据指定的端口号连接到特定的游戏服务器。

### 核心特性

- ✅ **精确路由**: 基于端口号实现精确的服务器路由（10000→game-server-1, 10001→game-server-2, 10002→game-server-3）
- ✅ **动态服务发现**: 使用Consul实现游戏服务器的动态注册和发现
- ✅ **负载均衡**: Envoy提供多种负载均衡策略
- ✅ **高可用性**: 支持HTTP健康检查和自动故障转移
- ✅ **易于扩展**: 支持400+游戏服务器的水平扩展

## 系统架构

```
客户端 (UDP) → Envoy Proxy (端口10000-10002) → 对应的游戏服务器
                     ↓
                Consul (服务发现 + 健康检查)

端口映射关系：
- localhost:10000 → game-server-1:8080
- localhost:10001 → game-server-2:8081  
- localhost:10002 → game-server-3:8082
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

**配置文件**: `envoy/envoy-precise-routing.yaml`

- **监听端口**: 10000-10002 (UDP)
- **管理端口**: 9901 (HTTP)

**关键配置**:
- 3个独立的UDP监听器，分别对应3个游戏服务器
- UDP代理过滤器 (`envoy.filters.udp.udp_proxy`)
- 基于端口号的精确路由
- STRICT_DNS集群发现

**精确路由配置**:
```yaml
# 游戏服务器1监听器
- name: game_server_1_listener
  address:
    socket_address:
      protocol: UDP
      address: 0.0.0.0
      port_value: 10000
  cluster: game_server_1_cluster
```

### 2. Consul 服务发现

**配置文件**: `consul/config.json`

- **Web UI**: http://localhost:8500
- **服务注册**: 自动注册游戏服务器
- **健康检查**: HTTP健康检查（端口9080-9082）

**健康检查修复**:
- 修复了UDP健康检查问题，改用HTTP健康检查端点
- 每个游戏服务器提供`/health`端点用于健康检查
- 健康检查端口 = UDP端口 + 1000

### 3. 游戏服务器

**语言**: Golang
**协议**: UDP
**端口范围**: 8080-8479 (可扩展)

**新增功能**:
- HTTP健康检查服务器（端口9080-9082）
- 自动注册到Consul服务发现
- 支持精确路由标识

每个游戏服务器启动时自动注册到Consul，包含以下元数据：
- 服务器ID
- 监听端口
- 协议类型
- 健康状态
- 注册时间戳

### 4. 服务注册器

**配置文件**: `scripts/register-service.py`

自动将游戏服务器注册到Consul，支持：
- 批量注册3个游戏服务器实例
- HTTP健康检查配置
- 服务注销和重新注册
- 服务元数据管理

**修复内容**:
- 更新健康检查配置，使用HTTP检查代替TCP检查
- 修复服务注册脚本，确保正确应用健康检查配置

## 使用示例

### 精确路由连接

客户端通过不同的Envoy端口精确路由到指定的游戏服务器：

#### 方式1：使用交互式测试客户端
```bash
docker exec -it test-client /app/test-client

# 客户端会显示选择菜单：
# 🚀 Envoy UDP代理测试客户端
# ================================
# 请选择要连接的游戏服务器:
# 1. game-server-1 (端口: 10000)
# 2. game-server-2 (端口: 10001)  
# 3. game-server-3 (端口: 10002)
# 请输入选择 (1-3):
```

#### 方式2：直接使用端口号连接
```bash
# 连接到game-server-1
echo "PING" | nc -u localhost 10000

# 连接到game-server-2
echo "STATUS" | nc -u localhost 10001

# 连接到game-server-3  
echo "BATTLE test" | nc -u localhost 10002
```

#### 方式3：编程实现
```python
import socket

# 连接到game-server-1
sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.sendto(b'PING', ('localhost', 10000))
response, _ = sock.recvfrom(1024)
print(response.decode())  # 输出: PONG from server game-server-1...
```

### 消息类型

- `PING` - 连接测试，返回服务器标识和时间戳
- `BATTLE <action>` - 战斗消息处理
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

修改 `envoy/envoy-precise-routing.yaml` 中的路由配置：

#### 添加新的游戏服务器路由
```yaml
# 添加新的监听器
- name: game_server_4_listener
  address:
    socket_address:
      protocol: UDP
      address: 0.0.0.0
      port_value: 10003
  listener_filters:
  - name: envoy.filters.udp.udp_proxy
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.filters.udp.udp_proxy.v3.UdpProxyConfig
      stat_prefix: game_server_4_proxy
      cluster: game_server_4_cluster
      idle_timeout: 60s

# 添加新的集群配置
- name: game_server_4_cluster
  connect_timeout: 0.25s
  type: STRICT_DNS
  lb_policy: ROUND_ROBIN
  load_assignment:
    cluster_name: game_server_4_cluster
    endpoints:
    - lb_endpoints:
      - endpoint:
          address:
            socket_address:
              address: game-server-4
              port_value: 8083
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

### 已修复的问题

1. **✅ Consul健康检查失败**
   - **问题**: Consul使用TCP检查UDP端口导致连接失败
   - **修复**: 改用HTTP健康检查端点，每个游戏服务器提供`/health`接口
   - **验证**: 所有游戏服务器健康检查状态为"passing"

2. **✅ Envoy配置兼容性**
   - **问题**: `hash_policy`字段在当前Envoy版本中不支持
   - **修复**: 移除不兼容配置，使用简化的UDP代理配置
   - **验证**: Envoy正常启动，加载3个监听器和3个集群

3. **✅ 精确路由功能**
   - **问题**: 客户端无法指定连接到特定的游戏服务器
   - **修复**: 实现基于端口号的精确路由机制
   - **验证**: 端口10000→game-server-1, 10001→game-server-2, 10002→game-server-3

### 常见问题

1. **连接超时**
   - 检查Docker网络配置
   - 验证端口映射是否正确
   - 确认Envoy代理是否正常运行

2. **服务注册失败**
   - 检查Consul服务状态
   - 验证网络连通性
   - 查看服务注册器日志

3. **路由错误**
   - 检查Envoy配置是否正确加载
   - 验证精确路由端口映射
   - 使用测试脚本验证路由功能

### 日志查看

```bash
# Envoy日志
docker logs envoy-proxy

# Consul日志
docker logs consul-server

# 游戏服务器日志
docker logs game-server-1

# 健康检查状态
curl http://localhost:8500/v1/agent/checks | python -m json.tool
```

### 测试工具

```bash
# 基本功能测试
python scripts/test-proxy.py

# 精确路由测试
python scripts/test-exact-routing.py

# 路由方案演示
python scripts/test-precise-routing.py
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