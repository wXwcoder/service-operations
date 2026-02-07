#!/usr/bin/env python3
"""
动态服务发现测试脚本
测试Envoy通过Consul DNS服务发现功能
"""

import socket
import time
import requests
import json

def test_consul_dns_discovery():
    """测试Consul DNS服务发现"""
    print("🔍 测试Consul DNS服务发现...")
    
    try:
        # 测试Consul DNS解析
        import socket
        
        # 解析game-server.service.consul
        result = socket.getaddrinfo('game-server.service.consul', 8080)
        print(f"✅ Consul DNS解析成功: {len(result)} 个IP地址")
        
        for addr_info in result:
            family, type, proto, canonname, sockaddr = addr_info
            print(f"   - {sockaddr[0]}:{sockaddr[1]}")
        
        return True
    except Exception as e:
        print(f"❌ Consul DNS解析失败: {e}")
        return False

def test_envoy_dynamic_proxy():
    """测试Envoy动态代理功能"""
    print("\n🚀 测试Envoy动态代理...")
    
    # 测试连接到Envoy代理
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5)
        
        # 发送测试消息
        test_message = "PING"
        sock.sendto(test_message.encode(), ('localhost', 10000))
        
        # 接收响应
        response, addr = sock.recvfrom(1024)
        response_text = response.decode()
        
        print(f"✅ Envoy代理响应: {response_text}")
        
        # 验证响应包含服务器标识
        if "PONG from server" in response_text:
            print("✅ 动态代理功能正常 - 请求被正确路由到游戏服务器")
            return True
        else:
            print("❌ 响应格式不正确")
            return False
            
    except socket.timeout:
        print("❌ 连接Envoy代理超时")
        return False
    except Exception as e:
        print(f"❌ 连接Envoy代理失败: {e}")
        return False
    finally:
        sock.close()

def test_consul_service_registration():
    """测试Consul服务注册状态"""
    print("\n📊 检查Consul服务注册状态...")
    
    try:
        # 查询Consul服务目录
        response = requests.get('http://localhost:8500/v1/catalog/service/game-server')
        if response.status_code == 200:
            services = response.json()
            print(f"✅ 发现 {len(services)} 个game-server实例")
            
            for service in services:
                service_id = service['ServiceID']
                address = service['ServiceAddress']
                port = service['ServicePort']
                print(f"   - {service_id}: {address}:{port}")
            
            return len(services) > 0
        else:
            print(f"❌ 查询Consul服务失败: {response.status_code}")
            return False
            
    except Exception as e:
        print(f"❌ 连接Consul失败: {e}")
        return False

def test_consul_health_checks():
    """测试Consul健康检查状态"""
    print("\n❤️ 检查Consul健康检查状态...")
    
    try:
        # 查询健康检查状态
        response = requests.get('http://localhost:8500/v1/health/service/game-server')
        if response.status_code == 200:
            health_info = response.json()
            
            healthy_count = 0
            for service in health_info:
                service_id = service['Service']['ID']
                checks = service['Checks']
                
                # 检查服务健康状态
                service_healthy = any(check['Status'] == 'passing' for check in checks if check['CheckID'].startswith('service:'))
                
                if service_healthy:
                    healthy_count += 1
                    print(f"✅ {service_id}: 健康状态正常")
                else:
                    print(f"❌ {service_id}: 健康状态异常")
            
            print(f"📊 健康服务数量: {healthy_count}/{len(health_info)}")
            return healthy_count == len(health_info)
        else:
            print(f"❌ 查询健康检查失败: {response.status_code}")
            return False
            
    except Exception as e:
        print(f"❌ 检查健康状态失败: {e}")
        return False

def test_load_balancing():
    """测试负载均衡功能"""
    print("\n⚖️ 测试负载均衡功能...")
    
    try:
        server_responses = set()
        
        # 发送多个请求，验证是否被均衡到不同服务器
        for i in range(10):
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(3)
            
            try:
                sock.sendto(f"PING_{i}".encode(), ('localhost', 10000))
                response, addr = sock.recvfrom(1024)
                response_text = response.decode()
                
                # 提取服务器标识
                if "from server" in response_text:
                    server_id = response_text.split("from server")[1].split(" at")[0].strip()
                    server_responses.add(server_id)
                    print(f"   请求 {i+1}: 由 {server_id} 处理")
                
            except socket.timeout:
                print(f"   请求 {i+1}: 超时")
            finally:
                sock.close()
            
            time.sleep(0.5)  # 短暂延迟
        
        print(f"📊 请求被分发到 {len(server_responses)} 个不同的服务器")
        
        if len(server_responses) > 1:
            print("✅ 负载均衡功能正常 - 请求被均衡分发")
            return True
        else:
            print("⚠️ 负载均衡可能未正常工作")
            return len(server_responses) > 0
            
    except Exception as e:
        print(f"❌ 负载均衡测试失败: {e}")
        return False

def main():
    """主测试函数"""
    print("🚀 Envoy动态服务发现测试开始")
    print("=" * 50)
    
    # 执行各项测试
    tests = [
        ("Consul服务注册", test_consul_service_registration),
        ("Consul健康检查", test_consul_health_checks),
        ("Envoy动态代理", test_envoy_dynamic_proxy),
        ("负载均衡", test_load_balancing),
    ]
    
    results = []
    for test_name, test_func in tests:
        try:
            result = test_func()
            results.append((test_name, result))
        except Exception as e:
            print(f"❌ {test_name}测试异常: {e}")
            results.append((test_name, False))
    
    # 输出测试总结
    print("\n" + "=" * 50)
    print("📋 测试结果总结:")
    
    passed = 0
    for test_name, result in results:
        status = "✅ 通过" if result else "❌ 失败"
        print(f"   {test_name}: {status}")
        if result:
            passed += 1
    
    print(f"\n🎯 总体结果: {passed}/{len(tests)} 项测试通过")
    
    if passed == len(tests):
        print("✨ 动态服务发现功能完全正常!")
    else:
        print("⚠️ 部分功能需要检查")

if __name__ == "__main__":
    main()