#!/bin/bash

# 将当前主机加入到RagFlow的Docker网络中
echo "将当前主机加入到RagFlow的Docker网络中..."

# 1. 创建一个网络代理容器
echo "1. 创建一个网络代理容器..."
docker run -d --name network-proxy --network docker_ragflow --restart always alpine sleep infinity
echo "网络代理容器已创建"

# 2. 创建网络地址映射
echo "2. 创建/etc/hosts映射..."
RAGFLOW_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ragflow-server)
if [ -z "$RAGFLOW_IP" ]; then
    echo "无法获取ragflow-server的IP地址，尝试从网络配置获取..."
    RAGFLOW_IP=$(docker network inspect docker_ragflow | grep -A 5 "ragflow-server" | grep "IPv4Address" | cut -d'"' -f4 | cut -d'/' -f1)
fi

if [ -n "$RAGFLOW_IP" ]; then
    echo "将在hosts文件中添加: $RAGFLOW_IP ragflow-server"
    # 检查是否已存在
    if grep -q "ragflow-server" /etc/hosts; then
        echo "hosts文件中已存在ragflow-server记录，正在更新..."
        sed -i '/ragflow-server/d' /etc/hosts
    fi
    echo "$RAGFLOW_IP ragflow-server" >> /etc/hosts
    echo "hosts文件更新完成"
else
    echo "错误: 无法获取ragflow-server的IP地址"
    exit 1
fi

# 3. 重新启动WeChat-RAG服务
echo "3. 重新编译并启动WeChat-RAG服务..."
# 停止当前运行的服务
pkill -f WeChat-RAG

# 重新编译
go build -o WeChat-RAG

# 后台启动新服务
./WeChat-RAG > wechat.log 2>&1 &
echo "WeChat-RAG服务已重启"

# 4. 测试连接
echo "4. 测试与RagFlow的连接..."
sleep 2
curl -s -o /dev/null -w "RagFlow服务连接测试: 状态码: %{http_code}\n" http://ragflow-server/api/v1/dbs

echo "网络配置完成" 