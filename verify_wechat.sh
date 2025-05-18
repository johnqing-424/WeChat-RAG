#!/bin/bash

TOKEN="wechat_rag_token"
TIMESTAMP="$(date +%s)"
NONCE="nonce_$TIMESTAMP"
ECHO_STR="echostr_$TIMESTAMP"

# 排序
STR=("$TOKEN" "$TIMESTAMP" "$NONCE")
IFS=$'\n' SORTED_STR=($(sort <<<"${STR[*]}"))
unset IFS
SORTED_STR="${SORTED_STR[0]}${SORTED_STR[1]}${SORTED_STR[2]}"

# 计算签名
SIGNATURE=$(echo -n "$SORTED_STR" | sha1sum | cut -d ' ' -f1)

echo "Token: $TOKEN"
echo "Timestamp: $TIMESTAMP"
echo "Nonce: $NONCE"
echo "Echostr: $ECHO_STR"
echo "Signature: $SIGNATURE"
echo "URL: http://localhost/wechat?signature=$SIGNATURE&timestamp=$TIMESTAMP&nonce=$NONCE&echostr=$ECHO_STR"

# 发送验证请求
echo -e "\n测试验证请求..."
curl -s "http://localhost/wechat?signature=$SIGNATURE&timestamp=$TIMESTAMP&nonce=$NONCE&echostr=$ECHO_STR"
echo -e "\n"

# 测试错误的签名
WRONG_SIGNATURE="abcdef1234567890abcdef1234567890abcdef12"
echo "测试错误的签名..."
curl -s "http://localhost/wechat?signature=$WRONG_SIGNATURE&timestamp=$TIMESTAMP&nonce=$NONCE&echostr=$ECHO_STR"
echo -e "\n" 