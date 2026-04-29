package signal

import (
	"math"
	"sort"

	"github.com/openclaw/internal/core"
)

// ActivationRouter 激活路由器（signal-activation-engine.md §2）
type ActivationRouter struct {
	matrix *ActivationMatrix
}

// ActivationMatrix 8×3×8 权重矩阵（signal-activation-engine.md §2.2）
type ActivationMatrix struct {
	weights map[core.AdapterType]map[core.SignalStrength]map[core.AdapterType]float64
}

// ActivationTarget 激活目标
type ActivationTarget struct {
	Adapter  core.AdapterType
	Weight   float64
	Priority core.Priority
}

// ContextQueryPlan 上下文查询计划（signal-activation-engine.md §2.3）
type ContextQueryPlan struct {
	SignalID         string
	TotalTokenBudget int
	Queries          []ContextQuery
}

// ContextQuery 上下文查询
type ContextQuery struct {
	QueryID       string
	TargetAdapter core.AdapterType
	QueryType     QueryType
	Params        QueryParams
	TokenBudget   int
	Priority      int    // 0 = 最高
	Purpose       string // 人类可读说明
}

// QueryType 查询类型
type QueryType string

const (
	QueryByKeyword       QueryType = "SearchByKeyword"
	QueryByPerson        QueryType = "SearchByPerson"
	QueryByEntity        QueryType = "SearchByEntity"
	QueryByTimeRange     QueryType = "SearchByTimeRange"
	QueryGetEntityDetail QueryType = "GetEntityDetail"
	QueryByTopic         QueryType = "SearchDecisionsByTopic"
)

// QueryParams 查询参数
type QueryParams map[string]interface{}

// NewActivationRouter 创建激活路由器
func NewActivationRouter() *ActivationRouter {
	return &ActivationRouter{
		matrix: NewDefaultActivationMatrix(),
	}
}

// Route 根据信号生成上下文查询计划（signal-activation-engine.md §2.4）
func (r *ActivationRouter) Route(signal *StateChangeSignal) *ContextQueryPlan {
	// 1. 根据信号强度确定 token 预算
	budget := computeBudget(signal.Strength)

	// 2. 始终优先搜索 MemoryGraph 中的已有决策
	var queries []ContextQuery

	qID := 0

	// 基于关键词的决策查询
	for _, kw := range signal.Context.Keywords {
		qID++
		queries = append(queries, ContextQuery{
			QueryID:       fmtQueryID(qID),
			TargetAdapter: core.AdapterIM,
			QueryType:     QueryByKeyword,
			Params:        QueryParams{"keyword": kw},
			TokenBudget:   budget / 4,
			Priority:      0,
			Purpose:       "搜索与关键词相关的已有决策",
		})
	}

	// 3. 查权重矩阵
	activations := r.matrix.Activate(signal.Adapter, signal.Strength)

	// 4. 为每个激活的适配器生成查询
	for _, act := range activations {
		if act.Priority == core.PrioritySkip {
			continue
		}
		qID++
		queries = append(queries, ContextQuery{
			QueryID:       fmtQueryID(qID),
			TargetAdapter: act.Adapter,
			QueryType:     QueryByKeyword,
			Params: QueryParams{
				"keywords": signal.Context.Keywords,
				"actor":    signal.Context.ActorID,
			},
			TokenBudget: budget / len(activations),
			Priority:    int(act.Priority),
			Purpose:     "从 " + string(act.Adapter) + " 获取关联上下文",
		})
	}

	// 5. 信号中的 URL → 直接查询对应实体
	for _, url := range signal.Context.EmbeddedURLs {
		if url.ExtractedToken != "" {
			qID++
			queries = append(queries, ContextQuery{
				QueryID:       fmtQueryID(qID),
				TargetAdapter: core.AdapterDocs,
				QueryType:     QueryGetEntityDetail,
				Params:        QueryParams{"doc_token": url.ExtractedToken},
				TokenBudget:   500,
				Priority:      0,
				Purpose:       "获取信号中的文档内容",
			})
		}
	}

	// 6. @提及的人员 → 查询人员关联
	for _, openID := range signal.Context.MentionedIDs {
		qID++
		queries = append(queries, ContextQuery{
			QueryID:       fmtQueryID(qID),
			TargetAdapter: core.AdapterContact,
			QueryType:     QueryByPerson,
			Params:        QueryParams{"open_id": openID},
			TokenBudget:   200,
			Priority:      1,
			Purpose:       "解析人员信息",
		})
	}

	// 7. 按 Priority 排序，裁剪到总预算
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].Priority < queries[j].Priority
	})
	queries = trimToBudget(queries, budget)

	return &ContextQueryPlan{
		SignalID:         signal.SignalID,
		TotalTokenBudget: budget,
		Queries:          queries,
	}
}

// computeBudget 根据信号强度计算 token 预算（signal-activation-engine.md §3.1）
func computeBudget(strength core.SignalStrength) int {
	switch strength {
	case core.SignalStrong:
		return 6000
	case core.SignalMedium:
		return 2000
	case core.SignalWeak:
		return 500
	default:
		return 500
	}
}

// trimToBudget 将查询裁剪到预算内
func trimToBudget(queries []ContextQuery, budget int) []ContextQuery {
	total := 0
	var result []ContextQuery
	for _, q := range queries {
		if total+q.TokenBudget > budget {
			break
		}
		total += q.TokenBudget
		result = append(result, q)
	}
	return result
}

func fmtQueryID(id int) string {
	return []string{"q0", "q1", "q2", "q3", "q4", "q5", "q6", "q7", "q8", "q9"}[id%10]
}

// ---- ActivationMatrix ----

// NewDefaultActivationMatrix 创建默认权重矩阵（signal-activation-engine.md §2.2）
func NewDefaultActivationMatrix() *ActivationMatrix {
	m := &ActivationMatrix{
		weights: make(map[core.AdapterType]map[core.SignalStrength]map[core.AdapterType]float64),
	}

	// 初始化
	for _, src := range allAdapters {
		m.weights[src] = make(map[core.SignalStrength]map[core.AdapterType]float64)
		for _, s := range []core.SignalStrength{core.SignalStrong, core.SignalMedium, core.SignalWeak} {
			m.weights[src][s] = make(map[core.AdapterType]float64)
		}
	}

	// IM 权重（signal-activation-engine.md §2.2 矩阵 第1行）
	m.set(core.AdapterIM, core.SignalStrong, core.AdapterVC, 0.8)
	m.set(core.AdapterIM, core.SignalStrong, core.AdapterDocs, 0.9)
	m.set(core.AdapterIM, core.SignalStrong, core.AdapterCalendar, 0.7)
	m.set(core.AdapterIM, core.SignalStrong, core.AdapterTask, 0.8)
	m.set(core.AdapterIM, core.SignalStrong, core.AdapterOKR, 0.3)
	m.set(core.AdapterIM, core.SignalStrong, core.AdapterContact, 0.6)
	m.set(core.AdapterIM, core.SignalStrong, core.AdapterWiki, 0.4)

	m.set(core.AdapterIM, core.SignalMedium, core.AdapterVC, 0.5)
	m.set(core.AdapterIM, core.SignalMedium, core.AdapterDocs, 0.7)
	m.set(core.AdapterIM, core.SignalMedium, core.AdapterCalendar, 0.4)
	m.set(core.AdapterIM, core.SignalMedium, core.AdapterTask, 0.6)
	m.set(core.AdapterIM, core.SignalMedium, core.AdapterOKR, 0.2)
	m.set(core.AdapterIM, core.SignalMedium, core.AdapterContact, 0.4)
	m.set(core.AdapterIM, core.SignalMedium, core.AdapterWiki, 0.3)

	m.set(core.AdapterIM, core.SignalWeak, core.AdapterVC, 0.2)
	m.set(core.AdapterIM, core.SignalWeak, core.AdapterDocs, 0.3)
	m.set(core.AdapterIM, core.SignalWeak, core.AdapterCalendar, 0.1)
	m.set(core.AdapterIM, core.SignalWeak, core.AdapterTask, 0.3)
	m.set(core.AdapterIM, core.SignalWeak, core.AdapterOKR, 0.0)
	m.set(core.AdapterIM, core.SignalWeak, core.AdapterContact, 0.2)
	m.set(core.AdapterIM, core.SignalWeak, core.AdapterWiki, 0.1)

	// VC 权重（第2行）
	m.set(core.AdapterVC, core.SignalStrong, core.AdapterIM, 0.9)
	m.set(core.AdapterVC, core.SignalStrong, core.AdapterDocs, 0.8)
	m.set(core.AdapterVC, core.SignalStrong, core.AdapterCalendar, 0.7)
	m.set(core.AdapterVC, core.SignalStrong, core.AdapterTask, 0.9)
	m.set(core.AdapterVC, core.SignalStrong, core.AdapterOKR, 0.4)
	m.set(core.AdapterVC, core.SignalStrong, core.AdapterContact, 0.7)
	m.set(core.AdapterVC, core.SignalStrong, core.AdapterWiki, 0.5)

	m.set(core.AdapterVC, core.SignalMedium, core.AdapterIM, 0.6)
	m.set(core.AdapterVC, core.SignalMedium, core.AdapterDocs, 0.5)
	m.set(core.AdapterVC, core.SignalMedium, core.AdapterCalendar, 0.4)
	m.set(core.AdapterVC, core.SignalMedium, core.AdapterTask, 0.7)
	m.set(core.AdapterVC, core.SignalMedium, core.AdapterOKR, 0.2)
	m.set(core.AdapterVC, core.SignalMedium, core.AdapterContact, 0.5)
	m.set(core.AdapterVC, core.SignalMedium, core.AdapterWiki, 0.3)

	// Docs 权重（第3行）
	m.set(core.AdapterDocs, core.SignalStrong, core.AdapterIM, 0.8)
	m.set(core.AdapterDocs, core.SignalStrong, core.AdapterVC, 0.6)
	m.set(core.AdapterDocs, core.SignalStrong, core.AdapterCalendar, 0.5)
	m.set(core.AdapterDocs, core.SignalStrong, core.AdapterTask, 0.7)
	m.set(core.AdapterDocs, core.SignalStrong, core.AdapterOKR, 0.3)
	m.set(core.AdapterDocs, core.SignalStrong, core.AdapterContact, 0.5)
	m.set(core.AdapterDocs, core.SignalStrong, core.AdapterWiki, 0.7)

	// Task 权重（第5行）
	m.set(core.AdapterTask, core.SignalStrong, core.AdapterIM, 0.7)
	m.set(core.AdapterTask, core.SignalStrong, core.AdapterVC, 0.5)
	m.set(core.AdapterTask, core.SignalStrong, core.AdapterDocs, 0.6)
	m.set(core.AdapterTask, core.SignalStrong, core.AdapterCalendar, 0.7)
	m.set(core.AdapterTask, core.SignalStrong, core.AdapterOKR, 0.6)
	m.set(core.AdapterTask, core.SignalStrong, core.AdapterContact, 0.4)
	m.set(core.AdapterTask, core.SignalStrong, core.AdapterWiki, 0.3)

	return m
}

func (m *ActivationMatrix) set(src core.AdapterType, strength core.SignalStrength, tgt core.AdapterType, w float64) {
	// 不对自身
	if src == tgt {
		return
	}
	m.weights[src][strength][tgt] = w
}

// Activate 查询矩阵，获取需要激活的目标适配器（按权重降序）（signal-activation-engine.md §2.4 step 3）
func (m *ActivationMatrix) Activate(src core.AdapterType, strength core.SignalStrength) []ActivationTarget {
	row, ok := m.weights[src][strength]
	if !ok {
		return nil
	}

	var targets []ActivationTarget
	for adapter, weight := range row {
		priority := classifyPriority(weight)
		targets = append(targets, ActivationTarget{
			Adapter:  adapter,
			Weight:   weight,
			Priority: priority,
		})
	}

	// 按权重降序
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Weight > targets[j].Weight
	})

	return targets
}

// classifyPriority 根据权重计算优先级（signal-activation-engine.md §2.2 权重阈值）
func classifyPriority(weight float64) core.Priority {
	switch {
	case weight >= 0.7:
		return core.PriorityMust
	case weight >= 0.4:
		return core.PriorityShould
	case weight >= 0.1:
		return core.PriorityMay
	default:
		return core.PrioritySkip
	}
}

// Reinforce 成功关联后，增加提供有用上下文的适配器权重（signal-activation-engine.md §2.5）
func (m *ActivationMatrix) Reinforce(src core.AdapterType, strength core.SignalStrength, usefulAdapter core.AdapterType, score float64) {
	old := m.weights[src][strength][usefulAdapter]
	// 指数移动平均 α=0.1
	newWeight := 0.9*old + 0.1*score
	m.weights[src][strength][usefulAdapter] = math.Min(newWeight, 1.0)
}

// Decay 定期衰减不常用的权重（signal-activation-engine.md §2.5）
func (m *ActivationMatrix) Decay(decayRate float64) {
	for src := range m.weights {
		for strength := range m.weights[src] {
			for tgt := range m.weights[src][strength] {
				m.weights[src][strength][tgt] *= (1 - decayRate)
			}
		}
	}
}

var allAdapters = []core.AdapterType{
	core.AdapterIM, core.AdapterVC, core.AdapterDocs,
	core.AdapterCalendar, core.AdapterTask, core.AdapterOKR,
	core.AdapterContact, core.AdapterWiki,
}
