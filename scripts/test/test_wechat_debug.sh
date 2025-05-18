#!/bin/bash

# 微信消息XML模板 - 确保严格符合微信格式
XML_TEMPLATE='<xml>
<ToUserName><![CDATA[gh_123456789abc]]></ToUserName>
<FromUserName><![CDATA[test_user_123]]></FromUserName>
<CreateTime>%s</CreateTime>
<MsgType><![CDATA[text]]></MsgType>
<Content><![CDATA[介绍一下公司]]></Content>
<MsgId>123456789</MsgId>
</xml>'

# 当前时间戳
TIMESTAMP=$(date +%s)

# 生成XML消息
XML_MSG=$(printf "$XML_TEMPLATE" "$TIMESTAMP")

# 显示请求内容
echo "===== 发送请求 ====="
echo "$XML_MSG"
echo "===================="

# 发送请求至本地服务（使用详细输出）
echo "===== 详细请求/响应信息 ====="
curl -v -X POST "http://localhost/wechat" \
  -H "Content-Type: application/xml" \
  -H "User-Agent: Mozilla/5.0 (Linux) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36 MicroMessenger/7.0.9.501 NetType/WIFI MiniProgramEnv/Windows WindowsWechat" \
  -d "$XML_MSG" > wechat_debug_response.xml

echo -e "\n===== 响应内容 ====="
cat wechat_debug_response.xml
echo -e "\n===================="

# 测试验证请求
echo -e "\n===== 测试Token验证接口 ====="
./verify_wechat.sh
echo -e "\n====================" 