package processors

import (
	"fmt"
)

// DecisionIdentifier 决策识别器（openclaw-architecture.md §2.1.1 Stage 2a）
// 从原始数据检测决策信号，判断是否包含决策内容
type DecisionIdentifier struct {
	keywords []string
}

// NewDecisionIdentifier 创建决策识别器
func NewDecisionIdentifier() *DecisionIdentifier {
	return &DecisionIdentifier{
		keywords: []string{
			"决定", "确认", "结论", "通过", "定下来", "就这么办",
			"approve", "LGTM", "decided", "confirmed",
		},
	}
}

// Identify 判断原始数据是否包含决策信号
func (di *DecisionIdentifier) Identify(raw interface{}) (bool, float64) {
	text := fmt.Sprintf("%v", raw)
	if text == "" {
		return false, 0
	}

	matchCount := 0
	for _, kw := range di.keywords {
		if contains(text, kw) {
			matchCount++
		}
	}

	if matchCount == 0 {
		return false, 0
	}

	confidence := float64(matchCount) / float64(len(di.keywords))
	if confidence > 1.0 {
		confidence = 1.0
	}

	return true, confidence
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
