package core

import (
	"fmt"
	"time"
)

// ConflictResolver 冲突解决器（decision-tree.md §5.3）
type ConflictResolver struct {
	memory *MemoryGraph
}

// NewConflictResolver 创建冲突解决器
func NewConflictResolver(mg *MemoryGraph) *ConflictResolver {
	return &ConflictResolver{memory: mg}
}

// Resolve 根据权重和语义矛盾评估结果处理冲突
// 语义矛盾评分由 LLM 外部提供（0.0-1.0）
func (cr *ConflictResolver) Resolve(newDecision *DecisionNode, oldDecision *DecisionNode, contradictionScore float64) (*Resolution, error) {
	if contradictionScore < 0.3 {
		// 无冲突，正常插入
		return &Resolution{Action: ResolveNoConflict}, nil
	}

	if contradictionScore >= 0.3 && contradictionScore <= 0.6 {
		// 自动标记 RELATES_TO（这里简化为轻度警告）
		return &Resolution{
			Action: ResolveRelates,
			Summary: fmt.Sprintf("决策 %s 与 %s 轻度相关 (score: %.2f)",
				newDecision.SDRID, oldDecision.SDRID, contradictionScore),
		}, nil
	}

	// > 0.6: 需要判断权重
	if newDecision.ParentDecision == oldDecision.SDRID {
		// 新决策是旧决策的细化
		return &Resolution{
			Action:  ResolveRefines,
			Refines: &oldDecision.SDRID,
			Summary: fmt.Sprintf("决策 %s 细化 %s", newDecision.SDRID, oldDecision.SDRID),
		}, nil
	}

	// 权重比较
	weightCmp := compareImpactLevel(newDecision.ImpactLevel, oldDecision.ImpactLevel)
	switch {
	case weightCmp > 0:
		// 新决策权重更高 → 替代旧决策
		return &Resolution{
			Action:     ResolveSupersedes,
			Supersedes: &oldDecision.SDRID,
			Summary:    fmt.Sprintf("决策 %s 替代 %s（权重更高）", newDecision.SDRID, oldDecision.SDRID),
		}, nil
	case weightCmp < 0:
		// 旧决策权重更高 → 冲突，需人工裁决
		return &Resolution{
			Action:  ResolveConflict,
			Summary: fmt.Sprintf("决策 %s 与 %s 冲突，旧决策权重更高，需人工裁决", newDecision.SDRID, oldDecision.SDRID),
		}, nil
	default:
		// 权重相当 → 创建 CONFLICT_WITH 关系
		return &Resolution{
			Action:  ResolveConflict,
			Summary: fmt.Sprintf("决策 %s 与 %s 冲突 (score: %.2f)，需人工裁决", newDecision.SDRID, oldDecision.SDRID, contradictionScore),
		}, nil
	}
}

// ResolutionAction 冲突解决动作
type ResolutionAction int

const (
	ResolveNoConflict ResolutionAction = iota
	ResolveRelates
	ResolveRefines
	ResolveSupersedes
	ResolveConflict // 需要人工裁决
)

// Resolution 冲突解决结果
type Resolution struct {
	Action     ResolutionAction
	Supersedes *string
	Refines    *string
	Summary    string
}

// compareImpactLevel 比较影响级别
// 返回 >0 表示 a > b, <0 表示 a < b, 0 表示相等
func compareImpactLevel(a, b ImpactLevel) int {
	return int(a) - int(b)
}

// CreateConflictRecord 创建冲突记录
func (cr *ConflictResolver) CreateConflictRecord(decisionA, decisionB string, score float64, description string) *Conflict {
	return &Conflict{
		ConflictID:         fmt.Sprintf("conf_%s", time.Now().Format("20060102_150405")),
		DecisionA:          decisionA,
		DecisionB:          decisionB,
		ContradictionScore: score,
		Description:        description,
		Status:             "pending",
		CreatedAt:          time.Now(),
	}
}

// L0Check 检查是否违反 L0 核心规则（decision-tree.md §3.1 / git-operations-design.md §3.3）
func (cr *ConflictResolver) L0Check(decision *DecisionNode) error {
	if decision.ImpactLevel == ImpactCritical && decision.Executor == "" {
		return fmt.Errorf("L0-003: critical 级别决策必须指定执行人")
	}
	return nil
}
