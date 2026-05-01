package context

// TokenBudget Token 预算配置
type TokenBudget struct {
	IndexLayer    int
	SearchResults int
	TopicDetail   int
	DecisionYAML  int
	DecisionFull  int
	Timeline      int
	Conflict      int
	Signals       int
	MaxTotal      int
}

// DefaultBudget 默认预算
func DefaultBudget() TokenBudget {
	return TokenBudget{
		IndexLayer:    800,
		SearchResults: 500,
		TopicDetail:   1000,
		DecisionYAML:  300,
		DecisionFull:  800,
		Timeline:      600,
		Conflict:      400,
		Signals:       300,
		MaxTotal:      8000,
	}
}

// BudgetTracker 预算追踪器
type BudgetTracker struct {
	Budget TokenBudget
	Used   int
	History []string
}

// NewBudgetTracker 创建预算追踪器
func NewBudgetTracker(budget TokenBudget) *BudgetTracker {
	return &BudgetTracker{
		Budget: budget,
		Used: 0,
		History: []string{},
	}
}

// CanUse 检查工具是否有足够预算
func (bt *BudgetTracker) CanUse(tool string) bool {
	cost := bt.toolCost(tool)
	return bt.Used+cost <= bt.Budget.MaxTotal
}

// Consume 消耗预算
func (bt *BudgetTracker) Consume(tool string) int {
	cost := bt.toolCost(tool)
	bt.Used += cost
	bt.History = append(bt.History, tool)
	return cost
}

// Remaining 获取剩余预算
func (bt *BudgetTracker) Remaining() int {
	return bt.Budget.MaxTotal - bt.Used
}

// Reset 重置预算
func (bt *BudgetTracker) Reset() {
	bt.Used = 0
	bt.History = []string{}
}

func (bt *BudgetTracker) toolCost(tool string) int {
	switch tool {
	case "memory.search":
		return bt.Budget.SearchResults
	case "memory.topic":
		return bt.Budget.TopicDetail
	case "memory.decision":
		return bt.Budget.DecisionYAML
	case "memory.decision_full":
		return bt.Budget.DecisionFull
	case "memory.timeline":
		return bt.Budget.Timeline
	case "memory.conflict":
		return bt.Budget.Conflict
	case "memory.signal":
		return bt.Budget.Signals
	default:
		return 100 // 默认成本
	}
}
