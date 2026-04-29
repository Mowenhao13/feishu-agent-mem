package core

import (
	"sync"
	"sync/atomic"
)

// MemoryGraph 内存中的决策图 — 运行时加速，启动时从 Git 重建（openclaw-architecture.md §2.1.2）
type MemoryGraph struct {
	mu sync.RWMutex

	// 决策索引: sdr_id → DecisionNode
	decisions map[string]*DecisionNode

	// 议题索引: topic_name → []sdr_id
	topics map[string][]string

	// 关系索引: sdr_id → []Relation
	relations map[string][]Relation

	// 跨议题引用索引: topic → []related_topic
	crossTopicRefs map[string][]string

	// 脏标记 — 待同步到 Git/Bitable 的决策
	dirtySet map[string]struct{}

	// 总计数
	count atomic.Int64
}

// NewMemoryGraph 创建新的 MemoryGraph
func NewMemoryGraph() *MemoryGraph {
	return &MemoryGraph{
		decisions:      make(map[string]*DecisionNode),
		topics:         make(map[string][]string),
		relations:      make(map[string][]Relation),
		crossTopicRefs: make(map[string][]string),
		dirtySet:       make(map[string]struct{}),
	}
}

// AddDecision 添加一条决策到内存索引
func (mg *MemoryGraph) AddDecision(d *DecisionNode) {
	mg.mu.Lock()
	defer mg.mu.Unlock()

	mg.decisions[d.SDRID] = d
	mg.topics[d.Topic] = append(mg.topics[d.Topic], d.SDRID)
	mg.relations[d.SDRID] = d.Relations
	mg.count.Add(1)

	// 更新跨议题引用索引
	for _, ref := range d.CrossTopicRefs {
		mg.crossTopicRefs[ref] = append(mg.crossTopicRefs[ref], d.Topic)
	}
}

// UpsertDecision 插入或更新决策
func (mg *MemoryGraph) UpsertDecision(d *DecisionNode) {
	mg.mu.Lock()
	defer mg.mu.Unlock()

	old, exists := mg.decisions[d.SDRID]
	if exists {
		// 从旧 topic 索引中移除
		oldTopic := old.Topic
		topics := mg.topics[oldTopic]
		for i, id := range topics {
			if id == d.SDRID {
				mg.topics[oldTopic] = append(topics[:i], topics[i+1:]...)
				break
			}
		}
	}

	mg.decisions[d.SDRID] = d
	mg.topics[d.Topic] = append(mg.topics[d.Topic], d.SDRID)
	mg.relations[d.SDRID] = d.Relations

	if !exists {
		mg.count.Add(1)
	}

	// 更新跨议题引用
	for _, ref := range d.CrossTopicRefs {
		mg.crossTopicRefs[ref] = append(mg.crossTopicRefs[ref], d.Topic)
	}
}

// GetDecision 按 SDR ID 获取决策
func (mg *MemoryGraph) GetDecision(sdrID string) *DecisionNode {
	mg.mu.RLock()
	defer mg.mu.RUnlock()

	return mg.decisions[sdrID]
}

// QueryByTopic 按议题检索 active 决策
func (mg *MemoryGraph) QueryByTopic(topic string) []*DecisionNode {
	mg.mu.RLock()
	defer mg.mu.RUnlock()

	var result []*DecisionNode
	for _, id := range mg.topics[topic] {
		if d, ok := mg.decisions[id]; ok && d.Status == StatusPending || d.Status == StatusInDiscussion || d.Status == StatusDecided || d.Status == StatusExecuting {
			result = append(result, d)
		}
	}
	return result
}

// QueryAllByTopic 按议题检索所有决策（含已完成的）
func (mg *MemoryGraph) QueryAllByTopic(topic string) []*DecisionNode {
	mg.mu.RLock()
	defer mg.mu.RUnlock()

	var result []*DecisionNode
	for _, id := range mg.topics[topic] {
		if d, ok := mg.decisions[id]; ok {
			result = append(result, d)
		}
	}
	return result
}

// QueryCrossTopic 跨议题检索（含 cross_topic_refs）（decision-tree.md §4.3）
func (mg *MemoryGraph) QueryCrossTopic(topic string) []*DecisionNode {
	mg.mu.RLock()
	defer mg.mu.RUnlock()

	seen := make(map[string]bool)
	var result []*DecisionNode

	// 1. 同议题
	for _, id := range mg.topics[topic] {
		if d, ok := mg.decisions[id]; ok {
			seen[d.SDRID] = true
			result = append(result, d)
		}
	}

	// 2. cross_topic_refs 指向该 topic 的决策
	for _, id := range mg.decisions {
		for _, ref := range id.CrossTopicRefs {
			if ref == topic && !seen[id.SDRID] {
				seen[id.SDRID] = true
				result = append(result, id)
			}
		}
	}

	return result
}

// DetectConflicts 检测新决策与现有决策的冲突（decision-tree.md §5.3）
func (mg *MemoryGraph) DetectConflicts(new *DecisionNode) []Conflict {
	mg.mu.RLock()
	defer mg.mu.RUnlock()

	var conflicts []Conflict

	// 候选冲突集: 同 Topic active 决策
	candidates := mg.topics[new.Topic]
	for _, id := range candidates {
		existing := mg.decisions[id]
		if existing == nil {
			continue
		}
		// 只与 active 决策比较
		if existing.Status != StatusPending && existing.Status != StatusInDiscussion &&
			existing.Status != StatusDecided && existing.Status != StatusExecuting {
			continue
		}
		if existing.SDRID == new.SDRID {
			continue
		}

		// 这里简化处理：返回候选列表，实际的语义矛盾评估由 LLM 完成
		conflicts = append(conflicts, Conflict{
			DecisionA: existing.SDRID,
			DecisionB: new.SDRID,
			Status:    "pending",
		})
	}

	// 跨议题冲突: cross_topic_refs 指向的 Topic 的 active 决策
	for _, refTopic := range new.CrossTopicRefs {
		for _, id := range mg.topics[refTopic] {
			existing := mg.decisions[id]
			if existing == nil {
				continue
			}
			if existing.Status != StatusPending && existing.Status != StatusInDiscussion &&
				existing.Status != StatusDecided && existing.Status != StatusExecuting {
				continue
			}
			conflicts = append(conflicts, Conflict{
				DecisionA: existing.SDRID,
				DecisionB: new.SDRID,
				Status:    "pending",
			})
		}
	}

	return conflicts
}

// MarkDirty 标记脏记录
func (mg *MemoryGraph) MarkDirty(sdrID string) {
	mg.mu.Lock()
	defer mg.mu.Unlock()
	mg.dirtySet[sdrID] = struct{}{}
}

// DirtyDecisions 获取所有脏记录
func (mg *MemoryGraph) DirtyDecisions() []string {
	mg.mu.RLock()
	defer mg.mu.RUnlock()

	ids := make([]string, 0, len(mg.dirtySet))
	for id := range mg.dirtySet {
		ids = append(ids, id)
	}
	return ids
}

// ClearDirty 清除脏标记
func (mg *MemoryGraph) ClearDirty(sdrID string) {
	mg.mu.Lock()
	defer mg.mu.Unlock()
	delete(mg.dirtySet, sdrID)
}

// RemoveDecision 移除决策
func (mg *MemoryGraph) RemoveDecision(sdrID string) {
	mg.mu.Lock()
	defer mg.mu.Unlock()

	d, ok := mg.decisions[sdrID]
	if !ok {
		return
	}

	// 从 topic 索引移除
	topics := mg.topics[d.Topic]
	for i, id := range topics {
		if id == sdrID {
			mg.topics[d.Topic] = append(topics[:i], topics[i+1:]...)
			break
		}
	}

	delete(mg.decisions, sdrID)
	delete(mg.relations, sdrID)
	delete(mg.dirtySet, sdrID)
	mg.count.Add(-1)
}

// ListTopics 列出所有议题
func (mg *MemoryGraph) ListTopics() []string {
	mg.mu.RLock()
	defer mg.mu.RUnlock()

	topics := make([]string, 0, len(mg.topics))
	for t := range mg.topics {
		topics = append(topics, t)
	}
	return topics
}

// Count 返回决策总数
func (mg *MemoryGraph) Count() int {
	return int(mg.count.Load())
}

// ListDecisionsByIDs 批量获取决策
func (mg *MemoryGraph) ListDecisionsByIDs(ids []string) []*DecisionNode {
	mg.mu.RLock()
	defer mg.mu.RUnlock()

	result := make([]*DecisionNode, 0, len(ids))
	for _, id := range ids {
		if d, ok := mg.decisions[id]; ok {
			result = append(result, d)
		}
	}
	return result
}
