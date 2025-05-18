package models

// WeChatMessage 是接收微信消息的结构体
type WeChatMessage struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	MsgId        string `xml:"MsgId"` // 添加消息ID字段，用于处理重试
}

// WeChatResponse 是返回给微信的消息结构体
type WeChatResponse struct {
	ToUserName   string `xml:"ToUserName,cdata"`
	FromUserName string `xml:"FromUserName,cdata"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType,cdata"`
	Content      string `xml:"Content,cdata"`
}
