# WeChat-RAG

基于RAGFlow的微信公众号智能问答系统，支持知识库检索与智能回复。

## 目录

- [项目简介](#项目简介)
- [系统架构](#系统架构)
- [配置说明](#配置说明)
- [部署方法](#部署方法)
  - [本地源码部署](#本地源码部署)
  - [Docker部署](#docker部署)
- [故障排除](#故障排除)

## 项目简介

WeChat-RAG 是一个连接微信公众号与RAGFlow的智能问答系统，支持：
- 微信公众号消息接收与回复
- 基于RAGFlow的知识库检索
- 智能回答生成
- 会话管理与缓存
- 自动网络环境检测与修复

## 系统架构

本系统由以下几个主要组件构成：
- 微信接口模块：处理微信公众号的消息接收与回复
- RAGFlow客户端：与RAGFlow服务通信，进行知识库检索和问答
- 配置管理：使用YAML配置文件进行统一配置
- 网络环境自修复：自动检测并修复网络连接问题

## 配置说明

系统使用 `config.yml` 文件进行配置，主要包括以下几个部分：

```yaml
# WeChat配置
wechat:
  app_id: "wx39fc841a05350758" 
  app_secret: "8280c222717449b5147b5cd9db7bbcda"
  token: "wechat_rag_token"
  token_url: "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s"

# RAGFlow服务配置
ragflow:
  base_url: "http://ragflow-server"
  api_key: "ragflow-cwNWNkZGFhMzMxNzExZjA5MmM5MDI0Mj"
  chat_id: "5e48a6dc331a11f0af0302420aff0606"
  dataset_id: "7b214898331711f09ded02420aff0606"
  max_retries: 2
  retry_interval: 1 # 秒
  request_timeout: 120 # 秒

# 服务器配置
server:
  port: 80
```

请根据实际情况修改以上配置参数：
- `wechat`: 微信公众号相关配置，包括AppID、AppSecret等
- `ragflow`: RAGFlow服务配置，包括服务URL、API密钥、聊天ID等
- `server`: 服务器配置，包括监听端口等

## 部署方法

### 本地源码部署

#### 1. 安装依赖

确保已安装Go 1.20或更高版本。

```bash
# 检查Go版本
go version

# 克隆仓库
git clone https://github.com/johnqing-424/WeChat-RAG.git
cd WeChat-RAG

# 安装依赖
go mod download
```

#### 2. 配置系统

编辑 `config.yml` 文件，根据实际情况修改配置参数：

```bash
# 编辑配置文件
vim config.yml
```

确保配置了正确的RAGFlow服务地址和微信公众号信息。

#### 3. 构建项目

```bash
# 构建项目
go build -o WeChat-RAG cmd/main.go
```

#### 4. 运行服务

```bash
# 运行服务
./WeChat-RAG
```

服务将在配置的端口上启动（默认为80端口）。

#### 5. 配置微信公众号

在微信公众号后台设置服务器地址为：
```
http://你的服务器IP/wechat
```

Token设置为与配置文件中一致（默认为 `wechat_rag_token`）。

### Docker部署

#### 1. 准备配置文件

与本地部署相同，编辑 `config.yml` 文件，根据实际情况修改配置参数。

#### 2. 构建Docker镜像

```bash
# 进入docker目录
cd docker

# 构建镜像
docker build -t wechat-rag:latest -f Dockerfile ..
```

#### 3. 启动容器

```bash
# 启动容器，连接到RAGFlow网络
docker run -d --name wechat-rag-container -p 80:80 --network=docker_ragflow wechat-rag:latest
```

> **注意**：请确保 `docker_ragflow` 是RAGFlow服务所在的网络名称，如有不同请修改为实际网络名称。

#### 4. 查看容器日志

```bash
# 查看容器日志
docker logs wechat-rag-container
```

#### 5. 管理容器

```bash
# 停止容器
docker stop wechat-rag-container

# 重新启动容器
docker start wechat-rag-container

# 删除容器
docker rm wechat-rag-container
```

## 故障排除

### 1. RAGFlow连接问题

系统启动时会自动检测RAGFlow服务的连接状态，支持多种连接方式：

- Docker内部网络连接（默认）
- 环境变量指定的地址
- 回退地址

如果自动检测失败，可以：
- 检查RAGFlow服务是否正常运行
- 确认网络连接是否正常
- 修改config.yml中的RAGFlow服务地址

### 2. 微信验证失败

如遇微信验证失败，请检查：
- 配置文件中的Token是否与微信公众号后台设置一致
- 服务器是否正常监听80端口
- 服务器防火墙是否允许80端口的HTTP请求

### 3. 消息超时

微信公众号要求5秒内必须响应，本系统实现了异步处理机制，用户可以：
- 发送 `/status` 查询之前问题的处理状态
- 系统会自动缓存问题的答案，当用户重新提问时立即返回

### 4. Docker网络问题

在Docker环境中，确保：
- WeChat-RAG容器与RAGFlow容器在同一网络
- 使用正确的网络名称启动容器
- 容器间可以通过容器名称相互访问