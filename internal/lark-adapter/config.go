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

// LoadConfig 从环境变量加载配置（默认使用 LARK_*）
func LoadConfig() *Config {
	return LoadConfigWithPrefix("LARK_")
}

// LoadConfigWithPrefix 从环境变量加载配置（使用指定前缀）
func LoadConfigWithPrefix(prefix string) *Config {
	cfg := &Config{
		AppID:     os.Getenv(prefix + "APP_ID"),
		AppSecret: os.Getenv(prefix + "APP_SECRET"),
	}

	// 解析 CHAT_IDS（逗号分隔）
	if raw := os.Getenv(prefix + "CHAT_IDS"); raw != "" {
		for _, id := range strings.Split(raw, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				cfg.ChatIDs = append(cfg.ChatIDs, id)
			}
		}
	}

	cfg.UserID = os.Getenv(prefix + "USER_ID")

	return cfg
}

// FirstChatID 返回第一个 chat-id（常用场景），不存在则返回空串
func (c *Config) FirstChatID() string {
	if len(c.ChatIDs) > 0 {
		return c.ChatIDs[0]
	}
	return ""
}

// IsConfigured 检查配置是否完整（至少有 AppID 和 AppSecret）
func (c *Config) IsConfigured() bool {
	return c.AppID != "" && c.AppSecret != ""
}
