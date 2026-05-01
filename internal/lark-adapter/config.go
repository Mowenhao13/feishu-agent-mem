package larkadapter

import (
	"os"
	"strings"
)

// Config 飞书适配器配置
type Config struct {
	AppID     string
	AppSecret string
	ChatIDs   []string
	UserID    string
}

// LoadConfig 从环境变量加载配置
func LoadConfig() *Config {
	cfg := &Config{
		AppID:     os.Getenv("LARK_APP_ID"),
		AppSecret: os.Getenv("LARK_APP_SECRET"),
	}

	// 解析 LARK_CHAT_IDS（逗号分隔）
	if raw := os.Getenv("LARK_CHAT_IDS"); raw != "" {
		for _, id := range strings.Split(raw, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				cfg.ChatIDs = append(cfg.ChatIDs, id)
			}
		}
	}

	cfg.UserID = os.Getenv("LARK_USER_ID")

	return cfg
}

// FirstChatID 返回第一个 chat-id（常用场景），不存在则返回空串
func (c *Config) FirstChatID() string {
	if len(c.ChatIDs) > 0 {
		return c.ChatIDs[0]
	}
	return ""
}
