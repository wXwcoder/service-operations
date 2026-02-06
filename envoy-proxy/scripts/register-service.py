#!/usr/bin/env python3
"""
游戏服务器注册脚本
用于向Consul注册UDP游戏服务器
"""

import requests
import json
import os
import sys
import time
from typing import Dict, List, Optional

class ConsulServiceRegistry:
    def __init__(self, consul_url: str = "http://consul-server:8500"):
        self.consul_url = consul_url
        
    def register_game_server(self, server_id: str, server_ip: str, server_port: int, 
                           tags: List[str] = None) -> bool:
        """注册游戏服务器到Consul"""
        
        service_data = {
            "ID": server_id,
            "Name": "game-server",
            "Tags": tags or ["udp", "game"],
            "Address": server_ip,
            "Port": server_port,
            "Meta": {
                "protocol": "udp",
                "server_type": "battle",
                "registered_at": time.strftime("%Y-%m-%d %H:%M:%S")
            },
            "Check": {
                "DeregisterCriticalServiceAfter": "5m",
                # 使用HTTP健康检查
                "HTTP": f"http://{server_ip}:{server_port + 1000}/health",
                "Interval": "10s",
                "Timeout": "2s"
            }
        }
        
        try:
            response = requests.put(
                f"{self.consul_url}/v1/agent/service/register",
                json=service_data,
                timeout=5
            )
            
            if response.status_code == 200:
                print(f"✅ 游戏服务器 {server_id} 注册成功: {server_ip}:{server_port}")
                return True
            else:
                print(f"❌ 注册失败: {response.status_code} - {response.text}")
                return False
                
        except requests.exceptions.RequestException as e:
            print(f"❌ 连接Consul失败: {e}")
            return False
    
    def deregister_game_server(self, server_id: str) -> bool:
        """从Consul注销游戏服务器"""
        
        try:
            response = requests.put(
                f"{self.consul_url}/v1/agent/service/deregister/{server_id}",
                timeout=5
            )
            
            if response.status_code == 200:
                print(f"✅ 游戏服务器 {server_id} 注销成功")
                return True
            else:
                print(f"❌ 注销失败: {response.status_code}")
                return False
                
        except requests.exceptions.RequestException as e:
            print(f"❌ 连接Consul失败: {e}")
            return False
    
    def list_game_servers(self) -> Optional[List[Dict]]:
        """获取所有已注册的游戏服务器"""
        
        try:
            response = requests.get(
                f"{self.consul_url}/v1/catalog/service/game-server",
                timeout=5
            )
            
            if response.status_code == 200:
                servers = response.json()
                print(f"📊 当前注册的游戏服务器数量: {len(servers)}")
                for server in servers:
                    print(f"   - {server['ServiceID']}: {server['ServiceAddress']}:{server['ServicePort']}")
                return servers
            else:
                print(f"❌ 获取服务器列表失败: {response.status_code}")
                return None
                
        except requests.exceptions.RequestException as e:
            print(f"❌ 连接Consul失败: {e}")
            return None

def main():
    """主函数 - 批量注册游戏服务器"""
    
    registry = ConsulServiceRegistry()
    
    # 检查Consul连接
    print("🔍 检查Consul连接...")
    try:
        response = requests.get("http://consul-server:8500/v1/agent/self", timeout=5)
        if response.status_code != 200:
            print("❌ Consul服务不可用")
            sys.exit(1)
    except:
        print("❌ 无法连接到Consul，请确保Consul服务正在运行")
        sys.exit(1)
    
    print("✅ Consul连接正常")
    
    # 批量注册游戏服务器（模拟400个服务器）
    print("\n🚀 开始注册游戏服务器...")
    
    # 注册3个演示服务器
    servers_to_register = [
        ("game-server-1", "game-server-1", 8080),
        ("game-server-2", "game-server-2", 8081), 
        ("game-server-3", "game-server-3", 8082)
    ]
    
    success_count = 0
    for server_id, server_ip, server_port in servers_to_register:
        if registry.register_game_server(server_id, server_ip, server_port):
            success_count += 1
    
    print(f"\n📊 注册完成: {success_count}/{len(servers_to_register)} 个服务器注册成功")
    
    # 显示已注册的服务器
    print("\n📋 当前已注册的游戏服务器:")
    registry.list_game_servers()
    
    print("\n✨ 游戏服务器注册完成!")

if __name__ == "__main__":
    main()