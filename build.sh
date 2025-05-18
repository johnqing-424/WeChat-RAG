#!/bin/bash

# 设置颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}开始一键部署RAGFlow与WeChat-RAG...${NC}"

# 1. 设置vm.max_map_count并持久化
echo -e "${GREEN}设置vm.max_map_count...${NC}"
current_max_map=$(sysctl vm.max_map_count | awk '{print $3}')
if [ "$current_max_map" -lt 262144 ]; then
  echo -e "${GREEN}设置vm.max_map_count=262144${NC}"
  sudo sysctl -w vm.max_map_count=262144
  # 持久化设置
  if ! grep -q "vm.max_map_count" /etc/sysctl.conf; then
    echo "vm.max_map_count=262144" | sudo tee -a /etc/sysctl.conf
  else
    sudo sed -i 's/vm.max_map_count=.*/vm.max_map_count=262144/' /etc/sysctl.conf
  fi
fi

# 2. 克隆RAGFlow仓库
echo -e "${GREEN}克隆RAGFlow仓库...${NC}"
if [ ! -d "ragflow" ]; then
  git clone https://github.com/infiniflow/ragflow.git
  cd ragflow
  git checkout -f v0.18.0
  cd ..
else
  echo -e "${YELLOW}RAGFlow仓库已存在，跳过克隆步骤${NC}"
  cd ragflow
  git checkout -f v0.18.0
  cd ..
fi

# 3. 修改RAGFlow的.env文件
echo -e "${GREEN}配置RAGFlow环境变量...${NC}"
cd ragflow/docker
if [ -f ".env" ]; then
  # 修改已有的.env文件
  sed -i 's/RAGFLOW_IMAGE=infiniflow\/ragflow:.*$/RAGFLOW_IMAGE=infiniflow\/ragflow:v0.18.0/' .env
  sed -i 's/DOC_ENGINE=.*$/DOC_ENGINE=infinity/' .env
else
  # 创建新的.env文件
  cat > .env << EOF
RAGFLOW_IMAGE=infiniflow/ragflow:v0.18.0
COMPOSE_PROJECT_NAME=ragflow
DOC_ENGINE=infinity
EOF
fi

# 4. 修改docker-compose.yml中的端口映射
echo -e "${GREEN}修改RAGFlow端口映射...${NC}"
# 使用sed修改端口映射
sed -i 's/80:80/8081:80/' docker-compose.yml
sed -i 's/443:443/8443:443/' docker-compose.yml

# 5. 启动RAGFlow服务
echo -e "${GREEN}启动RAGFlow服务...${NC}"
docker compose -f docker-compose.yml up -d
cd ../..

# 6. 等待RAGFlow服务启动完成
echo -e "${GREEN}等待RAGFlow服务启动...${NC}"
attempt=0
max_attempts=30
until $(curl --output /dev/null --silent --head --fail http://localhost:8081); do
  if [ ${attempt} -eq ${max_attempts} ]; then
    echo -e "${RED}RAGFlow服务启动超时，请检查日志${NC}"
    exit 1
  fi
  printf '.'
  attempt=$(($attempt+1))
  sleep 5
done
echo -e "${GREEN}RAGFlow服务已启动${NC}"

# 7. 交互式收集WeChat-RAG配置信息
echo -e "${GREEN}配置WeChat-RAG...${NC}"

# 创建临时目录存放配置文件
mkdir -p wechat-rag-config

# 交互式收集配置信息
echo -e "${YELLOW}请输入RAGFlow API密钥 (如果现在没有，可以先输入占位符后续修改)：${NC}"
read ragflow_api_key

echo -e "${YELLOW}请输入微信公众号Token：${NC}"
read wechat_token

echo -e "${YELLOW}请输入微信公众号AppID：${NC}"
read wechat_app_id

echo -e "${YELLOW}请输入微信公众号AppSecret：${NC}"
read wechat_app_secret

echo -e "${YELLOW}请输入微信公众号EncodingAESKey：${NC}"
read wechat_encoding_aes_key

# 创建config.yml文件
cat > wechat-rag-config/config.yml << EOF
ragflow:
  base_url: "http://ragflow-server:80"
  api_key: "${ragflow_api_key}"
  timeout: 10
  retry_count: 3

wechat:
  token: "${wechat_token}"
  appID: "${wechat_app_id}"
  appSecret: "${wechat_app_secret}"
  encoding_aes_key: "${wechat_encoding_aes_key}"
  response_timeout: 4500
EOF

# 8. 拉取WeChat-RAG镜像并启动容器
echo -e "${GREEN}拉取WeChat-RAG镜像...${NC}"
docker pull johnqing424/wechat-rag:latest

echo -e "${GREEN}启动WeChat-RAG容器...${NC}"
docker run -d \
  --name wechat-rag \
  --restart always \
  --network ragflow_default \
  -p 80:80 \
  -v $(pwd)/wechat-rag-config/config.yml:/app/config.yml \
  -v $(pwd)/wechat-rag-logs:/app/logs \
  crpi-ljtue4qaq06l5wcn.cn-hangzhou.personal.cr.aliyuncs.com/tensortec_johnqing/wechat-arg:latest

echo -e "${GREEN}部署完成！${NC}"
echo -e "${YELLOW}重要提示：${NC}"
echo "1. RAGFlow管理界面：http://localhost:8081"
echo "2. WeChat-RAG服务运行在80端口，符合微信公众号要求"
echo "3. 如果您需要更新配置，请修改：$(pwd)/wechat-rag-config/config.yml"
echo "4. 修改配置后请重启容器：docker restart wechat-rag"
echo "5. WeChat-RAG日志位于：$(pwd)/wechat-rag-logs"
echo "6. 如需访问RAGFlow容器日志请使用：docker logs ragflow-server"