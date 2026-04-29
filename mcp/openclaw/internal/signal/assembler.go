package signal

import (
	"math"
	"sort"

	"github.com/openclaw/internal/core"
)

// ContextAssembler 上下文装配器（signal-activation-engine.md §3）
type ContextAssembler struct {
	memory *core.MemoryGraph
}

// AssembledContext 装配后的上下文
type AssembledContext struct {
	SignalID         string
	Signal           *StateChangeSignal
	Decisions        []*core.DecisionNode
	CorrelationHints []CorrelationHint
	TokenUsed        int
}

// CorrelationHint 关联提示（signal-activation-engine.md §3.4）
type CorrelationHint struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
	SourceSDR   string  `json:"source_sdr,omitempty"`
}

// NewContextAssembler 创建上下文装配器
func NewContextAssembler(mg *core.MemoryGraph) *ContextAssembler {
	return &ContextAssembler{memory: mg}
}

// Assemble 装配上下文（signal-activation-engine.md §3.3）
func (ca *ContextAssembler) Assemble(signal *StateChangeSignal, plan *ContextQueryPlan) *AssembledContext {
	ctx := &AssembledContext{
		SignalID: signal.SignalID,
		Signal:   signal,
	}

	// 1. 从 MemoryGraph 查询相关决策
	topicDecisions := ca.queryRelatedDecisions(signal)

	// 2. 对决策排序（按相关度评分）
	scored := scoreDecisions(topicDecisions, signal)

	// 3. 贪心选择：优先填入高分决策，直到预算耗尽
	budget := plan.TotalTokenBudget
	tokensUsed := 0

	for _, s := range scored {
		tokenCost := estimateTokenCost(s.decision)
		if tokensUsed+tokenCost > budget {
			break
		}
		ctx.Decisions = append(ctx.Decisions, s.decision)
		tokensUsed += tokenCost
	}

	ctx.TokenUsed = tokensUsed

	// 4. 计算关联提示
	ctx.CorrelationHints = ca.computeHints(signal, ctx.Decisions)

	return ctx
}

// queryRelatedDecisions 从 MemoryGraph 查询相关决策
func (ca *ContextAssembler) queryRelatedDecisions(signal *StateChangeSignal) []*core.DecisionNode {
	var results []*core.DecisionNode
	seen := make(map[string]bool)

	// 按关键词搜索
	for _, kw := range signal.Context.Keywords {
		// 简化：遍历所有议题
		for _, topic := range ca.memory.ListTopics() {
			decisions := ca.memory.QueryByTopic(topic)
			for _, d := range decisions {
				if !seen[d.SDRID] && containsDecisionKeyword(d, kw) {
					seen[d.SDRID] = true
					results = append(results, d)
				}
			}
		}
	}

	return results
}

// containsDecisionKeyword 决策中是否包含关键词
func containsDecisionKeyword(d *core.DecisionNode, keyword string) bool {
	return contains(d.Title, keyword) || contains(d.Decision, keyword) || contains(d.Rationale, keyword)
}

// scoredDecision 带评分的决策
type scoredDecision struct {
	decision *core.DecisionNode
	score    float64
}

// scoreDecisions 按三个维度评分（signal-activation-engine.md §3.2）
func scoreDecisions(decisions []*core.DecisionNode, signal *StateChangeSignal) []scoredDecision {
	var scored []scoredDecision
	signalTime := signal.Timestamp

	for _, d := range decisions {
		// recency: e^(-0.1 × age_days)
		ageHours := signalTime.Sub(d.CreatedAt).Hours()
		ageDays := ageHours / 24
		recency := math.Exp(-0.1 * ageDays)

		// person_overlap: Jaccard(信号人员, 决策人员)
		signalPeople := append(signal.Context.MentionedIDs, signal.Context.ActorID)
		decisionPeople := append([]string{d.Proposer, d.Executor}, d.Stakeholders...)
		personOverlap := jaccard(signalPeople, decisionPeople)

		// keyword_overlap: Jaccard(信号关键词, 决策内容词)
		keywordOverlap := keywordJaccard(signal.Context.Keywords, d.Title+" "+d.Decision)

		// score = recency * 0.3 + person_overlap * 0.3 + keyword_overlap * 0.4
		score := recency*0.3 + personOverlap*0.3 + keywordOverlap*0.4

		scored = append(scored, scoredDecision{decision: d, score: score})
	}

	// 按分数降序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	return scored
}

// jaccard 计算 Jaccard 相似度
func jaccard(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	setA := make(map[string]bool)
	setB := make(map[string]bool)

	for _, s := range a {
		if s != "" {
			setA[s] = true
		}
	}
	for _, s := range b {
		if s != "" {
			setB[s] = true
		}
	}

	intersection := 0
	for s := range setA {
		if setB[s] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// keywordJaccard 关键词 Jaccard
func keywordJaccard(keywords []string, text string) float64 {
	if len(keywords) == 0 {
		return 0
	}
	matchCount := 0
	for _, kw := range keywords {
		if contains(text, kw) {
			matchCount++
		}
	}
	return float64(matchCount) / float64(len(keywords))
}

// estimateTokenCost 估算决策的 token 成本
func estimateTokenCost(d *core.DecisionNode) int {
	base := 200 // frontmatter 字段
	if d.Body != "" {
		base += len(d.Body) / 2
	}
	// 加上跨议题引用等额外字段
	base += len(d.CrossTopicRefs) * 20
	return base
}

// computeHints 计算关联提示（signal-activation-engine.md §3.4）
func (ca *ContextAssembler) computeHints(signal *StateChangeSignal, decisions []*core.DecisionNode) []CorrelationHint {
	var hints []CorrelationHint

	// DocDecisionLink: 信号中的文档 URL 与某决策关联同一文档
	for _, d := range decisions {
		for _, docToken := range d.FeishuLinks.RelatedDocTokens {
			for _, url := range signal.Context.EmbeddedURLs {
				if url.ExtractedToken == docToken {
					hints = append(hints, CorrelationHint{
						Type:        "DocDecisionLink",
						Description: "信号中的文档与已有决策关联",
						Confidence:  0.8,
						SourceSDR:   d.SDRID,
					})
				}
			}
		}
	}

	return hints
}
