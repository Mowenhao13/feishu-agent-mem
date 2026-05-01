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

// IMSender 群聊消息发送器
type IMSender struct {
	config *Config
	cli    *LarkCLI
}

// NewIMSender 创建消息发送器
func NewIMSender(cfg *Config) *IMSender {
	return &IMSender{
		config: cfg,
		cli:    NewLarkCLI(),
	}
}

// SendTextMessage 发送文本消息到指定 chat_id
func (s *IMSender) SendTextMessage(chatID, text string) (string, error) {
	args := []string{"im", "+messages-send", "--chat-id", chatID, "--text", text, "--as", "bot"}
	output, err := s.cli.RunCommand(args...)
	if err != nil {
		return "", fmt.Errorf("send text message failed: %w", err)
	}
	return string(output), nil
}

// SendTextMessageToAll 发送文本消息到所有配置的 chat_id
func (s *IMSender) SendTextMessageToAll(text string) ([]string, error) {
	var results []string
	for _, chatID := range s.config.ChatIDs {
		result, err := s.SendTextMessage(chatID, text)
		if err != nil {
			results = append(results, fmt.Sprintf("chat %s: error: %v", chatID, err))
		} else {
			results = append(results, fmt.Sprintf("chat %s: success: %s", chatID, result))
		}
	}
	return results, nil
}

// Name 实现接口
func (e *IMExtractor) Name() string {
	return "lark_im"
}

// Detect 检测消息变化：群聊 + P2P
func (e *IMExtractor) Detect(lastCheck time.Time) (*DetectResult, error) {
	var changes []Change

	// 首次检测：拉取最近 1 小时的消息作为基线
	if lastCheck.IsZero() {
		lastCheck = time.Now().Add(-1 * time.Hour)
	}

	cutoff := lastCheck.Unix()

	// 1. 检测群聊消息
	for _, chatID := range e.config.ChatIDs {
		id := chatID
		if strings.Contains(id, ",") {
			id = strings.Split(id, ",")[0]
		}

		items, err := e.getGroupMessageItems(id, lastCheck)
		if err != nil {
			continue
		}

		for _, item := range items {
			ts := extractTimestamp(item)
			if ts <= cutoff {
				continue
			}
			mid, _ := item["message_id"].(string)
			body := extractBody(item)
			senderName := extractSender(item)
			msgType, _ := item["msg_type"].(string)
			changes = append(changes, e.classifyMessageChange("group_message", mid, msgType, senderName, body, ts))
		}
	}

	// 2. P2P 双人会话
	if e.config.UserID != "" {
		p2pItems, err := e.getP2PMessageItems(lastCheck)
		if err == nil {
			for _, item := range p2pItems {
				ts := extractTimestamp(item)
				if ts <= cutoff {
					continue
				}
				mid, _ := item["message_id"].(string)
				body := extractBody(item)
				senderName := extractSender(item)
				msgType, _ := item["msg_type"].(string)
				changes = append(changes, e.classifyMessageChange("p2p_message", mid, msgType, senderName, body, ts))
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

// classifyMessageChange 根据消息类型分类变化
func (e *IMExtractor) classifyMessageChange(entityType, entityID, msgType, senderName, body string, timestamp int64) Change {
	var changeType, summary string

	switch msgType {
	case "text":
		changeType = "new_text"
		summary = fmt.Sprintf("[%s] %s: %s", entityTypeToLabel(entityType), senderName, truncateContent(body, 60))
	case "image":
		changeType = "new_image"
		summary = fmt.Sprintf("[%s] %s 发送了图片", entityTypeToLabel(entityType), senderName)
	case "file":
		changeType = "new_file"
		summary = fmt.Sprintf("[%s] %s 发送了文件", entityTypeToLabel(entityType), senderName)
	case "audio":
		changeType = "new_audio"
		summary = fmt.Sprintf("[%s] %s 发送了语音", entityTypeToLabel(entityType), senderName)
	case "video":
		changeType = "new_video"
		summary = fmt.Sprintf("[%s] %s 发送了视频", entityTypeToLabel(entityType), senderName)
	case "sticker":
		changeType = "new_sticker"
		summary = fmt.Sprintf("[%s] %s 发送了贴纸", entityTypeToLabel(entityType), senderName)
	case "post":
		changeType = "new_post"
		summary = fmt.Sprintf("[%s] %s 发送了富文本消息", entityTypeToLabel(entityType), senderName)
	case "interactive":
		changeType = "new_card"
		summary = fmt.Sprintf("[%s] %s 发送了卡片消息", entityTypeToLabel(entityType), senderName)
	case "share_chat":
		changeType = "new_share_chat"
		summary = fmt.Sprintf("[%s] %s 分享了群聊", entityTypeToLabel(entityType), senderName)
	case "share_user":
		changeType = "new_share_user"
		summary = fmt.Sprintf("[%s] %s 分享了名片", entityTypeToLabel(entityType), senderName)
	case "merge_forward":
		changeType = "new_merge_forward"
		summary = fmt.Sprintf("[%s] %s 转发了合并消息", entityTypeToLabel(entityType), senderName)
	default:
		changeType = "new"
		summary = fmt.Sprintf("[%s] %s: %s", entityTypeToLabel(entityType), senderName, truncateContent(body, 60))
	}

	return Change{
		Type:       changeType,
		EntityType: entityType,
		EntityID:   entityID,
		Summary:    summary,
		Timestamp:  timestamp,
	}
}

func entityTypeToLabel(entityType string) string {
	if entityType == "group_message" {
		return "群聊"
	}
	return "双人会话"
}

// Extract 提取全量数据
func (e *IMExtractor) Extract() error {
	rawData := make(map[string]any)

	for _, chatID := range e.config.ChatIDs {
		items, err := e.getGroupMessageItems(chatID, time.Time{})
		if err == nil {
			rawData["chat_messages"] = items
		}
	}

	if e.config.UserID != "" {
		items, err := e.getP2PMessageItems(time.Time{})
		if err == nil {
			rawData["p2p_messages"] = items
		}
	}

	result := &ExtractionResult{
		Source:      e.Name(),
		ExtractedAt: time.Now(),
		RawData:     rawData,
		Formatted:   map[string]any{"extracted": true},
	}
	return SaveToJSON(e.Name(), result)
}

// ========== P2P 双人会话 ==========

func (e *IMExtractor) getP2PMessageItems(lastCheck time.Time) ([]map[string]any, error) {
	args := []string{"im", "+chat-messages-list", "--user-id", e.config.UserID}
	if !lastCheck.IsZero() {
		args = append(args, "--start", lastCheck.Format(time.RFC3339))
	}
	output, err := e.cli.RunCommand(args...)
	if err != nil {
		return nil, err
	}
	return parseChatMessageList(output)
}

// ========== 群聊 ==========

func (e *IMExtractor) getGroupMessageItems(chatID string, lastCheck time.Time) ([]map[string]any, error) {
	args := []string{"im", "+chat-messages-list", "--chat-id", chatID}
	if !lastCheck.IsZero() {
		args = append(args, "--start", lastCheck.Format(time.RFC3339))
	}
	output, err := e.cli.RunCommand(args...)
	if err != nil {
		args2 := []string{"im", "+chat-messages-list", "--chat-id", chatID}
		if !lastCheck.IsZero() {
			args2 = append(args2, "--start", lastCheck.Format(time.RFC3339))
		}
		output, err = e.cli.RunCommand(args2...)
		if err != nil {
			return nil, err
		}
	}
	return parseChatMessageList(output)
}

// ========== 解析工具 ==========

func parseChatMessageList(output []byte) ([]map[string]any, error) {
	// lark-cli +chat-messages-list 直接返回数组格式
	var list []map[string]any
	if err := json.Unmarshal(output, &list); err == nil {
		return list, nil
	}

	// 兜底：可能是 {data: {messages: [...]}} 格式
	var wrapper map[string]any
	if err := json.Unmarshal(output, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse chat-messages-list output: %s", err)
	}
	if data, ok := wrapper["data"].(map[string]any); ok {
		if msgs, ok := data["messages"].([]any); ok {
			var result []map[string]any
			for _, m := range msgs {
				if mm, ok := m.(map[string]any); ok {
					result = append(result, mm)
				}
			}
			return result, nil
		}
	}
	return nil, fmt.Errorf("could not parse chat-messages-list output")
}

func extractTimestamp(item map[string]any) int64 {
	if ct, ok := item["create_time"].(string); ok {
		if ts := parseMessageTime(ct); ts > 0 {
			return ts
		}
	}
	if mt, ok := item["msg_time"].(string); ok {
		if ts := parseMessageTime(mt); ts > 0 {
			return ts
		}
	}
	return 0
}

func extractBody(item map[string]any) string {
	if content, ok := item["content"].(string); ok && content != "" {
		// 飞书文本消息 content 可能是 JSON: {"text":"xxx"}
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
	if body, ok := item["body"].(map[string]any); ok {
		if content, ok := body["content"].(string); ok {
			return content
		}
	}
	return ""
}

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
