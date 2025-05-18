#!/bin/bash

# 微信消息XML模板
XML_TEMPLATE='<xml>
  <ToUserName>gh_123456789abc</ToUserName>
  <FromUserName>test_user_789</FromUserName>
  <CreateTime>%s</CreateTime>
  <MsgType>text</MsgType>
  <Content>/status</Content>
  <MsgId>123456791</MsgId>
</xml>'

# 当前时间戳
TIMESTAMP=$(date +%s)

# 生成XML消息
XML_MSG=$(printf "$XML_TEMPLATE" "$TIMESTAMP")

echo "发送命令: /status 检查用户test_user_789的最后问题状态"
echo "------------------------"

# 发送请求到微信处理接口
curl -s -X POST "http://localhost/wechat" \
  -H "Content-Type: application/xml" \
  -d "$XML_MSG" | tee check_status_response.xml

echo -e "\n------------------------" 