#!/bin/bash
set -e

# 配置
MAX_RETRIES=5
RAGFLOW_SERVER="ragflow-server"
RAGFLOW_PORT=9380
RAGFLOW_ALT_PORT=8081
FALLBACK_RAGFLOW_HOST="114.215.255.105"
FALLBACK_RAGFLOW_PORT="8081"
CONFIG_FILE="/app/config.yml"

echo "WeChat-RAG 智能网络自修复启动脚本"
echo "===================================="

# 检测RAGFLOW服务器连接性
check_ragflow_connection() {
    local host=$1
    local port=$2
    echo "检测连接: $host:$port..."
    
    # 首先尝试ping通主机
    if ping -c 1 -W 1 $host >/dev/null 2>&1; then
        echo "✅ 可以ping通 $host"
        
        # 然后测试端口连接
        if nc -z -w2 $host $port >/dev/null 2>&1; then
            echo "✅ 端口 $port 开放"
            
            # 最后尝试HTTP连接
            local status=$(curl -s -o /dev/null -w "%{http_code}" http://${host}:${port}/api/v1/dbs 2>/dev/null || echo "000")
            if [[ "$status" =~ ^[2-3][0-9][0-9]$ ]]; then
                echo "✅ HTTP连接成功，状态码: $status"
                return 0
            else
                echo "❌ HTTP连接失败，状态码: $status"
            fi
        else
            echo "❌ 端口 $port 不可访问"
        fi
    else
        echo "❌ 无法ping通 $host"
    fi
    return 1
}

# 尝试DNS解析
resolve_dns() {
    local hostname=$1
    echo "尝试解析 $hostname..."
    
    # 使用多种工具尝试解析
    local ip=$(getent hosts $hostname | awk '{ print $1 }' || 
              dig +short $hostname || 
              nslookup $hostname | grep -A1 Name | grep Address | awk '{print $2}')
    
    if [[ -n "$ip" ]]; then
        echo "✅ 解析成功: $hostname -> $ip"
        return 0
    else
        echo "❌ 无法解析 $hostname"
        return 1
    fi
}

# 修复配置文件中的URL
update_config_url() {
    local old_url=$1
    local new_url=$2
    echo "更新配置: $old_url -> $new_url"
    
    # 备份原配置
    cp $CONFIG_FILE ${CONFIG_FILE}.bak
    
    # 更新配置中的URL (适应YAML格式)
    sed -i "s|base_url: \"$old_url\"|base_url: \"$new_url\"|g" $CONFIG_FILE
    
    echo "✅ 配置已更新"
}

# 自动发现RAGFlow服务
discover_ragflow() {
    echo "开始自动发现RAGFlow服务..."
    
    # 1. 尝试直接连接默认设置
    if check_ragflow_connection $RAGFLOW_SERVER $RAGFLOW_PORT; then
        echo "✅ 默认配置可用: http://$RAGFLOW_SERVER:$RAGFLOW_PORT"
        RAGFLOW_URL="http://$RAGFLOW_SERVER:$RAGFLOW_PORT"
        return 0
    elif check_ragflow_connection $RAGFLOW_SERVER $RAGFLOW_ALT_PORT; then
        echo "✅ 备选端口可用: http://$RAGFLOW_SERVER:$RAGFLOW_ALT_PORT"
        RAGFLOW_URL="http://$RAGFLOW_SERVER:$RAGFLOW_ALT_PORT"
        return 0
    fi
    
    # 2. 尝试使用Docker网络查找
    echo "尝试在Docker网络中查找RAGFlow服务..."
    
    # 检查是否在docker_ragflow网络中
    if ip -o addr | grep -q "docker_ragflow"; then
        echo "✅ 检测到docker_ragflow网络"
    else
        echo "⚠️ 未在docker_ragflow网络中，尝试使用备用方法"
    fi
    
    # 3. 尝试使用环境变量中提供的地址
    if [[ -n "$RAGFLOW_HOST" && -n "$RAGFLOW_PORT" ]]; then
        if check_ragflow_connection $RAGFLOW_HOST $RAGFLOW_PORT; then
            echo "✅ 环境变量配置可用: http://$RAGFLOW_HOST:$RAGFLOW_PORT"
            RAGFLOW_URL="http://$RAGFLOW_HOST:$RAGFLOW_PORT"
            return 0
        fi
    fi
    
    # 4. 尝试使用回退地址
    if check_ragflow_connection $FALLBACK_RAGFLOW_HOST $FALLBACK_RAGFLOW_PORT; then
        echo "✅ 回退地址可用: http://$FALLBACK_RAGFLOW_HOST:$FALLBACK_RAGFLOW_PORT"
        RAGFLOW_URL="http://$FALLBACK_RAGFLOW_HOST:$FALLBACK_RAGFLOW_PORT"
        return 0
    fi
    
    # 5. 所有方法都失败了
    echo "❌ 无法找到可用的RAGFlow服务"
    return 1
}

# 主流程
echo "1️⃣ 检查配置文件..."
if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "❌ 配置文件不存在: $CONFIG_FILE"
    exit 1
fi

# 读取并显示当前RAGFlow配置（使用Alpine兼容的命令）
current_ragflow_url=$(grep -A1 "ragflow:" $CONFIG_FILE | grep "base_url" | awk -F'"' '{print $2}' || echo "未找到")
echo "当前配置的RAGFlow URL: $current_ragflow_url"

echo "2️⃣ 网络环境检测与修复..."
if discover_ragflow; then
    echo "✅ 成功发现RAGFlow服务: $RAGFLOW_URL"
    
    # 如果配置文件中的URL与发现的不同，则更新
    if [[ "$current_ragflow_url" != "$RAGFLOW_URL" && -n "$current_ragflow_url" && -n "$RAGFLOW_URL" ]]; then
        update_config_url "$current_ragflow_url" "$RAGFLOW_URL"
    fi
else
    echo "⚠️ 无法发现RAGFlow服务，将使用原配置"
fi

echo "3️⃣ 启动WeChat-RAG服务..."
echo "===================================="
echo "启动时间: $(date)"
echo "配置文件: $CONFIG_FILE"
echo "===================================="

# 启动主程序
exec /app/wechat-rag
