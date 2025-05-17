package main

import (
	"github.com/gin-gonic/gin"
	"github.com/johnqing-424/WeChat-RAG/internal/wechat"
)

func main() {
	r := gin.Default()

	// 微信 Token 验证（GET 请求）
	r.GET("/wechat", wechat.VerifyWeChatToken)

	// 接收用户消息（POST 请求）
	r.POST("/wechat", wechat.HandleWeChatMessage)

	// 启动 Gin Web 服务
	if err := r.Run(":8000"); err != nil {
		panic(err)
	}
}
