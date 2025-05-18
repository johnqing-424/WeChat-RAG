#!/bin/bash

# 测试本地网络连接
echo "测试与RagFlow服务器的网络连接..."

# 检查服务可达性 (使用原始IP地址)
echo -e "\n1. 检查外部IP连接(114.215.255.105:8081)..."
curl -s -o /dev/null -w "状态码: %{http_code}, 连接时间: %{time_connect}秒, 总时间: %{time_total}秒\n" http://114.215.255.105:8081/api/v1/dbs

# 检查服务可达性 (使用容器名)
echo -e "\n2. 检查容器名连接(ragflow-server)..."
curl -s -o /dev/null -w "状态码: %{http_code}, 连接时间: %{time_connect}秒, 总时间: %{time_total}秒\n" http://ragflow-server/api/v1/dbs 2>/dev/null || echo "无法连接到ragflow-server，可能需要加入Docker网络"

# 检查容器网络
echo -e "\n3. 检查Docker网络配置..."
docker network inspect docker_ragflow

# 尝试从容器内访问
echo -e "\n4. 尝试从临时容器内访问RagFlow服务..."
docker run --rm --network docker_ragflow curlimages/curl curl -s -o /dev/null -w "状态码: %{http_code}, 连接时间: %{time_connect}秒, 总时间: %{time_total}秒\n" http://ragflow-server/api/v1/dbs || echo "临时容器测试失败" 