package context

import (
	"time"

	"feishu-mem/internal/core"
	"feishu-mem/internal/decision"
)

// IndexContent 索引层内容
type IndexContent struct {
	ProjectStatus   string           `json:"project_status"`
	Topics          []TopicSummary   `json:"topics"`
	RecentDecisions []DecisionDigest `json:"recent_decisions"`
	PendingItems    PendingItems     `json:"pending_items"`
	AvailableTools  []string         `json:"available_tools"`
	RefreshedAt     time.Time        `json:"refreshed_at"`
}

// TopicSummary 主题摘要
type TopicSummary struct {
	Name              string `json:"name"`
	ActiveCount       int    `json:"active_count"`
	LastChangeAt      string `json:"last_change_at"`
	ImpactDistribution string `json:"impact_distribution"`
}

// DecisionDigest 决策摘要
type DecisionDigest struct {
	SDRID       string                    `json:"sdr_id"`
	Title       string                    `json:"title"`
	ImpactLevel decision.ImpactLevel     `json:"impact_level"`
	Status      decision.DecisionStatus  `json:"status"`
	Topic       string                    `json:"topic"`
}

// PendingItems 待处理事项
type PendingItems struct {
	Conflicts int `json:"conflicts"`
	Signals   int `json:"signals"`
}

// GenerateIndex 从 MemoryGraph 生成索引
func GenerateIndex(mg *core.MemoryGraph, project string) *IndexContent {
	allDecisions := mg.GetAllDecisions()

	index := &IndexContent{
		RefreshedAt: time.Now(),
		AvailableTools: []string{
			"memory.search",
			"memory.topic",
			"memory.decision",
			"memory.timeline",
			"memory.conflict",
			"memory.signal",
		},
	}

	// 收集主题统计
	topicMap := make(map[string]*TopicSummary)
	for _, d := range allDecisions {
		topic := d.Topic
		if _, ok := topicMap[topic]; !ok {
			topicMap[topic] = &TopicSummary{
				Name: topic,
			}
		}
		if d.IsActive() {
			topicMap[topic].ActiveCount++
		}
	}

	// 转换为列表
	for _, summary := range topicMap {
		index.Topics = append(index.Topics, *summary)
	}

	// 收集最近决策（取前5个）
	var recent []*decision.DecisionNode
	for _, d := range allDecisions {
		recent = append(recent, d)
		if len(recent) >= 5 {
			break
		}
	}

	// 转换为摘要
	for _, d := range recent {
		index.RecentDecisions = append(index.RecentDecisions, DecisionDigest{
			SDRID:       d.SDRID,
			Title:       d.Title,
			ImpactLevel: d.ImpactLevel,
			Status:      d.Status,
			Topic:       d.Topic,
		})
	}

	index.PendingItems = PendingItems{
		Conflicts: 0, // 暂时为0
		Signals:   0,
	}

	index.ProjectStatus = "active"

	return index
}

// RenderAsMarkdown 将索引渲染为 Markdown
func (idx *IndexContent) RenderAsMarkdown() string {
	var output string

	output += "## Project Memory Index\n"
	output += "Refreshed at: " + idx.RefreshedAt.Format(time.RFC3339) + "\n\n"

	// 主题列表
	output += "### Topics\n"
	for _, topic := range idx.Topics {
		output += "- **" + topic.Name + "** (" + string(rune(topic.ActiveCount+'0')) + " active decisions)\n"
	}

	// 最近决策
	output += "\n### Recent Decisions\n"
	for _, d := range idx.RecentDecisions {
		statusEmoji := "📋"
		switch d.Status {
		case decision.StatusDecided:
			statusEmoji = "✅"
		case decision.StatusExecuting:
			statusEmoji = "🔄"
		case decision.StatusCompleted:
			statusEmoji = "🏁"
		}
		output += "- " + statusEmoji + " **" + d.SDRID + "**: " + d.Title + " (" + d.Topic + ")\n"
	}

	// 可用工具
	output += "\n### Available Tools\n"
	for _, tool := range idx.AvailableTools {
		output += "- " + tool + "\n"
	}

	return output
}
