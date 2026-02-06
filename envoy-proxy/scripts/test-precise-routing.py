#!/usr/bin/env python3
"""
Envoy精确路由测试脚本
演示如何通过不同的Envoy端口精确路由到指定的游戏服务器
"""

import socket
import time

def test_precise_routing():
    """测试精确路由功能"""
    
    print("🚀 Envoy精确路由测试")
    print("=" * 50)
    
    # 定义端口映射关系
    server_mapping = {
        "game-server-1": 10000,
        "game-server-2": 10001, 
        "game-server-3": 10002
    }
    
    # 测试消息
    test_messages = ["PING", "STATUS", "BATTLE test"]
    
    for server_name, envoy_port in server_mapping.items():
        print(f"\n🎯 测试连接到 {server_name} (Envoy端口: {envoy_port})")
        print("-" * 40)
        
        for message in test_messages:
            try:
                # 创建UDP socket
                sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
                sock.settimeout(5)
                
                # 发送消息到Envoy代理
                server_address = ('localhost', envoy_port)
                start_time = time.time()
                sock.sendto(message.encode(), server_address)
                
                # 接收响应
                response, _ = sock.recvfrom(1024)
                end_time = time.time()
                
                response_time = round((end_time - start_time) * 1000, 2)
                response_text = response.decode()
                
                # 检查响应是否来自正确的服务器
                if server_name in response_text:
                    print(f"✅ {message}: {response_text} ({response_time}ms)")
                else:
                    print(f"❌ {message}: 响应来自错误的服务器")
                
                sock.close()
                
            except socket.timeout:
                print(f"❌ {message}: 请求超时")
            except Exception as e:
                print(f"❌ {message}: 错误 - {e}")
    
    print("\n" + "=" * 50)
    print("📋 精确路由测试总结:")
    print("✅ 方案设计: 基于端口号的精确路由")
    print("✅ 路由机制: Envoy监听不同端口，映射到不同游戏服务器")
    print("✅ 客户端使用: 通过指定Envoy端口来选择目标服务器")
    print("\n💡 使用说明:")
    print("   连接到 localhost:10000 → game-server-1")
    print("   连接到 localhost:10001 → game-server-2") 
    print("   连接到 localhost:10002 → game-server-3")

def demonstrate_client_usage():
    """演示客户端使用方法"""
    
    print("\n" + "=" * 50)
    print("👤 客户端使用示例")
    print("=" * 50)
    
    print("\n1. 使用更新后的测试客户端:")
    print("   docker exec -it test-client /app/test-client")
    print("   → 客户端会提示选择游戏服务器 (1-3)")
    print("   → 自动连接到对应的Envoy端口")
    
    print("\n2. 直接使用UDP客户端:")
    print("   # 连接到game-server-1")
    print("   echo 'PING' | nc -u localhost 10000")
    
    print("   # 连接到game-server-2") 
    print("   echo 'STATUS' | nc -u localhost 10001")
    
    print("   # 连接到game-server-3")
    print("   echo 'BATTLE test' | nc -u localhost 10002")
    
    print("\n3. 编程实现:")
    print("   import socket")
    print("   ")
    print("   # 连接到game-server-1")
    print("   sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)")
    print("   sock.sendto(b'PING', ('localhost', 10000))")
    print("   response, _ = sock.recvfrom(1024)")
    print("   print(response.decode())")

def main():
    """主函数"""
    
    print("🎯 Envoy Proxy精确路由方案")
    print("=" * 50)
    
    # 测试精确路由（如果Envoy正在运行）
    try:
        test_precise_routing()
    except Exception as e:
        print(f"⚠️  Envoy服务未运行，显示方案设计:")
        print("\n📋 精确路由方案设计:")
        print("✅ Envoy配置: 3个独立的UDP监听器")
        print("   - 端口10000 → game-server-1:8080")
        print("   - 端口10001 → game-server-2:8081") 
        print("   - 端口10002 → game-server-3:8082")
        print("✅ 客户端: 通过端口号选择目标服务器")
        print("✅ 优势: 精确路由，无需复杂的协议解析")
    
    demonstrate_client_usage()
    
    print("\n✨ 方案优势总结:")
    print("✅ 精确控制: 客户端可以指定连接到特定的游戏服务器")
    print("✅ 简单易用: 基于端口号的路由，无需复杂配置")
    print("✅ 扩展性强: 支持400+服务器的水平扩展")
    print("✅ 兼容性好: 与现有UDP协议完全兼容")

if __name__ == "__main__":
    main()