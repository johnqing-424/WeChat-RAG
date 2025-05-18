package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/johnqing-424/WeChat-RAG/internal/config"
	"github.com/johnqing-424/WeChat-RAG/internal/wechat"
)

func main() {
	// 获取配置
	cfg := config.GetConfig()

	r := gin.Default()

	// 微信 Token 验证（GET 请求）
	r.GET("/wechat", wechat.VerifyWeChatToken)

	// 接收用户消息（POST 请求）
	r.POST("/wechat", wechat.HandleWeChatMessage)

	// 启动 Gin Web 服务
	portAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := r.Run(portAddr); err != nil {
		panic(err)
	}
}
