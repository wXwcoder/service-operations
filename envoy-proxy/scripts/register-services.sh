#!/bin/bash
# 批量注册游戏服务器到Consul的脚本

CONSUL_URL=${CONSUL_URL:-"http://localhost:8500"}

echo "开始批量注册游戏服务器到Consul..."

# 注册游戏服务器1
echo "注册 game-server-1..."
curl -X PUT ${CONSUL_URL}/v1/agent/service/register \
  -d '{
    "ID": "game-server-1",
    "Name": "game-server",
    "Tags": ["udp", "game", "battle"],
    "Address": "game-server-1",
    "Port": 8080,
    "Meta": {
      "protocol": "udp",
      "server_type": "battle"
    },
    "Check": {
      "HTTP": "http://game-server-1:9080/health",
      "Interval": "10s",
      "Timeout": "2s",
      "DeregisterCriticalServiceAfter": "5m"
    }
  }'

sleep 1

# 注册游戏服务器2
echo "注册 game-server-2..."
curl -X PUT ${CONSUL_URL}/v1/agent/service/register \
  -d '{
    "ID": "game-server-2",
    "Name": "game-server",
    "Tags": ["udp", "game", "battle"],
    "Address": "game-server-2",
    "Port": 8081,
    "Meta": {
      "protocol": "udp",
      "server_type": "battle"
    },
    "Check": {
      "HTTP": "http://game-server-2:9081/health",
      "Interval": "10s",
      "Timeout": "2s",
      "DeregisterCriticalServiceAfter": "5m"
    }
  }'

sleep 1

# 注册游戏服务器3
echo "注册 game-server-3..."
curl -X PUT ${CONSUL_URL}/v1/agent/service/register \
  -d '{
    "ID": "game-server-3",
    "Name": "game-server",
    "Tags": ["udp", "game", "battle"],
    "Address": "game-server-3",
    "Port": 8082,
    "Meta": {
      "protocol": "udp",
      "server_type": "battle"
    },
    "Check": {
      "HTTP": "http://game-server-3:9082/health",
      "Interval": "10s",
      "Timeout": "2s",
      "DeregisterCriticalServiceAfter": "5m"
    }
  }'

echo "游戏服务器注册完成！"

# 验证注册结果
echo "验证注册的服务..."
curl -s ${CONSUL_URL}/v1/health/service/game-server | python -m json.tool