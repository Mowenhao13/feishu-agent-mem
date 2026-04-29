package memory

import (
	"time"

	"github.com/openclaw/internal/core"
)

// ---- MCP 工具定义（memory-openclaw-integration.md §3.3）----

// SearchInput memory.search 输入
type SearchInput struct {
	Query       string `json:"query"`
	Topic       string `json:"topic,omitempty"`
	ImpactLevel string `json:"impact_level,omitempty"`
	Status      string `json:"status,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

// SearchResult memory.search 输出
type SearchResult struct {
	Results   []SearchResultItem `json:"results"`
	Total     int                `json:"total"`
	TokenCost int                `json:"token_cost"`
}

// SearchResultItem 搜索结果条目
type SearchResultItem struct {
	SDRID       string  `json:"sdr_id"`
	Title       string  `json:"title"`
	Topic       string  `json:"topic"`
	ImpactLevel string  `json:"impact_level"`
	Status      string  `json:"status"`
	Phase       string  `json:"phase"`
	DecidedAt   string  `json:"decided_at,omitempty"`
	Relevance   float64 `json:"relevance_score"`
}

// TopicInput memory.topic 输入
type TopicInput struct {
	TopicName        string `json:"topic_name"`
	IncludeCrossRefs bool   `json:"include_cross_refs"`
}

// TopicResult memory.topic 输出
type TopicResult struct {
	Topic           string              `json:"topic"`
	ActiveDecisions []TopicDecisionItem `json:"active_decisions"`
	CrossTopicRefs  []CrossRefItem      `json:"cross_topic_refs,omitempty"`
	TotalActive     int                 `json:"total_active"`
	TokenCost       int                 `json:"token_cost"`
}

// TopicDecisionItem 议题下决策条目
type TopicDecisionItem struct {
	SDRID       string `json:"sdr_id"`
	Title       string `json:"title"`
	ImpactLevel string `json:"impact_level"`
	Parent      string `json:"parent,omitempty"`
	Children    int    `json:"children"`
}

// CrossRefItem 跨议题引用条目
type CrossRefItem struct {
	SDRID     string `json:"sdr_id"`
	FromTopic string `json:"from_topic"`
	Relation  string `json:"relation"`
}

// DecisionInput memory.decision 输入
type DecisionInput struct {
	SDRID            string `json:"sdr_id"`
	IncludeFull      bool   `json:"include_full"`
	IncludeHistory   bool   `json:"include_history"`
	IncludeRelations bool   `json:"include_relations"`
}

// DecisionInputResult 定义在下面

// DecisionInputResult memory.decision 输出（简化）
type DecisionInputResult struct {
	SDRID          string          `json:"sdr_id"`
	Title          string          `json:"title"`
	Decision       string          `json:"decision"`
	Rationale      string          `json:"rationale"`
	Topic          string          `json:"topic"`
	Phase          string          `json:"phase"`
	ImpactLevel    string          `json:"impact_level"`
	CrossTopicRefs []string        `json:"cross_topic_refs"`
	Status         string          `json:"status"`
	Proposer       string          `json:"proposer"`
	Executor       string          `json:"executor"`
	FullContent    string          `json:"full_content,omitempty"`
	Relations      []core.Relation `json:"relations,omitempty"`
	TokenCost      int             `json:"token_cost"`
}

// TimelineInput memory.timeline 输入
type TimelineInput struct {
	Project string `json:"project"`
	Topic   string `json:"topic,omitempty"`
	Days    int    `json:"days"`
}

// TimelineResult memory.timeline 输出
type TimelineResult struct {
	Timeline  []TimelineEvent `json:"timeline"`
	TokenCost int             `json:"token_cost"`
}

// TimelineEvent 时间线事件
type TimelineEvent struct {
	Date   string         `json:"date"`
	Events []TimelineItem `json:"events"`
}

// TimelineItem 时间线条目
type TimelineItem struct {
	SDRID  string `json:"sdr_id"`
	Action string `json:"action"` // created | updated | status_change
	Title  string `json:"title"`
	Actor  string `json:"actor"`
}

// ConflictInput memory.conflict 输入
type ConflictInput struct {
	ConflictID string `json:"conflict_id,omitempty"`
}

// ConflictResult memory.conflict 输出
type ConflictResult struct {
	Conflicts []ConflictItem `json:"conflicts"`
	TokenCost int            `json:"token_cost"`
}

// ConflictItem 冲突条目
type ConflictItem struct {
	ConflictID         string  `json:"conflict_id"`
	DecisionA          string  `json:"decision_a"`
	DecisionB          string  `json:"decision_b"`
	ContradictionScore float64 `json:"contradiction_score"`
	Description        string  `json:"description"`
	Status             string  `json:"status"`
}

// SignalInput memory.signal 输入
type SignalInput struct {
	Adapter  string `json:"adapter,omitempty"`
	Strength string `json:"strength,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// SignalResult memory.signal 输出
type SignalResult struct {
	Signals   []SignalItem `json:"signals"`
	TokenCost int          `json:"token_cost"`
}

// SignalItem 信号条目
type SignalItem struct {
	SignalID   string `json:"signal_id"`
	Adapter    string `json:"adapter"`
	Strength   string `json:"strength"`
	Summary    string `json:"summary"`
	Timestamp  string `json:"timestamp"`
	DecisionID string `json:"decision_id,omitempty"`
}

// MemoryTools 记忆系统工具集合
type MemoryTools struct {
	graph *core.MemoryGraph
}

// NewMemoryTools 创建工具集合
func NewMemoryTools(mg *core.MemoryGraph) *MemoryTools {
	return &MemoryTools{graph: mg}
}

// Search 搜索决策（memory-openclaw-integration.md §6.3.1）
func (mt *MemoryTools) Search(input SearchInput) *SearchResult {
	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}

	var results []SearchResultItem
	seen := make(map[string]bool)

	// 遍历所有议题
	for _, topic := range mt.graph.ListTopics() {
		if input.Topic != "" && topic != input.Topic {
			continue
		}

		decisions := mt.graph.QueryAllByTopic(topic)
		for _, d := range decisions {
			if seen[d.SDRID] {
				continue
			}
			if !matchesFilter(d, input) {
				continue
			}
			if !keywordMatch(d, input.Query) {
				continue
			}

			seen[d.SDRID] = true
			results = append(results, SearchResultItem{
				SDRID:       d.SDRID,
				Title:       d.Title,
				Topic:       d.Topic,
				ImpactLevel: d.ImpactLevel.String(),
				Status:      d.Status.String(),
				Phase:       d.Phase,
				Relevance:   calcRelevance(d, input.Query),
			})

			if len(results) >= limit {
				break
			}
		}
	}

	return &SearchResult{
		Results:   results,
		Total:     len(results),
		TokenCost: len(results) * 50,
	}
}

// Topic 查看议题详情（memory-openclaw-integration.md §6.3.2）
func (mt *MemoryTools) Topic(input TopicInput) *TopicResult {
	decisions := mt.graph.QueryAllByTopic(input.TopicName)

	var activeItems []TopicDecisionItem
	for _, d := range decisions {
		activeItems = append(activeItems, TopicDecisionItem{
			SDRID:       d.SDRID,
			Title:       d.Title,
			ImpactLevel: d.ImpactLevel.String(),
			Parent:      d.ParentDecision,
			Children:    d.ChildrenCount,
		})
	}

	result := &TopicResult{
		Topic:           input.TopicName,
		ActiveDecisions: activeItems,
		TotalActive:     len(activeItems),
		TokenCost:       len(activeItems) * 80,
	}

	// 跨议题引用
	if input.IncludeCrossRefs {
		cross := mt.graph.QueryCrossTopic(input.TopicName)
		for _, d := range cross {
			if d.Topic == input.TopicName {
				continue
			}
			result.CrossTopicRefs = append(result.CrossTopicRefs, CrossRefItem{
				SDRID:     d.SDRID,
				FromTopic: d.Topic,
			})
		}
	}

	return result
}

// Decision 查看决策详情（memory-openclaw-integration.md §6.3.3）
func (mt *MemoryTools) Decision(input DecisionInput) *DecisionInputResult {
	d := mt.graph.GetDecision(input.SDRID)
	if d == nil {
		return nil
	}

	detail := &DecisionInputResult{
		SDRID:          d.SDRID,
		Title:          d.Title,
		Decision:       d.Decision,
		Rationale:      d.Rationale,
		Topic:          d.Topic,
		Phase:          d.Phase,
		ImpactLevel:    d.ImpactLevel.String(),
		CrossTopicRefs: d.CrossTopicRefs,
		Status:         d.Status.String(),
		Proposer:       d.Proposer,
		Executor:       d.Executor,
		TokenCost:      300,
	}

	if input.IncludeFull {
		detail.FullContent = d.Body
		detail.TokenCost = 800
	}

	if input.IncludeRelations {
		detail.Relations = d.Relations
	}

	return detail
}

// Timeline 查看时间线（memory-openclaw-integration.md §6.3.4）
func (mt *MemoryTools) Timeline(input TimelineInput) *TimelineResult {
	if input.Days <= 0 {
		input.Days = 30
	}

	cutoff := time.Now().Add(-time.Duration(input.Days) * 24 * time.Hour)
	events := make(map[string][]TimelineItem)

	for _, topic := range mt.graph.ListTopics() {
		if input.Topic != "" && topic != input.Topic {
			continue
		}
		for _, d := range mt.graph.QueryAllByTopic(topic) {
			if d.CreatedAt.Before(cutoff) {
				continue
			}
			dateKey := d.CreatedAt.Format("2006-01-02")
			events[dateKey] = append(events[dateKey], TimelineItem{
				SDRID:  d.SDRID,
				Action: "created",
				Title:  d.Title,
				Actor:  d.Proposer,
			})
		}
	}

	var timeline []TimelineEvent
	for date, items := range events {
		timeline = append(timeline, TimelineEvent{Date: date, Events: items})
	}

	return &TimelineResult{
		Timeline:  timeline,
		TokenCost: len(timeline) * 50,
	}
}

// ---- 辅助函数 ----

func matchesFilter(d *core.DecisionNode, input SearchInput) bool {
	if input.ImpactLevel != "" && d.ImpactLevel.String() != input.ImpactLevel {
		return false
	}
	if input.Status != "" && d.Status.String() != input.Status {
		return false
	}
	return true
}

func keywordMatch(d *core.DecisionNode, query string) bool {
	if query == "" {
		return true
	}
	return contains(d.Title, query) || contains(d.Decision, query) || contains(d.Rationale, query)
}

func calcRelevance(d *core.DecisionNode, query string) float64 {
	if query == "" {
		return 0.5
	}
	score := 0.0
	if contains(d.Title, query) {
		score += 0.4
	}
	if contains(d.Decision, query) {
		score += 0.3
	}
	if contains(d.Rationale, query) {
		score += 0.2
	}
	return score
}

// contains 子串搜索
func contains(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
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
