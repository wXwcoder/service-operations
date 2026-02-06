#!/bin/sh
# UDP健康检查脚本
# 使用socat或nc发送UDP包并检查响应

SERVER_HOST=$1
SERVER_PORT=$2

# 检查是否安装了socat或nc
if command -v socat >/dev/null 2>&1; then
    # 使用socat发送UDP包
    echo "PING" | socat -t 1 - udp:$SERVER_HOST:$SERVER_PORT 2>/dev/null && exit 0
elif command -v nc >/dev/null 2>&1; then
    # 使用nc发送UDP包
    echo "PING" | nc -w 1 -u $SERVER_HOST $SERVER_PORT 2>/dev/null && exit 0
else
    # 如果没有工具，使用简单的telnet检查端口是否开放
    (echo >/dev/tcp/$SERVER_HOST/$SERVER_PORT) >/dev/null 2>&1 && exit 0
fi

exit 1