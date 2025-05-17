package wechat

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

const (
	TokenURL  = "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s "
	AppID     = "wx39fc841a05350758"               // 替换为你的 AppID
	AppSecret = "8280c222717449b5147b5cd9db7bbcda" // 替换为你的 AppSecret
)

var (
	accessToken string
	expireTime  time.Time
	mutex       sync.RWMutex
)

// GetAccessToken 获取当前有效的 access_token，如果过期则自动刷新
func GetAccessToken() (string, error) {
	mutex.RLock()
	if time.Now().Before(expireTime) {
		defer mutex.RUnlock()
		return accessToken, nil
	}
	mutex.RUnlock()

	return refreshToken()
}

// refreshToken 刷新 access_token
func refreshToken() (string, error) {
	url := fmt.Sprintf(TokenURL, AppID, AppSecret)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("获取 access_token 失败: %d - %s", result.ErrCode, result.ErrMsg)
	}

	mutex.Lock()
	accessToken = result.AccessToken
	expireTime = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	mutex.Unlock()

	return accessToken, nil
}
