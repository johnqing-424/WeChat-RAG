package ragflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/johnqing-424/WeChat-RAG/internal/config"
)

// 全局配置变量
var (
	cfg            = config.GetConfig()
	RagFlowBaseURL = cfg.RagFlow.BaseURL
	ApiKey         = cfg.RagFlow.ApiKey
	ChatID         = cfg.RagFlow.ChatID
	DatasetID      = cfg.RagFlow.DatasetID
	MaxRetries     = cfg.RagFlow.MaxRetries
	RetryInterval  = time.Duration(cfg.RagFlow.RetryInterval) * time.Second
	RequestTimeout = time.Duration(cfg.RagFlow.RequestTimeout) * time.Second
)

type Chunk struct {
	Content      string `json:"content"`
	DocumentName string `json:"document_name"`
}

type RetrievalResponse struct {
	Code int     `json:"code"`
	Data []Chunk `json:"data"`
}

// RetrieveChunks 检索知识库中的 chunk
func RetrieveChunks(question string) ([]Chunk, error) {
	// 使用v1版本的检索API
	url := fmt.Sprintf("%s/api/v1/retrieval", RagFlowBaseURL)

	// 使用更通用的请求结构
	reqBody := map[string]interface{}{
		"question":    question,
		"dataset_ids": []string{DatasetID},
		"top_k":       5,
	}

	body, _ := json.Marshal(reqBody)

	// 发送POST请求
	respBody, err := makeHTTPRequestWithRetry("POST", url, body)
	if err != nil {
		fmt.Println("检索知识块失败:", err)
		return []Chunk{}, nil // 失败时返回空结果而不报错，以便继续后续处理
	}

	var result RetrievalResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return []Chunk{}, nil
	}

	return result.Data, nil
}

type CompletionRequest struct {
	Question  string `json:"question"`
	SessionID string `json:"session_id,omitempty"`
	Stream    bool   `json:"stream"`
}

type CompletionResponse struct {
	Code int `json:"code"`
	Data struct {
		Answer    string `json:"answer"`
		SessionID string `json:"session_id"`
	} `json:"data"`
}

// 会话缓存
var (
	sessionCache     = make(map[string]string) // 用户ID -> 会话ID的映射
	sessionCacheLock sync.RWMutex
)

// 确保会话存在
func EnsureSession(userID string) (string, error) {
	// 检查缓存中是否已有该用户的会话ID
	sessionCacheLock.RLock()
	sessionID, exists := sessionCache[userID]
	sessionCacheLock.RUnlock()

	// 如果已存在有效的会话ID，直接返回
	if exists && sessionID != "" {
		// 检查会话是否仍然有效（可以添加会话有效性检查）
		fmt.Println("使用缓存的会话ID:", sessionID, "用户ID:", userID)
		return sessionID, nil
	}

	// 如果不存在，创建新会话
	sessionName := "wechat_" + userID
	fmt.Println("创建新会话，名称:", sessionName)

	// 添加重试逻辑
	var newSessionID string
	var createErr error

	for retries := 0; retries <= MaxRetries; retries++ {
		if retries > 0 {
			time.Sleep(RetryInterval * time.Duration(retries))
			fmt.Printf("第%d次重试创建会话\n", retries)
		}

		// 创建新会话
		newSessionID, createErr = CreateSession(sessionName)
		if createErr == nil {
			break
		}

		fmt.Printf("创建会话失败(尝试%d/%d): %v\n", retries+1, MaxRetries+1, createErr)
		if retries == MaxRetries {
			return "", createErr
		}
	}

	// 保存到缓存
	sessionCacheLock.Lock()
	sessionCache[userID] = newSessionID
	sessionCacheLock.Unlock()

	fmt.Println("已缓存会话ID:", newSessionID, "用户ID:", userID)
	return newSessionID, nil
}

// 清理指定用户的会话缓存
func ClearSessionCache(userID string) {
	sessionCacheLock.Lock()
	defer sessionCacheLock.Unlock()

	delete(sessionCache, userID)
	fmt.Println("已清理用户会话缓存:", userID)
}

// QueryRagFlow 调用 RAGFlow 获取答案（基于知识库）
func QueryRagFlow(question, userID string) (string, error) {
	// 设置超时context
	ctx, cancel := context.WithTimeout(context.Background(), 140*time.Second) // 从35秒增加到140秒
	defer cancel()

	// 创建结果通道
	resultChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		// 确保会话存在
		sessionID, err := EnsureSession(userID)
		if err != nil {
			fmt.Println("创建会话失败:", err)
			errChan <- err
			return
		}

		// 使用正确的API路径
		url := fmt.Sprintf("%s/api/v1/chats/%s/completions", RagFlowBaseURL, ChatID)

		// 构造请求
		reqBody := map[string]interface{}{
			"question":   question,
			"session_id": sessionID,
			"stream":     false,
		}

		body, _ := json.Marshal(reqBody)

		// 发送请求并处理响应
		fmt.Println("发送RAGFlow查询，会话ID:", sessionID)

		// 添加重试逻辑
		var respBody []byte
		var reqErr error
		for retries := 0; retries <= MaxRetries; retries++ {
			if retries > 0 {
				// 如果是重试，等待一段时间
				time.Sleep(RetryInterval * time.Duration(retries))
				fmt.Printf("第%d次重试RAGFlow查询\n", retries)
			}

			respBody, reqErr = makeHTTPRequestWithRetry("POST", url, body)
			if reqErr == nil {
				break
			}

			fmt.Printf("RAGFlow查询失败(尝试%d/%d): %v\n", retries+1, MaxRetries+1, reqErr)
			if retries == MaxRetries {
				errChan <- reqErr
				return
			}
		}

		// 使用通用JSON解析以适应可能的不同响应结构
		var result map[string]interface{}
		if err := json.Unmarshal(respBody, &result); err != nil {
			fmt.Println("解析响应失败:", err)
			errChan <- err
			return
		}

		// 提取回答，处理不同可能的响应结构
		answer := extractAnswer(result)
		if answer == "" {
			fmt.Println("无法从响应中提取有效答案")
			errChan <- fmt.Errorf("无法提取答案")
			return
		}

		resultChan <- answer
	}()

	// 等待结果或超时
	select {
	case answer := <-resultChan:
		return answer, nil
	case err := <-errChan:
		// 返回错误信息
		return fmt.Sprintf("抱歉，我无法回答这个问题。系统错误: %v", err), err
	case <-ctx.Done():
		fmt.Println("RAGFlow查询超时")
		// 返回超时错误
		return "抱歉，响应超时。请稍后再试。", ctx.Err()
	}
}

// getLocalAnswer 不再使用预设答案，而是返回一个通用的错误响应
func getLocalAnswer(question string) string {
	return fmt.Sprintf("抱歉，我现在无法回答关于\"%s\"的问题。请稍后再试。", question)
}

// 从不同的响应结构中提取答案
func extractAnswer(result map[string]interface{}) string {
	// 提取RAGFlow对话API的标准响应格式
	if data, ok := result["data"].(map[string]interface{}); ok {
		if answer, ok := data["answer"].(string); ok && answer != "" {
			return answer
		}
	}

	// 尝试从OpenAI格式响应中提取答案
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok && content != "" {
					return content
				}
			}
		}
	}

	// 其他备用提取方法
	if data, ok := result["data"].(map[string]interface{}); ok {
		if content, ok := data["content"].(string); ok && content != "" {
			return content
		}
		if response, ok := data["response"].(string); ok && response != "" {
			return response
		}
	}

	// 直接从顶层字段提取
	if answer, ok := result["answer"].(string); ok && answer != "" {
		return answer
	}
	if content, ok := result["content"].(string); ok && content != "" {
		return content
	}

	return ""
}

// QueryLLMFreeAnswer 使用模型直接回答问题
func QueryLLMFreeAnswer(question string) (string, error) {
	// 使用OpenAI兼容API
	url := fmt.Sprintf("%s/api/v1/chats_openai/%s/chat/completions", RagFlowBaseURL, ChatID)

	// 按OpenAI格式构造请求
	reqBody := map[string]interface{}{
		"model": "model",
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": question,
			},
		},
		"stream": false,
	}

	body, _ := json.Marshal(reqBody)

	// 发送请求并处理响应
	respBody, err := makeHTTPRequestWithRetry("POST", url, body)
	if err != nil {
		fmt.Println("LLM查询失败:", err)
		return getDefaultAnswer(question), nil
	}

	// 使用通用JSON解析
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return getDefaultAnswer(question), nil
	}

	// 提取回答
	answer := extractAnswer(result)
	if answer == "" {
		return getDefaultAnswer(question), nil
	}

	return answer, nil
}

// 获取默认答案
func getDefaultAnswer(question string) string {
	return fmt.Sprintf("非常抱歉，我无法回答您的问题：%s。系统正在升级中，请稍后再试。", question)
}

// CreateSession 创建新的会话，如果会话已存在则返回现有会话ID
func CreateSession(sessionName string) (string, error) {
	// 使用正确的会话创建API路径
	url := fmt.Sprintf("%s/api/v1/chats/%s/sessions", RagFlowBaseURL, ChatID)

	reqBody := map[string]interface{}{
		"name": sessionName,
	}

	body, _ := json.Marshal(reqBody)

	// 发送请求并处理响应
	respBody, err := makeHTTPRequestWithRetry("POST", url, body)
	if err != nil {
		fmt.Println("创建会话请求失败:", err)
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	// 检查API响应
	if code, ok := result["code"].(float64); ok && code != 0 {
		message := "未知错误"
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return "", fmt.Errorf("API错误: %s (代码: %.0f)", message, code)
	}

	// 获取会话ID
	var sessionID string
	if data, ok := result["data"].(map[string]interface{}); ok {
		if id, ok := data["id"].(string); ok {
			sessionID = id
			fmt.Println("成功创建/获取会话ID:", sessionID)
		}
	}

	return sessionID, nil
}

// makeHTTPRequestWithRetry 带重试机制的HTTP请求
func makeHTTPRequestWithRetry(method, url string, body []byte) ([]byte, error) {
	var lastErr error

	for i := 0; i <= MaxRetries; i++ {
		// 如果不是第一次请求，等待一段时间后重试
		if i > 0 {
			time.Sleep(RetryInterval)
			fmt.Printf("重试第%d次请求: %s\n", i, url)
		}

		// 创建HTTP请求
		req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Authorization", "Bearer "+ApiKey)
		req.Header.Set("Content-Type", "application/json")

		// 创建带超时的HTTP客户端
		client := &http.Client{
			Timeout: RequestTimeout,
		}

		// 记录请求详情
		fmt.Printf("发送%s请求到: %s\n请求体: %s\n", method, url, string(body))

		// 发送请求
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			fmt.Printf("HTTP请求失败: %v\n", err)
			continue
		}

		// 确保响应体被关闭
		defer resp.Body.Close()

		// 读取响应体
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		// 记录响应详情
		fmt.Printf("收到响应: 状态码=%d\n响应体: %s\n", resp.StatusCode, string(respBody))

		// 如果状态码是405，尝试切换HTTP方法
		if resp.StatusCode == http.StatusMethodNotAllowed && i < MaxRetries {
			fmt.Printf("收到405错误，尝试切换HTTP方法\n")
			if method == "POST" {
				method = "GET"
			} else {
				method = "POST"
			}
			continue
		}

		// 检查HTTP状态码，但只记录而不失败，以确保能尽可能返回响应
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("警告: HTTP状态码错误: %d\n", resp.StatusCode)
		}

		// 请求成功
		return respBody, nil
	}

	// 所有重试都失败了
	return nil, fmt.Errorf("在%d次尝试后请求失败: %v", MaxRetries+1, lastErr)
}
