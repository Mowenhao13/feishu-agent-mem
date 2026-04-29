package signal

import (
	"time"

	"github.com/openclaw/internal/core"
)

// StateChangeSignal 状态变化信号协议（signal-activation-engine.md §1）
type StateChangeSignal struct {
	SignalID  string           `json:"signal_id"` // UUID，用于去重
	Adapter   core.AdapterType `json:"adapter"`   // IM/VC/Docs/Calendar/Task/OKR/Contact/Wiki
	Timestamp time.Time        `json:"timestamp"`
	EventType string           `json:"event_type"` // 原始飞书事件类型

	ChangeType    core.ChangeType `json:"change_type"`    // created | updated | deleted | status_change
	ChangeSummary string          `json:"change_summary"` // 人类可读摘要

	PrimaryID  string   `json:"primary_id"`  // 变更的实体 ID
	RelatedIDs []string `json:"related_ids"` // 引用的其他实体

	Context  SignalContext       `json:"context"`
	Strength core.SignalStrength `json:"strength"` // strong | medium | weak
}

// SignalContext 信号上下文（signal-activation-engine.md §1）
type SignalContext struct {
	Keywords        []string      `json:"keywords"`
	DecisionSignals []string      `json:"decision_signals"` // 匹配到的决策信号词
	MentionedIDs    []string      `json:"mentioned_ids"`    // @提及的用户
	ActorID         string        `json:"actor_id"`         // 触发变更的人
	ParticipantIDs  []string      `json:"participant_ids"`  // 参会人/协作者
	EmbeddedURLs    []EmbeddedURL `json:"embedded_urls"`    // 解析出的 URL
	ContentSnippet  string        `json:"content_snippet"`  // 截断的内容片段
	EventTime       time.Time     `json:"event_time"`
}

// EmbeddedURL 嵌入式 URL
type EmbeddedURL struct {
	URL            string `json:"url"`
	ExtractedToken string `json:"extracted_token,omitempty"`
	DocType        string `json:"doc_type,omitempty"`
}

// SignalStrength 信号强度（从 core 复用，这里保留冗余以明确依赖）

// DecisionKeywords 决策关键词列表（openclaw-architecture.md §4.2）
var DecisionKeywords = []string{
	"决定", "确认", "结论", "通过", "定下来", "就这么办",
	"approve", "LGTM", "decided", "confirmed", "agreed",
	"就这么定了", "不再讨论", "最终方案",
}

// ContainsDecisionKeyword 检查文本是否包含决策关键词
func ContainsDecisionKeyword(text string) (bool, string) {
	for _, kw := range DecisionKeywords {
		// 简单包含检测
		if contains(text, kw) {
			return true, kw
		}
	}
	return false, ""
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

// searchSubstring 简单子串搜索
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
