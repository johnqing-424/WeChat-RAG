package ragflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	RagFlowBaseURL = "http://your-ragflow-server/api/v1"
	ApiKey         = "your-api-key-here"
	ChatID         = "your-chat-id"
)

type RagFlowRequest struct {
	Question  string `json:"question"`
	SessionID string `json:"session_id"`
	Stream    bool   `json:"stream"`
}

type RagFlowResponse struct {
	Code int `json:"code"`
	Data struct {
		Answer    string `json:"answer"`
		Reference struct {
			Chunks []struct {
				DocumentName string `json:"document_name"`
			} `json:"chunks"`
		} `json:"reference"`
		SessionID string `json:"session_id"`
	} `json:"data"`
}

// QueryRagFlow 调用 RAGFlow 获取答案
func QueryRagFlow(question string) (string, error) {
	reqBody := RagFlowRequest{
		Question:  question,
		SessionID: "", // 可以动态管理
		Stream:    false,
	}

	body, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/chats/%s/completions", RagFlowBaseURL, ChatID)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+ApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)

	var result RagFlowResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("RAGFlow 返回错误码 %d", result.Code)
	}

	return result.Data.Answer, nil
}
