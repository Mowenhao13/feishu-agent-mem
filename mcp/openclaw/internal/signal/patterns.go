package signal

import (
	"time"

	"github.com/openclaw/internal/core"
)

// CorrelationPattern 跨适配器关联模式（signal-activation-engine.md §5）
type CorrelationPattern struct {
	PatternID           string
	Name                string
	TriggerSignals      []SignalPattern
	RequiredContext     []ContextRequirement
	Action              PatternAction
	ConfidenceThreshold float64
}

// SignalPattern 信号匹配模式
type SignalPattern struct {
	Adapter     core.AdapterType
	MinStrength core.SignalStrength
	TimeWindow  time.Duration // 多信号在时间窗口内
}

// ContextRequirement 上下文需求
type ContextRequirement struct {
	Adapter core.AdapterType
	Purpose string
}

// PatternAction 匹配后执行的动作
type PatternAction struct {
	Type          string // "create_decision" | "advance_status" | "link_entities"
	NewStatus     core.DecisionStatus
	CommitMessage string
}

// PatternMatcher 关联模式匹配器（signal-activation-engine.md §5）
type PatternMatcher struct {
	patterns  []*CorrelationPattern
	signalBuf map[string][]*StateChangeSignal // adapter → signals 缓冲
}

// NewPatternMatcher 创建模式匹配器
func NewPatternMatcher() *PatternMatcher {
	pm := &PatternMatcher{
		signalBuf: make(map[string][]*StateChangeSignal),
	}
	pm.patterns = defaultPatterns()
	return pm
}

// PushSignal 推送信号到缓冲，检查是否有匹配的模式
func (pm *PatternMatcher) PushSignal(signal *StateChangeSignal, assembled *AssembledContext) *PatternMatchResult {
	adapterKey := string(signal.Adapter)
	pm.signalBuf[adapterKey] = append(pm.signalBuf[adapterKey], signal)

	// 清理过期信号（超过 2 小时）
	cutoff := signal.Timestamp.Add(-2 * time.Hour)
	for key, signals := range pm.signalBuf {
		var valid []*StateChangeSignal
		for _, s := range signals {
			if s.Timestamp.After(cutoff) {
				valid = append(valid, s)
			}
		}
		pm.signalBuf[key] = valid
	}

	// 对每个模式检查是否匹配
	for _, p := range pm.patterns {
		if result := pm.matchPattern(p, signal, assembled); result != nil {
			return result
		}
	}

	return nil
}

// matchPattern 检查单个模式是否匹配
func (pm *PatternMatcher) matchPattern(pattern *CorrelationPattern, signal *StateChangeSignal, ctx *AssembledContext) *PatternMatchResult {
	// 简单实现：检查信号是否匹配第一个触发条件
	if len(pattern.TriggerSignals) == 0 {
		return nil
	}

	first := pattern.TriggerSignals[0]
	if signal.Adapter != first.Adapter || signal.Strength < first.MinStrength {
		return nil
	}

	// 检查是否需要其他上下文信号
	for _, req := range pattern.RequiredContext {
		// 检查上下文是否包含所需适配器的数据
		_ = req
	}

	return &PatternMatchResult{
		PatternID:   pattern.PatternID,
		PatternName: pattern.Name,
		Confidence:  pattern.ConfidenceThreshold,
		Action:      pattern.Action,
	}
}

// PatternMatchResult 模式匹配结果
type PatternMatchResult struct {
	PatternID   string
	PatternName string
	Confidence  float64
	Action      PatternAction
}

// defaultPatterns 5 种跨适配器关联模式（signal-activation-engine.md §5 表）
func defaultPatterns() []*CorrelationPattern {
	return []*CorrelationPattern{
		{
			PatternID: "im_meeting_decision",
			Name:      "IM + 会议 → 决策",
			TriggerSignals: []SignalPattern{
				{Adapter: core.AdapterIM, MinStrength: core.SignalMedium, TimeWindow: 2 * time.Hour},
			},
			RequiredContext: []ContextRequirement{
				{Adapter: core.AdapterVC, Purpose: "获取 ±2h 内的会议纪要"},
			},
			Action: PatternAction{
				Type: "create_decision",
			},
			ConfidenceThreshold: 0.7,
		},
		{
			PatternID: "task_milestone_progress",
			Name:      "任务 + 里程碑 → 进度",
			TriggerSignals: []SignalPattern{
				{Adapter: core.AdapterTask, MinStrength: core.SignalStrong, TimeWindow: 1 * time.Hour},
			},
			RequiredContext: []ContextRequirement{
				{Adapter: core.AdapterCalendar, Purpose: "获取 ±1d 内的里程碑"},
			},
			Action: PatternAction{
				Type: "advance_status",
			},
			ConfidenceThreshold: 0.7,
		},
		{
			PatternID: "doc_approval_im_confirm",
			Name:      "文档审批 + IM → 确认",
			TriggerSignals: []SignalPattern{
				{Adapter: core.AdapterDocs, MinStrength: core.SignalStrong, TimeWindow: 1 * time.Hour},
			},
			RequiredContext: []ContextRequirement{
				{Adapter: core.AdapterIM, Purpose: "获取近期 IM 讨论"},
			},
			Action: PatternAction{
				Type:      "advance_status",
				NewStatus: core.StatusDecided,
			},
			ConfidenceThreshold: 0.8,
		},
		{
			PatternID: "meeting_create_tasks",
			Name:      "会议结束 → 创建任务",
			TriggerSignals: []SignalPattern{
				{Adapter: core.AdapterVC, MinStrength: core.SignalStrong, TimeWindow: 30 * time.Minute},
			},
			RequiredContext: []ContextRequirement{
				{Adapter: core.AdapterVC, Purpose: "获取 AI 纪要待办列表"},
			},
			Action: PatternAction{
				Type: "create_decision",
			},
			ConfidenceThreshold: 0.9,
		},
		{
			PatternID: "okr_task_alignment",
			Name:      "OKR + 任务 → 对齐",
			TriggerSignals: []SignalPattern{
				{Adapter: core.AdapterTask, MinStrength: core.SignalStrong, TimeWindow: 2 * time.Hour},
			},
			RequiredContext: []ContextRequirement{
				{Adapter: core.AdapterOKR, Purpose: "获取 KR 进度"},
			},
			Action: PatternAction{
				Type: "link_entities",
			},
			ConfidenceThreshold: 0.6,
		},
	}
}
