package signal

import (
	"github.com/openclaw/internal/core"
)

// DecisionStateMachine 决策状态机（signal-activation-engine.md §4）
type DecisionStateMachine struct {
	memory *core.MemoryGraph
	rules  []TransitionRule
}

// TransitionRule 转换规则（signal-activation-engine.md §4.2）
type TransitionRule struct {
	CurrentStatus core.DecisionStatus
	Adapter       core.AdapterType
	MinStrength   core.SignalStrength
	Condition     string // 额外条件描述
	NewStatus     core.DecisionStatus
}

// StateTransition 状态转换结果
type StateTransition struct {
	DecisionID    string              `json:"decision_id"`
	OldStatus     core.DecisionStatus `json:"old_status"`
	NewStatus     core.DecisionStatus `json:"new_status"`
	TriggerSignal *StateChangeSignal  `json:"trigger_signal"`
	Action        string              `json:"action"` // create | update | status_change
}

// NewDecisionStateMachine 创建状态机
func NewDecisionStateMachine(mg *core.MemoryGraph) *DecisionStateMachine {
	sm := &DecisionStateMachine{
		memory: mg,
	}
	sm.rules = defaultTransitionRules()
	return sm
}

// EvaluateTransition 评估状态转换（signal-activation-engine.md §4.3）
func (sm *DecisionStateMachine) EvaluateTransition(current *core.DecisionNode, signal *StateChangeSignal) *StateTransition {
	for _, rule := range sm.rules {
		if rule.CurrentStatus == current.Status &&
			rule.Adapter == signal.Adapter &&
			signal.Strength >= rule.MinStrength {
			// 检查额外条件
			if !sm.evaluateCondition(rule.Condition, current, signal) {
				continue
			}
			return &StateTransition{
				DecisionID:    current.SDRID,
				OldStatus:     current.Status,
				NewStatus:     rule.NewStatus,
				TriggerSignal: signal,
				Action:        "status_change",
			}
		}
	}
	return nil
}

// EvaluateNewDecision 评估是否需要创建新决策
func (sm *DecisionStateMachine) EvaluateNewDecision(signal *StateChangeSignal) *StateTransition {
	// IM 信号含决策关键词 → 创建新决策候选
	if signal.Adapter == core.AdapterIM && signal.Strength >= core.SignalMedium {
		return &StateTransition{
			NewStatus:     core.StatusPending,
			TriggerSignal: signal,
			Action:        "create",
		}
	}

	// VC 会议结束 + 有 AI 待办 → 创建新决策
	if signal.Adapter == core.AdapterVC && signal.Strength == core.SignalStrong {
		return &StateTransition{
			NewStatus:     core.StatusPending,
			TriggerSignal: signal,
			Action:        "create",
		}
	}

	return nil
}

// evaluateCondition 评估额外条件
func (sm *DecisionStateMachine) evaluateCondition(condition string, current *core.DecisionNode, signal *StateChangeSignal) bool {
	switch condition {
	case "all_tasks_completed":
		// 检查决策关联的所有任务是否全部完成—需外部提供
		return true
	case "has_ai_todos":
		// 检查会议纪要是否有待办
		return true
	case "has_approval_comment":
		// 检查文档评论是否含审批
		return true
	default:
		return true
	}
}

// defaultTransitionRules 默认转换规则（signal-activation-engine.md §4.2）
func defaultTransitionRules() []TransitionRule {
	return []TransitionRule{
		// 待决策 → 决策中
		{StatusPending, core.AdapterIM, core.SignalMedium, "", StatusInDiscussion},
		{StatusPending, core.AdapterCalendar, core.SignalMedium, "", StatusInDiscussion},

		// 决策中 → 已决策
		{StatusInDiscussion, core.AdapterDocs, core.SignalStrong, "has_approval_comment", StatusDecided},
		{StatusInDiscussion, core.AdapterVC, core.SignalStrong, "has_ai_todos", StatusDecided},
		{StatusInDiscussion, core.AdapterIM, core.SignalStrong, "", StatusDecided},

		// 已决策 → 执行中
		{StatusDecided, core.AdapterTask, core.SignalMedium, "", StatusExecuting},

		// 执行中 → 已完成
		{StatusExecuting, core.AdapterTask, core.SignalStrong, "all_tasks_completed", StatusCompleted},
	}
}
