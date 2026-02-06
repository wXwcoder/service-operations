#!/usr/bin/env python3
"""
Envoy精确路由功能测试
验证通过不同端口精确路由到指定游戏服务器的功能
"""

import socket
import time

def test_exact_routing():
    """测试精确路由功能"""
    
    print("🎯 Envoy精确路由功能测试")
    print("=" * 60)
    
    # 精确路由配置：端口 → 游戏服务器
    routing_config = {
        10000: "game-server-1",
        10001: "game-server-2", 
        10002: "game-server-3"
    }
    
    test_results = {}
    
    for envoy_port, expected_server in routing_config.items():
        print(f"\n🔍 测试端口 {envoy_port} → {expected_server}")
        print("-" * 40)
        
        try:
            # 创建UDP socket
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(5)
            
            # 发送PING消息
            server_address = ('localhost', envoy_port)
            start_time = time.time()
            sock.sendto(b'PING', server_address)
            
            # 接收响应
            response, _ = sock.recvfrom(1024)
            end_time = time.time()
            
            response_time = round((end_time - start_time) * 1000, 2)
            response_text = response.decode()
            
            # 检查响应是否来自正确的服务器
            if expected_server in response_text:
                print(f"✅ 路由正确: {response_text} ({response_time}ms)")
                test_results[envoy_port] = "PASS"
            else:
                print(f"❌ 路由错误: 期望 {expected_server}, 实际响应: {response_text}")
                test_results[envoy_port] = "FAIL"
            
            sock.close()
            
        except socket.timeout:
            print(f"❌ 请求超时")
            test_results[envoy_port] = "TIMEOUT"
        except Exception as e:
            print(f"❌ 错误: {e}")
            test_results[envoy_port] = "ERROR"
    
    # 测试总结
    print("\n" + "=" * 60)
    print("📊 精确路由测试总结")
    print("=" * 60)
    
    passed = sum(1 for result in test_results.values() if result == "PASS")
    total = len(test_results)
    
    for port, result in test_results.items():
        expected_server = routing_config[port]
        status_emoji = "✅" if result == "PASS" else "❌"
        print(f"{status_emoji} 端口 {port} → {expected_server}: {result}")
    
    print(f"\n📈 测试结果: {passed}/{total} 通过")
    
    if passed == total:
        print("🎉 精确路由功能完全正常！")
    else:
        print("⚠️  部分路由功能需要检查")

def test_individual_ports():
    """分别测试每个端口的连接"""
    
    print("\n" + "=" * 60)
    print("🔧 详细端口测试")
    print("=" * 60)
    
    test_cases = [
        (10000, "game-server-1"),
        (10001, "game-server-2"),
        (10002, "game-server-3")
    ]
    
    for port, server_name in test_cases:
        print(f"\n🎯 测试端口 {port} ({server_name})")
        
        # 测试PING
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(5)
            sock.sendto(b'PING', ('localhost', port))
            response, _ = sock.recvfrom(1024)
            print(f"   ✅ PING: {response.decode()}")
            sock.close()
        except Exception as e:
            print(f"   ❌ PING失败: {e}")
        
        # 测试STATUS
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(5)
            sock.sendto(b'STATUS', ('localhost', port))
            response, _ = sock.recvfrom(1024)
            print(f"   ✅ STATUS: {response.decode()}")
            sock.close()
        except Exception as e:
            print(f"   ❌ STATUS失败: {e}")
        
        # 测试BATTLE
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(5)
            sock.sendto(b'BATTLE test', ('localhost', port))
            response, _ = sock.recvfrom(1024)
            print(f"   ✅ BATTLE: {response.decode()}")
            sock.close()
        except Exception as e:
            print(f"   ❌ BATTLE失败: {e}")

def main():
    """主函数"""
    
    print("🚀 Envoy Proxy精确路由验证")
    print("=" * 60)
    
    # 测试精确路由
    test_exact_routing()
    
    # 详细测试每个端口
    test_individual_ports()
    
    print("\n" + "=" * 60)
    print("💡 使用说明")
    print("=" * 60)
    print("""
通过不同的Envoy端口精确路由到指定的游戏服务器：

🔹 连接到 localhost:10000 → game-server-1
🔹 连接到 localhost:10001 → game-server-2  
🔹 连接到 localhost:10002 → game-server-3

示例命令：
# 连接到game-server-1
echo "PING" | nc -u localhost 10000

# 连接到game-server-2
echo "STATUS" | nc -u localhost 10001

# 连接到game-server-3
echo "BATTLE test" | nc -u localhost 10002

使用更新后的测试客户端：
docker exec -it test-client /app/test-client
""")

if __name__ == "__main__":
    main()