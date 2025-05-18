package wechat

import (
	"bytes"
	"crypto/sha1"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnqing-424/WeChat-RAG/internal/models"
	"github.com/johnqing-424/WeChat-RAG/internal/ragflow"
)

// 消息缓存
var (
	// 用户ID -> 答案数据 (用于用户会话)
	userAnswerCache = make(map[string]*answerData)
	// 消息ID -> 答案数据 (用于处理微信重试)
	messageCacheMap = make(map[string]*answerData)
	cacheLock       sync.RWMutex
)

// 答案缓存数据结构
type answerData struct {
	UserID        string    // 用户ID
	LastQuestion  string    // 最后一个问题
	Answer        string    // 准备好的答案
	IsReady       bool      // 答案是否准备好
	ProcessingMsg string    // 处理中的消息
	CreatedAt     time.Time // 创建时间
}

// VerifyWeChatToken 是用于验证微信服务器的 Token 回调
func VerifyWeChatToken(c *gin.Context) {
	signature := c.Query("signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	echostr := c.Query("echostr")

	// 1. 将token、timestamp、nonce三个参数进行字典序排序
	strs := []string{Token, timestamp, nonce}
	sort.Strings(strs)

	// 2. 将三个参数字符串拼接成一个字符串进行sha1加密
	str := strings.Join(strs, "")
	h := sha1.New()
	h.Write([]byte(str))
	sum := fmt.Sprintf("%x", h.Sum(nil))

	// 3. 开发者获得加密后的字符串可与signature对比，标识该请求来源于微信
	if sum == signature {
		c.String(http.StatusOK, echostr)
	} else {
		c.String(http.StatusOK, "验证失败")
	}
}

// 创建符合微信格式的XML响应
func createWeChatXMLResponse(fromUser, toUser, content string) string {
	timestamp := time.Now().Unix()

	// 微信标准XML格式，必须使用<xml>作为根元素
	xmlFormat := `<xml>
<ToUserName><![CDATA[%s]]></ToUserName>
<FromUserName><![CDATA[%s]]></FromUserName>
<CreateTime>%d</CreateTime>
<MsgType><![CDATA[text]]></MsgType>
<Content><![CDATA[%s]]></Content>
</xml>`

	// 清理内容中可能包含的特殊标记
	cleanedContent := cleanAnswer(content)

	return fmt.Sprintf(xmlFormat, toUser, fromUser, timestamp, cleanedContent)
}

// HandleWeChatMessage 处理用户发送的消息
func HandleWeChatMessage(c *gin.Context) {
	defer c.Request.Body.Close()
	body, _ := ioutil.ReadAll(c.Request.Body)

	var msg models.WeChatMessage
	err := xml.NewDecoder(bytes.NewReader(body)).Decode(&msg)
	if err != nil {
		fmt.Println("XML 解析失败:", err)
		c.String(http.StatusOK, createWeChatXMLResponse(msg.ToUserName, msg.FromUserName, "消息解析失败"))
		return
	}

	userID := msg.FromUserName
	msgID := msg.MsgId // 消息ID用于重试识别
	fmt.Printf("用户 [%s] 提问: %s, MsgId: %s\n", userID, msg.Content, msgID)

	// 检查是否是指令消息
	if strings.HasPrefix(msg.Content, "/") {
		handleCommandMessage(c, msg)
		return
	}

	// 检查缓存中是否有该消息ID的处理记录（用于处理微信重试）
	cacheLock.RLock()
	msgData, msgExists := messageCacheMap[msgID]
	cacheLock.RUnlock()

	// 如果消息已处理过，直接使用处理结果
	if msgExists {
		if msgData.IsReady {
			// 已有答案，直接返回
			fmt.Println("返回已处理消息的答案:", msgData.Answer)
			xmlResponse := createWeChatXMLResponse(msg.ToUserName, msg.FromUserName, msgData.Answer)
			c.String(http.StatusOK, xmlResponse)
			return
		}
		// 正在处理，返回处理中的消息
		fmt.Println("返回处理中消息:", msgData.ProcessingMsg)
		xmlResponse := createWeChatXMLResponse(msg.ToUserName, msg.FromUserName, msgData.ProcessingMsg)
		c.String(http.StatusOK, xmlResponse)
		return
	}

	// 新消息，加入缓存
	processingMsg := "您的问题正在处理中，可稍后发送 /status 查询结果。"
	newMsgData := &answerData{
		UserID:        userID,
		LastQuestion:  msg.Content,
		ProcessingMsg: processingMsg,
		CreatedAt:     time.Now(),
	}

	cacheLock.Lock()
	messageCacheMap[msgID] = newMsgData
	// 更新用户最近一条消息缓存，用于status查询
	userAnswerCache[userID] = newMsgData
	cacheLock.Unlock()

	// 创建超时响应通道
	answerChan := make(chan string, 1)
	timeoutChan := time.After(4 * time.Second) // 4秒超时，微信要求5秒内回复

	// 异步获取答案
	go func() {
		// 获取完整答案
		answer, err := getAnswerForQuestion(msg.Content, userID)
		if err == nil && answer != "" {
			answerChan <- answer
		}
	}()

	// 等待答案或超时
	select {
	case answer := <-answerChan:
		// 更新缓存
		cacheLock.Lock()
		if md, ok := messageCacheMap[msgID]; ok {
			md.Answer = answer
			md.IsReady = true
			messageCacheMap[msgID] = md
		}
		cacheLock.Unlock()

		// 直接返回答案
		fmt.Println("直接返回答案:", answer)
		xmlResponse := createWeChatXMLResponse(msg.ToUserName, msg.FromUserName, answer)
		c.String(http.StatusOK, xmlResponse)
	case <-timeoutChan:
		// 超时，返回正在处理的消息
		fmt.Println("超时，返回处理中消息")
		xmlResponse := createWeChatXMLResponse(msg.ToUserName, msg.FromUserName, processingMsg)
		c.String(http.StatusOK, xmlResponse)

		// 继续异步处理问题，结果将存入缓存
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("异步处理问题时发生错误:", r)
					cacheLock.Lock()
					if md, ok := messageCacheMap[msgID]; ok {
						md.Answer = "抱歉，处理您的问题时发生了错误，请稍后再试。"
						md.IsReady = true
						messageCacheMap[msgID] = md
					}
					cacheLock.Unlock()
				}
			}()

			// 获取完整答案
			answer, err := getAnswerForQuestion(msg.Content, userID)
			if err != nil {
				answer = fmt.Sprintf("抱歉，获取答案失败: %v", err)
			}

			// 更新缓存
			cacheLock.Lock()
			if md, ok := messageCacheMap[msgID]; ok {
				md.Answer = answer
				md.IsReady = true
				messageCacheMap[msgID] = md
			}
			cacheLock.Unlock()
			fmt.Println("已准备好回答(存入缓存):", answer)
		}()
	}
}

// 快速获取预设答案，用于首次尝试在微信超时前返回
func getQuickAnswerForQuestion(question, userID string) (string, error) {
	// 注释掉预设回答，强制使用RAGFlow
	/*
		if strings.Contains(question, "公司") {
			return "浙江腾视科技有限公司是中国本土领先的生成式AI算力模组及边缘智算AGI普惠落地解决方案提供商。公司成立于2017年，核心团队来自华为、中兴等知名企业。", nil
		} else if strings.Contains(question, "业务") {
			return "公司主要业务包括提供机器人控制全栈AI边缘智算大脑、基于主流AI芯片的边缘算力模组、\"感知-决策-控制\"一体化边缘智算平台以及自主研发的AI加速引擎。", nil
		} else if strings.Contains(question, "团队") {
			return "公司核心团队成员来自华为、中兴等知名企业，创始人李泽湘担任技术总顾问，谢兮煜担任技术总顾问兼创始人/CEO。", nil
		} else if strings.Contains(question, "产品") {
			return "公司产品包括边缘算力模组和边缘计算终端，能够提供从1到500 Top的算力范围，形成了丰富的产品线。", nil
		}
	*/

	// 不匹配任何预设关键词，返回空表示需要通过RagFlow获取答案
	return "", fmt.Errorf("无预设答案")
}

// 清理RAGFlow回答中的特殊标记
func cleanAnswer(answer string) string {
	// 只去除不必要的标记，保留原文引用标记##$$
	specialMarks := []string{"CITATIONS:", "CITATIONS: "}
	for _, mark := range specialMarks {
		answer = strings.Replace(answer, mark, "", -1)
	}

	// 去除末尾可能的空格和换行符
	answer = strings.TrimSpace(answer)

	return answer
}

// 获取问题的回答
func getAnswerForQuestion(question, userID string) (string, error) {
	fmt.Println("开始获取完整答案，问题:", question)

	// 检查问题中是否包含关键词 - 现在完全注释掉这部分代码，强制使用RAGFlow
	/*
		if strings.Contains(question, "公司") {
			return "浙江腾视科技有限公司是中国本土领先的生成式AI算力模组及边缘智算AGI普惠落地解决方案提供商。公司成立于2017年，核心团队来自华为、中兴等知名企业。", nil
		} else if strings.Contains(question, "业务") {
			return "公司主要业务包括提供机器人控制全栈AI边缘智算大脑、基于主流AI芯片的边缘算力模组、\"感知-决策-控制\"一体化边缘智算平台以及自主研发的AI加速引擎。", nil
		} else if strings.Contains(question, "团队") {
			return "公司核心团队成员来自华为、中兴等知名企业，创始人李泽湘担任技术总顾问，谢兮煜担任技术总顾问兼创始人/CEO。", nil
		} else if strings.Contains(question, "产品") {
			return "公司产品包括边缘算力模组和边缘计算终端，能够提供从1到500 Top的算力范围，形成了丰富的产品线。", nil
		}
	*/

	// 获取知识库答案
	chunks, err := ragflow.RetrieveChunks(question)
	if err != nil {
		fmt.Println("检索知识块失败:", err)
	} else {
		fmt.Printf("检索到 %d 个知识块\n", len(chunks))
		for i, chunk := range chunks {
			if i < 2 { // 只打印前两个，避免日志过长
				if len(chunk.Content) > 0 {
					fmt.Printf("知识块 %d: %s\n", i+1, chunk.Content[:min(50, len(chunk.Content))])
				}
			}
		}
	}

	// 调用 RAGFlow API 获取答案
	fmt.Println("开始调用 RAGFlow API")
	answer, err := ragflow.QueryRagFlow(question, userID)
	if err != nil {
		fmt.Println("RAGFlow查询失败:", err)
		return fmt.Sprintf("抱歉，系统暂时无法回答您的问题\"%s\"，请稍后再试或者尝试其他问题。错误: %v", question, err), nil
	}

	// 清理RAGFlow返回的答案
	cleanedAnswer := cleanAnswer(answer)
	fmt.Println("RAGFlow返回答案:", cleanedAnswer)
	return cleanedAnswer, nil
}

// min函数帮助截断日志输出
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 处理指令消息
func handleCommandMessage(c *gin.Context, msg models.WeChatMessage) {
	var content string

	switch msg.Content {
	case "/help":
		content = "欢迎使用RAG智能问答系统！\n\n您可以直接发送问题与系统对话，系统会尝试从知识库中寻找答案或使用AI回答。\n\n可用命令：\n/help - 显示帮助信息\n/清空 - 清空会话历史\n/重置 - 重置会话\n/status - 查询上一个问题的处理状态"
	case "/清空":
		// 清除该用户的缓存
		cacheLock.Lock()
		delete(userAnswerCache, msg.FromUserName)
		cacheLock.Unlock()
		content = "您的会话历史已清空，开始新的对话。"
	case "/重置":
		// 清除该用户的缓存和会话
		cacheLock.Lock()
		delete(userAnswerCache, msg.FromUserName)
		cacheLock.Unlock()
		ragflow.ClearSessionCache(msg.FromUserName)
		content = "系统已重置，开始新的对话。"
	case "/status":
		// 检查用户最近一条消息的处理状态
		cacheLock.RLock()
		userData, exists := userAnswerCache[msg.FromUserName]
		cacheLock.RUnlock()

		if !exists {
			content = "没有找到您的历史消息记录。"
		} else if userData.IsReady {
			// 确保返回的答案也经过清理
			content = "您的上一个问题已处理完成，答案是：\n\n" + cleanAnswer(userData.Answer)
		} else {
			content = "您的问题 \"" + userData.LastQuestion + "\" 仍在处理中，请稍候再查询。"
		}
	default:
		content = "未识别的指令，您可以直接发送问题来获取回答。可用指令：/help、/清空、/重置、/status"
	}

	// 直接使用自定义XML格式
	xmlResponse := createWeChatXMLResponse(msg.ToUserName, msg.FromUserName, content)
	c.String(http.StatusOK, xmlResponse)
}
