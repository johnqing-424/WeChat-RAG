#!/bin/bash

# 微信消息XML模板
XML_TEMPLATE='<xml>
  <ToUserName>gh_123456789abc</ToUserName>
  <FromUserName>test_user_456</FromUserName>
  <CreateTime>%s</CreateTime>
  <MsgType>text</MsgType>
  <Content>%s</Content>
  <MsgId>123456789</MsgId>
</xml>'

# 当前时间戳
TIMESTAMP=$(date +%s)

# 测试问题 - 使用"介绍一下业务"
QUESTION="介绍一下业务"

# 生成XML消息
XML_MSG=$(printf "$XML_TEMPLATE" "$TIMESTAMP" "$QUESTION")

echo "发送问题: $QUESTION"
echo "------------------------"

# 发送请求到微信处理接口
curl -s -X POST "http://localhost/wechat" \
  -H "Content-Type: application/xml" \
  -d "$XML_MSG" | tee response2.xml

echo -e "\n------------------------"
echo "响应已保存到response2.xml" 