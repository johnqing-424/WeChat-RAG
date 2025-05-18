#!/bin/bash

# 将当前主机加入到RagFlow的Docker网络中
echo "配置网络环境..."

# 1. 确保已停止本地运行的WeChat-RAG进程
pkill -f WeChat-RAG || true
echo "已停止本地WeChat-RAG进程"

# 2. 创建/确保网络存在
docker network inspect docker_ragflow >/dev/null 2>&1 || docker network create docker_ragflow
echo "已确认网络可用"

# 3. 配置hosts映射 - 这里我们保留这部分以备后续与RAGFlow集成
RAGFLOW_IP="127.0.0.1"  # 临时设置，后续集成时再修改
if grep -q "ragflow-server" /etc/hosts; then
    echo "更新hosts文件..."
    sed -i '/ragflow-server/d' /etc/hosts
fi
echo "$RAGFLOW_IP ragflow-server" >> /etc/hosts
echo "hosts文件更新完成"

# 4. 删除之前的容器（如果存在）
docker rm -f wechat-rag-container >/dev/null 2>&1 || true
echo "已清理旧容器"

echo "网络配置完成"