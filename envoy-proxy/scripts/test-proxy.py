#!/usr/bin/env python3
"""
Envoy代理测试脚本
用于测试UDP代理功能和负载均衡
"""

import socket
import time
import threading
from concurrent.futures import ThreadPoolExecutor
from typing import List, Dict

class UDPProxyTester:
    def __init__(self, proxy_host: str = "localhost", proxy_port: int = 10000):
        self.proxy_host = proxy_host
        self.proxy_port = proxy_port
        
    def send_udp_message(self, message: str, timeout: int = 5) -> str:
        """发送UDP消息到代理"""
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(timeout)
            
            server_address = (self.proxy_host, self.proxy_port)
            sock.sendto(message.encode(), server_address)
            
            response, _ = sock.recvfrom(1024)
            sock.close()
            
            return response.decode()
            
        except socket.timeout:
            return "TIMEOUT: 请求超时"
        except Exception as e:
            return f"ERROR: {str(e)}"
    
    def test_single_message(self, message: str) -> Dict:
        """测试单个消息"""
        start_time = time.time()
        response = self.send_udp_message(message)
        end_time = time.time()
        
        return {
            "message": message,
            "response": response,
            "response_time": round((end_time - start_time) * 1000, 2)
        }
    
    def test_basic_functionality(self) -> bool:
        """测试基本功能"""
        print("🧪 测试Envoy代理基本功能...")
        
        tests = [
            ("PING", "PONG"),
            ("BATTLE test", "BATTLE_RESPONSE"),
            ("STATUS", "STATUS_RESPONSE")
        ]
        
        success_count = 0
        for message, expected_prefix in tests:
            result = self.test_single_message(message)
            
            if result["response"].startswith(expected_prefix):
                print(f"✅ {message}: {result['response']} ({result['response_time']}ms)")
                success_count += 1
            else:
                print(f"❌ {message}: {result['response']}")
        
        return success_count == len(tests)
    
    def test_load_balancing(self, num_requests: int = 100) -> Dict:
        """测试负载均衡"""
        print(f"\n⚖️  测试负载均衡 ({num_requests} 个请求)...")
        
        responses = []
        start_time = time.time()
        
        with ThreadPoolExecutor(max_workers=10) as executor:
            futures = [executor.submit(self.send_udp_message, f"PING_{i}") 
                      for i in range(num_requests)]
            
            for future in futures:
                responses.append(future.result())
        
        end_time = time.time()
        
        # 分析响应
        server_responses = {}
        for response in responses:
            if "from server" in response:
                # 提取服务器ID
                parts = response.split("from server")
                if len(parts) > 1:
                    server_id = parts[1].split()[0]
                    server_responses[server_id] = server_responses.get(server_id, 0) + 1
        
        total_time = round((end_time - start_time) * 1000, 2)
        avg_time = round(total_time / num_requests, 2)
        
        print(f"📊 总耗时: {total_time}ms")
        print(f"📊 平均响应时间: {avg_time}ms")
        print(f"📊 请求分布:")
        
        for server_id, count in server_responses.items():
            percentage = (count / num_requests) * 100
            print(f"   - {server_id}: {count} 次 ({percentage:.1f}%)")
        
        return {
            "total_requests": num_requests,
            "total_time_ms": total_time,
            "avg_time_ms": avg_time,
            "distribution": server_responses
        }
    
    def test_concurrent_clients(self, num_clients: int = 10, requests_per_client: int = 10) -> Dict:
        """测试并发客户端"""
        print(f"\n👥 测试并发客户端 ({num_clients} 个客户端, 每个 {requests_per_client} 个请求)...")
        
        def client_worker(client_id: int):
            results = []
            for i in range(requests_per_client):
                message = f"CLIENT_{client_id}_REQUEST_{i}"
                result = self.test_single_message(message)
                results.append(result)
            return results
        
        start_time = time.time()
        
        with ThreadPoolExecutor(max_workers=num_clients) as executor:
            futures = [executor.submit(client_worker, i) for i in range(num_clients)]
            all_results = []
            
            for future in futures:
                all_results.extend(future.result())
        
        end_time = time.time()
        
        total_requests = num_clients * requests_per_client
        total_time = round((end_time - start_time) * 1000, 2)
        throughput = round(total_requests / (total_time / 1000), 2)
        
        print(f"📊 总请求数: {total_requests}")
        print(f"📊 总耗时: {total_time}ms")
        print(f"📊 吞吐量: {throughput} 请求/秒")
        
        # 统计成功率
        success_count = sum(1 for r in all_results if not r["response"].startswith("ERROR") and not r["response"].startswith("TIMEOUT"))
        success_rate = (success_count / total_requests) * 100
        
        print(f"📊 成功率: {success_rate:.1f}%")
        
        return {
            "total_clients": num_clients,
            "requests_per_client": requests_per_client,
            "total_requests": total_requests,
            "total_time_ms": total_time,
            "throughput": throughput,
            "success_rate": success_rate
        }

def main():
    """主测试函数"""
    tester = UDPProxyTester()
    
    print("🚀 Envoy UDP代理测试开始")
    print("=" * 50)
    
    # 1. 测试基本功能
    basic_success = tester.test_basic_functionality()
    
    if not basic_success:
        print("\n❌ 基本功能测试失败，停止后续测试")
        return
    
    # 2. 测试负载均衡
    lb_results = tester.test_load_balancing(50)
    
    # 3. 测试并发性能
    concurrent_results = tester.test_concurrent_clients(5, 20)
    
    print("\n" + "=" * 50)
    print("✨ 测试完成!")
    
    # 输出总结报告
    print("\n📋 测试总结报告:")
    print(f"✅ 基本功能: {'通过' if basic_success else '失败'}")
    print(f"📊 负载均衡: {lb_results['total_requests']} 个请求, {lb_results['total_time_ms']}ms")
    print(f"⚡ 并发性能: {concurrent_results['throughput']} 请求/秒, {concurrent_results['success_rate']:.1f}% 成功率")

if __name__ == "__main__":
    main()