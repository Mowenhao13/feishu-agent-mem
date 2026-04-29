package memory

import (
	"fmt"
	"time"

	"github.com/openclaw/internal/core"
)

// MemoryIndex 记忆索引 — 始终注入 LLM 上下文的轻量索引（memory-openclaw-integration.md §3.2）
type MemoryIndex struct {
	IndexRefreshTime time.Time         `json:"index_refresh_time"`
	ProjectStatus    string            `json:"project_status"`
	TopicSummary     []TopicSummary    `json:"topic_summary"`
	RecentDecisions  []DecisionSummary `json:"recent_decisions"`
	PendingItems     PendingItems      `json:"pending_items"`
}

// TopicSummary 议题摘要
type TopicSummary struct {
	Topic      string `json:"topic"`
	Active     int    `json:"active"`
	LastChange string `json:"last_change"`
	LevelDist  string `json:"level_dist"`
}

// DecisionSummary 决策摘要
type DecisionSummary struct {
	SDRID       string `json:"sdr_id"`
	Title       string `json:"title"`
	ImpactLevel string `json:"impact_level"`
	Status      string `json:"status"`
}

// PendingItems 待处理项
type PendingItems struct {
	UnresolvedConflicts int `json:"unresolved_conflicts"`
	PendingSignals      int `json:"pending_signals"`
}

// IndexBuilder 索引构建器
type IndexBuilder struct {
	memory *core.MemoryGraph
}

// NewIndexBuilder 创建索引构建器
func NewIndexBuilder(mg *core.MemoryGraph) *IndexBuilder {
	return &IndexBuilder{memory: mg}
}

// BuildIndex 构建当前索引
func (ib *IndexBuilder) BuildIndex() *MemoryIndex {
	idx := &MemoryIndex{
		IndexRefreshTime: time.Now(),
		ProjectStatus:    "活跃",
	}

	// 获取各议题统计
	for _, topic := range ib.memory.ListTopics() {
		all := ib.memory.QueryAllByTopic(topic)
		active := ib.memory.QueryByTopic(topic)

		var majorCount, minorCount int
		for _, d := range all {
			switch d.ImpactLevel {
			case core.ImpactMajor, core.ImpactCritical:
				majorCount++
			case core.ImpactMinor, core.ImpactAdvisory:
				minorCount++
			}
		}

		lastChange := "N/A"
		if len(all) > 0 {
			latest := all[0]
			for _, d := range all {
				if d.CreatedAt.After(latest.CreatedAt) {
					latest = d
				}
			}
			lastChange = fmt.Sprintf("%dh 前", int(time.Since(latest.CreatedAt).Hours()))
		}

		idx.TopicSummary = append(idx.TopicSummary, TopicSummary{
			Topic:      topic,
			Active:     len(active),
			LastChange: lastChange,
			LevelDist:  fmt.Sprintf("major:%d, minor:%d", majorCount, minorCount),
		})
	}

	// 最近 7 天的决策
	recentCutoff := time.Now().Add(-7 * 24 * time.Hour)
	for _, topic := range ib.memory.ListTopics() {
		for _, d := range ib.memory.QueryAllByTopic(topic) {
			if d.CreatedAt.After(recentCutoff) {
				idx.RecentDecisions = append(idx.RecentDecisions, DecisionSummary{
					SDRID:       d.SDRID,
					Title:       d.Title,
					ImpactLevel: d.ImpactLevel.String(),
					Status:      d.Status.String(),
				})
				if len(idx.RecentDecisions) >= 10 {
					break
				}
			}
		}
	}

	return idx
}

// FormatIndex 格式化索引为 LLM 可读的 Markdown（~800 tokens）
func (idx *MemoryIndex) FormatIndex() string {
	text := fmt.Sprintf("## 记忆系统索引（自动生成，最后刷新: %s）\n\n", idx.IndexRefreshTime.Format("2006-01-02 15:04"))
	text += fmt.Sprintf("### 项目状态\n- 项目状态: %s\n\n", idx.ProjectStatus)

	text += "### 议题概览\n"
	text += "| 议题 | Active | 最近变更 | 影响级别分布 |\n"
	text += "|------|--------|---------|------------|\n"
	for _, t := range idx.TopicSummary {
		text += fmt.Sprintf("| %s | %d | %s | %s |\n", t.Topic, t.Active, t.LastChange, t.LevelDist)
	}

	text += "\n### 最近决策（最近 7 天）\n"
	for _, d := range idx.RecentDecisions {
		text += fmt.Sprintf("- [%s] %s → %s, %s\n", d.SDRID, d.Title, d.ImpactLevel, d.Status)
	}

	text += "\n### 待处理\n"
	text += fmt.Sprintf("- 未解决冲突: %d\n", idx.PendingItems.UnresolvedConflicts)
	text += fmt.Sprintf("- 待确认信号: %d\n", idx.PendingItems.PendingSignals)

	text += "\n### 可用工具\n"
	text += "- `memory.search` — 搜索决策\n"
	text += "- `memory.topic` — 查看某议题详情\n"
	text += "- `memory.decision` — 查看某决策详情\n"
	text += "- `memory.timeline` — 查看时间线\n"
	text += "- `memory.conflict` — 查看冲突详情\n"
	text += "- `memory.signal` — 查看最近信号\n"

	return text
}
