package extractors

import (
	"context"

	"github.com/openclaw/internal/adapters"
	"github.com/openclaw/internal/core"
)

// IMExtractor 群聊消息提取器（decision-extraction.md §4.1）
type IMExtractor struct {
	cli *adapters.LarkCli
}

// NewIMExtractor 创建 IM 提取器
func NewIMExtractor(cli *adapters.LarkCli) *IMExtractor {
	return &IMExtractor{cli: cli}
}

// Extract 从群聊消息中提取决策候选
func (e *IMExtractor) Extract(ctx context.Context, chatID string, startTime, endTime int64) ([]*core.DecisionNode, error) {
	messages, err := e.cli.ListMessages(ctx, chatID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	var candidates []*core.DecisionNode
	for _, msg := range messages {
		// 检测决策关键词
		if containsDecisionKeyword(msg.Body) {
			candidates = append(candidates, &core.DecisionNode{
				SDRID:    core.GenerateSDRID(len(candidates) + 1),
				Title:    truncateString(msg.Body, 80),
				Decision: msg.Body,
				Status:   core.StatusPending,
				FeishuLinks: core.FeishuLinks{
					RelatedMessageIDs: []string{msg.MessageID},
					RelatedChatIDs:    []string{msg.ChatID},
				},
			})
		}
	}

	return candidates, nil
}

// PinExtractor Pin 消息提取器
type PinExtractor struct {
	cli *adapters.LarkCli
}

func NewPinExtractor(cli *adapters.LarkCli) *PinExtractor {
	return &PinExtractor{cli: cli}
}

func (e *PinExtractor) Extract(ctx context.Context, chatID string) ([]*core.DecisionNode, error) {
	pins, err := e.cli.GetPinMessages(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var candidates []*core.DecisionNode
	for _, pin := range pins {
		// Pin 消息自动标记为高价值决策候选
		candidates = append(candidates, &core.DecisionNode{
			SDRID:    core.GenerateSDRID(len(candidates) + 1),
			Title:    "[Pin] " + truncateString(pin.Body, 60),
			Decision: pin.Body,
			Status:   core.StatusPending,
			FeishuLinks: core.FeishuLinks{
				RelatedMessageIDs: []string{pin.MessageID},
				RelatedChatIDs:    []string{pin.ChatID},
			},
		})
	}

	return candidates, nil
}

func containsDecisionKeyword(text string) bool {
	keywords := []string{"决定", "确认", "结论", "通过", "定下来", "approve", "LGTM", "decided"}
	for _, kw := range keywords {
		if contains(text, kw) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if s[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
