package larkadapter

import (
	"os"
)

// Config 飞书适配器配置
type Config struct {
	// AppID 飞书应用 ID
	AppID string
	// AppSecret 飞书应用密钥
	AppSecret string
	// ChatIDs 监控的群聊 ID 列表
	ChatIDs []string
}

// LoadConfig 从环境变量加载配置
func LoadConfig() *Config {
	cfg := &Config{
		AppID:     os.Getenv("LARK_APP_ID"),
		AppSecret: os.Getenv("LARK_APP_SECRET"),
	}

	// 从 LARK_CHAT_IDS 环境变量读取群聊 ID（逗号分隔）
	if chatIDs := os.Getenv("LARK_CHAT_IDS"); chatIDs != "" {
		// 在实际使用时需要解析逗号分隔的字符串
		// 这里先保留原始字符串，具体解析逻辑在各适配器中实现
		cfg.ChatIDs = []string{chatIDs}
	}

	return cfg
}

// CheckLarkConfig 检查飞书配置是否完整
func CheckLarkConfig() (missing []string) {
	if os.Getenv("LARK_APP_ID") == "" {
		missing = append(missing, "LARK_APP_ID")
	}
	if os.Getenv("LARK_APP_SECRET") == "" {
		missing = append(missing, "LARK_APP_SECRET")
	}
	return missing
}
