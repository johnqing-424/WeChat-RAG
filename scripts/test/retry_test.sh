#!/bin/bash

# 微信消息XML模板
XML_TEMPLATE='<xml>
  <ToUserName>gh_123456789abc</ToUserName>
  <FromUserName>test_user_789</FromUserName>
  <CreateTime>%s</CreateTime>
  <MsgType>text</MsgType>
  <Content>如何提高工作效率？</Content>
  <MsgId>987654321</MsgId>
</xml>'

# 当前时间戳
TIMESTAMP=$(date +%s)

# 生成XML消息
XML_MSG=$(printf "$XML_TEMPLATE" "$TIMESTAMP")

echo "发送问题: 如何提高工作效率？ (MsgId: 987654321)"
echo "------------------------"
echo "第一次请求 (应该返回处理中消息)..."

# 发送第一次请求
curl -s -X POST "http://localhost/wechat" \
  -H "Content-Type: application/xml" \
  -d "$XML_MSG" | tee retry_response1.xml

echo -e "\n------------------------"
echo "等待3秒..."
sleep 3

echo "第二次请求 (模拟微信重试，应该返回处理中或完成的消息)..."
# 发送第二次请求（相同MsgId，模拟微信重试）
curl -s -X POST "http://localhost/wechat" \
  -H "Content-Type: application/xml" \
  -d "$XML_MSG" | tee retry_response2.xml

echo -e "\n------------------------" 