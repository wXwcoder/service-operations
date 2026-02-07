# 游戏服务器动态UDP代理系统

基于Envoy + Consul + Control Plane的动态UDP端口转发架构，专为游戏战斗服设计。

## 架构概述

- **Consul**: 服务发现和健康检查
- **Control Plane**: 基于go-control-plane的xDS服务器，监听Consul服务变化并动态生成Envoy配置
- **Envoy**: 动态UDP代理，根据Control Plane配置监听不同端口并转发到对应的游戏服务器
- **Game Servers**: 游戏战斗服务器，注册到Consul并暴露UDP端口

## 快速启动

```bash
# 启动整个系统
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f control-plane
```

## 配置说明

### 游戏服务器注册到Consul

游戏服务器启动时会向Consul注册，重要配置项包括：
- `Name`: 服务名称（必须为`game-server`）
- `Meta.envoy_external_port`: 指定外部访问的UDP端口
- `Meta.protocol`: 协议类型（必须为`udp`）

### 端口映射

- 外部端口10000 → game-server-1:8080
- 外部端口10001 → game-server-2:8081
- 外部端口10002 → game-server-3:8082

## 测试

启动系统后，可以通过以下命令测试UDP转发：

```bash
# 测试外部端口10000（转发到game-server-1）
echo "PING" | nc -u -w1 localhost 10000

# 测试外部端口10001（转发到game-server-2）
echo "PING" | nc -u -w1 localhost 10001

# 测试外部端口10002（转发到game-server-3）
echo "PING" | nc -u -w1 localhost 10002
```

## 组件详情

### Control Plane

位于 `control-plane/` 目录，实现了以下功能：
- 监听Consul服务注册/注销事件
- 动态生成Envoy的Listener和Cluster配置
- 通过xDS协议推送配置给Envoy

### Envoy配置

位于 `envoy/envoy-dynamic-udp.yaml`，配置为：
- 从Control Plane动态获取配置
- 监听多个UDP端口（10000-10100）
- 根据配置将流量转发到对应的游戏服务器

### 游戏服务器

位于 `game-server/` 目录，实现了：
- UDP消息处理
- Consul服务注册
- HTTP健康检查接口

## 环境变量

### Control Plane
- `CONSUL_ADDR`: Consul服务器地址 (默认: consul-server:8500)
- `XDS_PORT`: xDS服务端口 (默认: 18000)
- `HEALTH_PORT`: 健康检查端口 (默认: 8080)

### Game Server
- `SERVER_ID`: 服务器唯一标识
- `SERVER_PORT`: 内部UDP端口
- `EXTERNAL_PORT`: 外部UDP端口
- `CONSUL_URL`: Consul服务器URL

## 故障排查

1. 检查所有服务是否正常运行：
   ```bash
   docker-compose ps
   ```

2. 查看Control Plane日志：
   ```bash
   docker-compose logs control-plane
   ```

3. 检查Consul服务注册：
   - 访问 http://localhost:8500
   - 查看是否有名为`game-server`的服务注册

4. 验证端口监听：
   ```bash
   netstat -an | grep UDP | grep 1000
   ```