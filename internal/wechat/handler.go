package wechat

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnqing-424/WeChat-RAG/internal/models"
	"github.com/johnqing-424/WeChat-RAG/internal/ragflow"
)

// WeChatMessage 微信用户发送的消息结构体
type WeChatMessage struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   string `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	MsgId        string `xml:"MsgId"`
}

// WeChatResponse 微信响应格式
type WeChatResponse struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
}

// VerifyWeChatToken 是用于验证微信服务器的 Token 回调
func VerifyWeChatToken(c *gin.Context) {
	echoStr := c.Query("echostr")
	c.String(http.StatusOK, echoStr)
}

func HandleWeChatMessage(c *gin.Context) {
	defer c.Request.Body.Close()
	body, _ := ioutil.ReadAll(c.Request.Body)

	var msg models.WeChatMessage
	err := xml.NewDecoder(bytes.NewReader(body)).Decode(&msg)
	if err != nil {
		c.XML(http.StatusOK, gin.H{"return_code": "FAIL"})
		return
	}

	fmt.Printf("用户 [%s] 提问: %s\n", msg.FromUserName, msg.Content)

	answer, err := ragflow.QueryRagFlow(msg.Content)
	if err != nil {
		answer = "抱歉，系统暂时无法回答您的问题。"
	}

	response := models.WeChatResponse{
		ToUserName:   msg.FromUserName,
		FromUserName: msg.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      answer,
	}

	c.XML(http.StatusOK, response)
}

// SendCustomerMessage 使用微信客服消息接口推送回复
type Text interface{}

type CustomerMessage struct {
	ToUser  string `json:"touser"`
	MsgType string `json:"msgtype"`
	Text    Text   `json:"text"`
}

func SendCustomerMessage(openid, content string) {
	token, err := GetAccessToken()
	if err != nil {
		fmt.Println("获取 access_token 失败:", err)
		return
	}

	type Text struct {
		Content string `json:"content"`
	}

	type CustomerMessage struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Text    Text   `json:"text"`
	}

	msg := CustomerMessage{
		ToUser:  openid,
		MsgType: "text",
		Text: Text{
			Content: content,
		},
	}

	body, _ := json.Marshal(msg)
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=%s ", token)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("推送失败:", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("微信返回结果:", string(respBody))
}
