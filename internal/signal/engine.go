package signal

import (
	"log"
	"strings"

	larkadapter "feishu-mem/internal/lark-adapter"
	"feishu-mem/internal/decision"
)

// MemoryGraphInterface 内存图接口
type MemoryGraphInterface interface {
	GetAllDecisions() []*decision.DecisionNode
	UpsertDecision(node *decision.DecisionNode, project string)
}

// PipelineInterface 流水线接口
type PipelineInterface interface {
	ApplyMutation(mut *DecisionMutation) error
}

// SignalActivationEngine 信号激活引擎编排器
type SignalActivationEngine struct {
	Emitters     map[AdapterType]StateChangeEmitter
	Router       *ActivationRouter
	Assembler    *ContextAssembler
	StateMachine *DecisionStateMachine
	Patterns     *PatternMatcher

	Pipeline PipelineInterface
	Memory   MemoryGraphInterface
}

// NewSignalActivationEngine 创建信号激活引擎
func NewSignalActivationEngine(
	pipeline PipelineInterface,
	memory MemoryGraphInterface,
) *SignalActivationEngine {
	return &SignalActivationEngine{
		Emitters:     NewEmitters(),
		Router:       NewActivationRouter(),
		Assembler:    NewContextAssembler(),
		StateMachine: NewDecisionStateMachine(),
		Patterns:     NewPatternMatcher(),
		Pipeline:     pipeline,
		Memory:       memory,
	}
}

// OnDetectResult 处理检测器返回的结果 — 每条决策消息生成独立决策
func (e *SignalActivationEngine) OnDetectResult(
	adapter AdapterType,
	result *larkadapter.DetectResult,
) (*ProcessingReport, error) {
	var allMutations []*DecisionMutation
	totalTokenUsage := 0

	for _, change := range result.Changes {
		// 只处理文本类型的消息
		if change.Type != "new_text" && change.Type != "new_post" {
			continue
		}

		// 检查消息是否包含决策关键词
		matchedKeywords := matchDecisionKeywords(change.Summary)
		if len(matchedKeywords) == 0 {
			continue
		}

		// 为每条决策消息创建独立信号
		sig := NewSignal(adapter, change.Summary)
		sig.Strength = StrengthMedium
		sig.Context.ContentSnippet = change.Summary
		sig.Context.Keywords = matchedKeywords
		sig.Context.DecisionSignals = matchedKeywords

		// 创建独立决策节点
		proposer := extractSenderFromSummary(change.Summary)
		newNode := decision.NewDecisionNode(
			GenerateSDRID(),
			"Auto-extracted decision",
			"feishu-mem",
			"general",
		)
		newNode.Status = decision.StatusPending
		newNode.Decision = change.Summary
		newNode.Proposer = proposer
		newNode.Executor = extractExecutorFromSummary(change.Summary)
		newNode.ImpactLevel = extractImpactFromSummary(change.Summary)

		// 创建 mutation
		mut := e.StateMachine.CreateMutationForNewDecision(newNode, sig)
		allMutations = append(allMutations, mut)

		// 应用 mutation
		if e.Pipeline != nil {
			if err := e.Pipeline.ApplyMutation(mut); err != nil {
				log.Printf("[SignalEngine] ApplyMutation failed for %s: %v", newNode.SDRID, err)
			}
		}

		totalTokenUsage += 200 // 估算
	}

	if len(allMutations) == 0 {
		return nil, nil
	}

	return &ProcessingReport{
		SignalID:      GenerateSDRID(),
		Mutations:     allMutations,
		Conflicts:     []Conflict{},
		TokenUsage:    totalTokenUsage,
		DecisionCount: len(allMutations),
	}, nil
}

// matchDecisionKeywords 匹配消息中的决策关键词
func matchDecisionKeywords(text string) []string {
	decisionWords := []string{
		"决定", "decided", "确认", "LGTM", "lgtm",
		"approve", "通过", "定下来", "就这么办", "confirmed",
	}
	var matched []string
	for _, w := range decisionWords {
		if strings.Contains(text, w) {
			matched = append(matched, w)
		}
	}
	return matched
}

// extractSenderFromSummary 从消息摘要中提取发送者名称
// 摘要格式: "[群聊] 莫文豪: 决定..."
func extractSenderFromSummary(summary string) string {
	if idx := strings.Index(summary, "] "); idx >= 0 {
		rest := summary[idx+2:]
		if colonIdx := strings.Index(rest, ": "); colonIdx >= 0 {
			return rest[:colonIdx]
		}
	}
	return ""
}

// extractExecutorFromSummary 从消息中提取执行人
func extractExecutorFromSummary(text string) string {
	executorLabels := []string{"执行人"}
	for _, label := range executorLabels {
		if idx := strings.Index(text, label); idx >= 0 {
			rest := text[idx+len(label):]
			// 提取执行人后的名字（到逗号、句号或结尾）
			endIdx := len(rest)
			for i, c := range rest {
				if c == ',' || c == '，' || c == '。' || c == ' ' {
					endIdx = i
					break
				}
			}
			return strings.TrimSpace(rest[:endIdx])
		}
	}
	return ""
}

// extractImpactFromSummary 从消息中提取影响级别
func extractImpactFromSummary(text string) decision.ImpactLevel {
	if strings.Contains(text, "critical") || strings.Contains(text, "关键") {
		return decision.ImpactCritical
	}
	if strings.Contains(text, "major") || strings.Contains(text, "重要") {
		return decision.ImpactMajor
	}
	if strings.Contains(text, "minor") || strings.Contains(text, "轻微") {
		return decision.ImpactMinor
	}
	return decision.ImpactAdvisory
}

// PatternMatcher 模式匹配器
type PatternMatcher struct{}

func NewPatternMatcher() *PatternMatcher {
	return &PatternMatcher{}
}
