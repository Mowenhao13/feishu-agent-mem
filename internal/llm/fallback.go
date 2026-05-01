// internal/llm/fallback.go

package llm

import (
	"strings"
)

// Fallback 降级策略（纯规则实现）
// 用途：LLM 不可用时使用
type Fallback struct {
	decisionKeywords []string
}

func NewFallback() *Fallback {
	return &Fallback{
		decisionKeywords: []string{
			"决定", "decided", "确认", "LGTM", "lgtm",
			"approve", "通过", "定下来", "结论", "最终方案",
		},
	}
}

// ExtractDecision 降级：关键词匹配提取决策
func (f *Fallback) ExtractDecision(content string) *ExtractionResult {
	result := &ExtractionResult{
		HasDecision:   false,
		Confidence:    0.0,
		ExtractedFrom: content,
	}

	for _, kw := range f.decisionKeywords {
		if strings.Contains(content, kw) {
			result.HasDecision = true
			result.Confidence = 0.7 // 降级置信度较低
			result.Decision = &DecisionExtract{
				Title:    "Auto-extracted decision",
				Decision: content,
				Proposer: "keyword_match",
			}
			break
		}
	}

	return result
}

// ClassifyTopic 降级：关键词匹配分类
func (f *Fallback) ClassifyTopic(content string, topics []string) *ClassificationResult {
	// 简单的关键词匹配
	for _, topic := range topics {
		if strings.Contains(content, topic) {
			return &ClassificationResult{
				Topic:      topic,
				Confidence: 0.6,
				Reasoning:  "keyword_match",
			}
		}
		// 特殊规则：数据库相关
		if topic == "数据库架构" && strings.Contains(content, "PostgreSQL") ||
			strings.Contains(content, "MySQL") || strings.Contains(content, "数据库") {
			return &ClassificationResult{
				Topic:      topic,
				Confidence: 0.6,
				Reasoning:  "keyword_match",
			}
		}
	}

	return &ClassificationResult{
		Topic:      "",
		Confidence: 0.0,
	}
}

// DetectCrossTopic 降级：简单规则
func (f *Fallback) DetectCrossTopic(node interface{}) *CrossTopicResult {
	// 简化实现
	return &CrossTopicResult{
		IsCrossTopic: false,
		Confidence:   0.5,
	}
}
