#!/usr/bin/env python3
"""
手动重新注册游戏服务器，使用HTTP健康检查
"""

import requests
import json

def register_game_server(server_id, server_ip, server_port):
    """手动注册游戏服务器"""
    
    # HTTP健康检查端口 = UDP端口 + 1000
    health_port = server_port + 1000
    
    service_data = {
        "ID": server_id,
        "Name": "game-server",
        "Tags": ["udp", "game"],
        "Address": server_ip,
        "Port": server_port,
        "Meta": {
            "protocol": "udp",
            "server_type": "battle"
        },
        "Check": {
            "DeregisterCriticalServiceAfter": "5m",
            "HTTP": f"http://{server_ip}:{health_port}/health",
            "Interval": "10s",
            "Timeout": "2s"
        }
    }
    
    try:
        response = requests.put(
            "http://localhost:8500/v1/agent/service/register",
            json=service_data,
            timeout=5
        )
        
        if response.status_code == 200:
            print(f"✅ 游戏服务器 {server_id} 注册成功")
            print(f"   UDP端口: {server_port}, 健康检查端口: {health_port}")
            return True
        else:
            print(f"❌ 注册失败: {response.status_code} - {response.text}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ 连接Consul失败: {e}")
        return False

def main():
    """主函数"""
    
    print("🚀 手动重新注册游戏服务器...")
    
    # 注册3个游戏服务器
    servers = [
        ("game-server-1", "game-server-1", 8080),
        ("game-server-2", "game-server-2", 8081),
        ("game-server-3", "game-server-3", 8082)
    ]
    
    success_count = 0
    for server_id, server_ip, server_port in servers:
        if register_game_server(server_id, server_ip, server_port):
            success_count += 1
    
    print(f"\n📊 注册完成: {success_count}/{len(servers)} 个服务器注册成功")
    
    # 检查健康检查状态
    print("\n🔍 检查健康检查状态...")
    try:
        response = requests.get("http://localhost:8500/v1/agent/checks", timeout=5)
        if response.status_code == 200:
            checks = response.json()
            for check_id, check_info in checks.items():
                status = check_info.get("Status", "unknown")
                check_type = check_info.get("Type", "unknown")
                print(f"   {check_id}: {status} (Type: {check_type})")
    except Exception as e:
        print(f"❌ 检查健康状态失败: {e}")

if __name__ == "__main__":
    main()