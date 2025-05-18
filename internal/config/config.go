package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v2"
)

// Config 是配置的根结构体
type Config struct {
	WeChat  WeChatConfig  `yaml:"wechat"`
	RagFlow RagFlowConfig `yaml:"ragflow"`
	Server  ServerConfig  `yaml:"server"`
}

// WeChatConfig 包含微信相关配置
type WeChatConfig struct {
	AppID     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
	Token     string `yaml:"token"`
	TokenURL  string `yaml:"token_url"`
}

// RagFlowConfig 包含RAGFlow服务相关配置
type RagFlowConfig struct {
	BaseURL        string `yaml:"base_url"`
	ApiKey         string `yaml:"api_key"`
	ChatID         string `yaml:"chat_id"`
	DatasetID      string `yaml:"dataset_id"`
	MaxRetries     int    `yaml:"max_retries"`
	RetryInterval  int    `yaml:"retry_interval"`
	RequestTimeout int    `yaml:"request_timeout"`
}

// ServerConfig 包含服务器相关配置
type ServerConfig struct {
	Port int `yaml:"port"`
}

var (
	config     *Config
	configOnce sync.Once
)

// GetConfig 返回配置单例
func GetConfig() *Config {
	configOnce.Do(func() {
		config = &Config{}
		err := loadConfig(config)
		if err != nil {
			log.Printf("加载配置文件失败: %v，将使用默认值\n", err)
			setDefaultConfig(config)
		}
	})

	return config
}

// loadConfig 从配置文件加载配置
func loadConfig(cfg *Config) error {
	// 尝试从多个位置查找配置文件
	configPaths := []string{
		"config.yml",    // 当前目录
		"../config.yml", // 上级目录
		filepath.Join(os.Getenv("HOME"), "config.yml"), // 用户主目录
	}

	var configData []byte
	var err error

	// 尝试从各个路径读取配置
	for _, path := range configPaths {
		configData, err = ioutil.ReadFile(path)
		if err == nil {
			log.Printf("从 %s 加载配置\n", path)
			break
		}
	}

	if err != nil {
		return fmt.Errorf("无法找到配置文件: %w", err)
	}

	// 解析YAML配置
	err = yaml.Unmarshal(configData, cfg)
	if err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return nil
}

// setDefaultConfig 设置默认配置
func setDefaultConfig(cfg *Config) {
	// 默认微信配置
	cfg.WeChat = WeChatConfig{
		AppID:     "wx39fc841a05350758",
		AppSecret: "8280c222717449b5147b5cd9db7bbcda",
		Token:     "wechat_rag_token",
		TokenURL:  "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
	}

	// 默认RAGFlow配置
	cfg.RagFlow = RagFlowConfig{
		BaseURL:        "http://ragflow-server",
		ApiKey:         "ragflow-cwNWNkZGFhMzMxNzExZjA5MmM5MDI0Mj",
		ChatID:         "5e48a6dc331a11f0af0302420aff0606",
		DatasetID:      "7b214898331711f09ded02420aff0606",
		MaxRetries:     2,
		RetryInterval:  1,
		RequestTimeout: 120,
	}

	// 默认服务器配置
	cfg.Server = ServerConfig{
		Port: 80,
	}
}
