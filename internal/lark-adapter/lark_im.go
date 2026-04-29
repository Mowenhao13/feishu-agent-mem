package larkadapter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// IMExtractor 群聊消息提取器
type IMExtractor struct {
	config *Config
	cli    *LarkCLI
}

// NewIMExtractor 创建群聊提取器
func NewIMExtractor(cfg *Config) *IMExtractor {
	return &IMExtractor{
		config: cfg,
		cli:    NewLarkCLI(),
	}
}

// Name 实现接口
func (e *IMExtractor) Name() string {
	return "lark_im"
}

// Detect 检测消息变化：群聊 + P2P 双人会话
// 使用 +chat-messages-list 获取消息后按时间过滤（无需额外 scope）
func (e *IMExtractor) Detect(lastCheck time.Time) (*DetectResult, error) {
	changes := []Change{}
	cutoff := lastCheck.Unix()

	// 首次检测跳过，不把已有数据报为新增
	if lastCheck.IsZero() {
		result := &DetectResult{
			Source:     e.Name(),
			HasChanges: false,
			DetectedAt: time.Now(),
			LastCheck:  lastCheck,
		}
		_ = SaveDetectResult(result)
		return result, nil
	}

	// 1. 检测群聊消息
	if len(e.config.ChatIDs) > 0 {
		chatID := e.config.ChatIDs[0]
		if strings.Contains(chatID, ",") {
			chatID = strings.Split(chatID, ",")[0]
		}

		// 获取最新消息并在本地按时间过滤
		items, err := e.getGroupMessageItems(chatID)
		if err == nil {
			for _, item := range items {
				ts := extractTimestamp(item)
				if ts > cutoff {
					mid, _ := item["message_id"].(string)
					body := extractBody(item)
					senderName := extractSender(item)
					changes = append(changes, Change{
						Type:       "new",
						EntityType: "group_message",
						EntityID:   mid,
						Summary:    fmt.Sprintf("[群聊] %s: %s", senderName, truncateContent(body, 60)),
						Timestamp:  ts,
					})
				}
			}
		}
	}

	// 2. 检测 P2P 双人会话
	p2pItems, err := e.getP2PMessageItems()
	if err == nil {
		for _, item := range p2pItems {
			ts := extractTimestamp(item)
			if ts > cutoff {
				mid, _ := item["message_id"].(string)
				body := extractBody(item)
				senderName := extractSender(item)
				changes = append(changes, Change{
					Type:       "new",
					EntityType: "p2p_message",
					EntityID:   mid,
					Summary:    fmt.Sprintf("[双人会话] %s: %s", senderName, truncateContent(body, 60)),
					Timestamp:  ts,
				})
			}
		}
	}

	result := &DetectResult{
		Source:     e.Name(),
		HasChanges: len(changes) > 0,
		DetectedAt: time.Now(),
		LastCheck:  lastCheck,
		Changes:    changes,
	}

	_ = SaveDetectResult(result)
	return result, nil
}

// Extract 提取全量数据
func (e *IMExtractor) Extract() error {
	rawData := make(map[string]any)
	errors := make(map[string]string)

	// P2P 会话消息
	if items, err := e.getP2PMessageItems(); err == nil {
		rawData["p2p_messages"] = items
	} else {
		errors["p2p_messages"] = err.Error()
	}

	// 群聊消息 & pin
	if len(e.config.ChatIDs) > 0 {
		chatID := e.config.ChatIDs[0]
		if strings.Contains(chatID, ",") {
			chatID = strings.Split(chatID, ",")[0]
		}

		if items, err := e.getGroupMessageItems(chatID); err == nil {
			rawData["chat_messages"] = items
		} else {
			errors["chat_messages"] = err.Error()
		}

		if pins, err := e.getPinMessages(chatID); err == nil {
			rawData["pin_messages"] = pins
		} else {
			errors["pin_messages"] = err.Error()
		}
	}

	if len(errors) > 0 {
		rawData["_errors"] = errors
	}

	result := &ExtractionResult{
		Source:      e.Name(),
		ExtractedAt: time.Now(),
		RawData:     rawData,
		Formatted:   map[string]any{"extracted": true},
	}

	if err := SaveToJSON(e.Name(), result); err != nil {
		return fmt.Errorf("save result failed: %w", err)
	}

	return nil
}

// ========== P2P 双人会话 ==========

// getP2PMessageItems 获取 P2P 双人会话的消息列表
func (e *IMExtractor) getP2PMessageItems() ([]map[string]any, error) {
	// 尝试用 +chat-messages-list --user-id 获取 P2P 消息
	// 如果 P2P 会话不存在，返回 nil 不报错
	output, err := e.cli.RunCommand(
		"im", "+chat-messages-list",
		"--user-id", "ou_2e2874a35309566a79a8116d505b1d46",
	)
	if err != nil {
		return nil, err
	}
	return parseChatMessageList(output)
}

// ========== 群聊 ==========

// getGroupMessageItems 获取群聊消息列表
func (e *IMExtractor) getGroupMessageItems(chatID string) ([]map[string]any, error) {
	output, err := e.cli.RunCommand(
		"im", "+chat-messages-list",
		"--chat-id", chatID,
		"--page-all",
	)
	if err != nil {
		output, err = e.cli.RunCommand(
			"im", "+chat-messages-list",
			"--chat-id", chatID,
		)
		if err != nil {
			return nil, err
		}
	}
	return parseChatMessageList(output)
}

// ========== Pin 消息 ==========

func (e *IMExtractor) getPinMessages(chatID string) ([]any, error) {
	output, err := e.cli.RunCommand(
		"im", "pins", "list",
		"--params", fmt.Sprintf(`{"chat_id": "%s"}`, chatID),
	)
	if err != nil {
		return nil, err
	}

	var result []any
	if err := json.Unmarshal(output, &result); err != nil {
		var single any
		if err := json.Unmarshal(output, &single); err != nil {
			return nil, err
		}
		result = []any{single}
	}
	return result, nil
}

// ========== 解析工具 ==========

// parseChatMessageList 解析 +chat-messages-list 的输出为统一的消息列表
func parseChatMessageList(output []byte) ([]map[string]any, error) {
	var list []any
	if err := json.Unmarshal(output, &list); err != nil {
		var single any
		if err := json.Unmarshal(output, &single); err != nil {
			return nil, err
		}
		list = []any{single}
	}

	var items []map[string]any
	for _, outer := range list {
		outerMap, ok := outer.(map[string]any)
		if !ok {
			continue
		}
		data, ok := outerMap["data"].(map[string]any)
		if !ok {
			continue
		}
		// +chat-messages-list 返回 messages 字段
		rawList, ok := data["messages"].([]any)
		if !ok {
			continue
		}
		for _, raw := range rawList {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			items = append(items, item)
		}
	}
	return items, nil
}

// extractTimestamp 从消息中提取时间戳（尝试多个字段）
func extractTimestamp(item map[string]any) int64 {
	// 优先用 create_time 字段
	if ct, ok := item["create_time"].(string); ok {
		if ts := parseMessageTime(ct); ts > 0 {
			return ts
		}
	}
	// 尝试 msg_time
	if mt, ok := item["msg_time"].(string); ok {
		if ts := parseMessageTime(mt); ts > 0 {
			return ts
		}
	}
	return 0
}

// extractBody 从消息中提取文本内容
func extractBody(item map[string]any) string {
	// 先尝试直接 content 字段
	if content, ok := item["content"].(string); ok && content != "" {
		// 飞书文本消息的 content 可能是 JSON 格式：{"text":"xxx"}
		if strings.HasPrefix(content, "{") {
			var obj map[string]any
			if err := json.Unmarshal([]byte(content), &obj); err == nil {
				if text, ok := obj["text"].(string); ok {
					return text
				}
			}
		}
		return content
	}
	// 尝试 body.content
	if body, ok := item["body"].(map[string]any); ok {
		if content, ok := body["content"].(string); ok {
			return content
		}
	}
	return ""
}

// extractSender 从消息中提取发送者名称
func extractSender(item map[string]any) string {
	if sender, ok := item["sender"].(map[string]any); ok {
		if name, ok := sender["name"].(string); ok {
			return name
		}
		if id, ok := sender["id"].(string); ok {
			return id
		}
	}
	return "unknown"
}

func truncateContent(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func parseMessageTime(timeStr string) int64 {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05-07:00",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, timeStr); err == nil {
			return t.Unix()
		}
	}
	return 0
}
