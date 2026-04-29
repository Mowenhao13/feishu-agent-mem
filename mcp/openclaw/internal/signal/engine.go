package signal

import (
	"sync"
	"time"

	"github.com/openclaw/internal/core"
)

// SignalActivationEngine 信号编排器（signal-activation-engine.md §6）
type SignalActivationEngine struct {
	emitters     map[core.AdapterType]StateChangeEmitter
	router       *ActivationRouter
	assembler    *ContextAssembler
	stateMachine *DecisionStateMachine
	patterns     *PatternMatcher
	memory       *core.MemoryGraph

	// 输出回调
	onMutation func(mutation *core.DecisionMutation)

	mu sync.Mutex
}

// StateChangeEmitter 信号发射器接口（signal-activation-engine.md §1）
type StateChangeEmitter interface {
	EmitSignal(event *core.LarkEvent) *StateChangeSignal
	AdapterType() core.AdapterType
}

// ProcessingReport 处理报告
type ProcessingReport struct {
	SignalID     string
	Adapter      core.AdapterType
	Transition   *StateTransition
	TokenUsed    int
	Correlations []CorrelationHint
}

// NewSignalActivationEngine 创建信号激活引擎
func NewSignalActivationEngine(mg *core.MemoryGraph) *SignalActivationEngine {
	return &SignalActivationEngine{
		emitters:     make(map[core.AdapterType]StateChangeEmitter),
		router:       NewActivationRouter(),
		assembler:    NewContextAssembler(mg),
		stateMachine: NewDecisionStateMachine(mg),
		patterns:     NewPatternMatcher(),
		memory:       mg,
	}
}

// RegisterEmitter 注册信号发射器
func (e *SignalActivationEngine) RegisterEmitter(emitter StateChangeEmitter) {
	e.emitters[emitter.AdapterType()] = emitter
}

// SetMutationHandler 设置变更回调
func (e *SignalActivationEngine) SetMutationHandler(handler func(mutation *core.DecisionMutation)) {
	e.onMutation = handler
}

// OnEvent 事件处理入口（signal-activation-engine.md §6）
func (e *SignalActivationEngine) OnEvent(event *core.LarkEvent) *ProcessingReport {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Step 1: 识别适配器，发射信号
	adapter := identifyAdapter(event)
	emitter, ok := e.emitters[adapter]
	if !ok {
		return nil
	}

	signal := emitter.EmitSignal(event)
	if signal == nil {
		return nil // 不相关事件，跳过
	}

	// Step 2: 激活路由—决定查哪些适配器
	plan := e.router.Route(signal)

	// Step 3: 执行查询 + 上下文装配
	context := e.assembler.Assemble(signal, plan)

	// Step 4: 匹配关联模式 + 评估状态转换
	report := &ProcessingReport{
		SignalID:     signal.SignalID,
		Adapter:      adapter,
		TokenUsed:    context.TokenUsed,
		Correlations: context.CorrelationHints,
	}

	// 4a: 尝试匹配已有决策的状态转换
	for _, d := range context.Decisions {
		transition := e.stateMachine.EvaluateTransition(d, signal)
		if transition != nil {
			report.Transition = transition
			e.applyTransition(transition)
			break
		}
	}

	// 4b: 如果没有匹配的状态转换，检查是否要创建新决策
	if report.Transition == nil {
		transition := e.stateMachine.EvaluateNewDecision(signal)
		if transition != nil {
			report.Transition = transition
			e.applyTransition(transition)
		}
	}

	// 4c: 匹配关联模式
	patternResult := e.patterns.PushSignal(signal, context)
	if patternResult != nil && report.Transition == nil {
		// 模式匹配触发创建
		if patternResult.Action.Type == "create_decision" {
			report.Transition = &StateTransition{
				NewStatus:     core.StatusPending,
				TriggerSignal: signal,
				Action:        "create",
			}
			e.applyTransition(report.Transition)
		}
	}

	return report
}

// applyTransition 应用状态转换，通过回调通知外部
func (e *SignalActivationEngine) applyTransition(transition *StateTransition) {
	if e.onMutation == nil {
		return
	}

	mutation := &core.DecisionMutation{
		Type:  transition.Action,
		SDRID: transition.DecisionID,
		Fields: map[string]interface{}{
			"status": transition.NewStatus.String(),
		},
		CommitMessage: "状态变更: " + transition.OldStatus.String() + " → " + transition.NewStatus.String(),
	}

	e.onMutation(mutation)
}

// identifyAdapter 从事件中识别适配器类型
func identifyAdapter(event *core.LarkEvent) core.AdapterType {
	switch event.Type {
	case "im.message.receive_v1", "im.message.pin", "im.message.unpin":
		return core.AdapterIM
	case "vc.meeting.meeting_ended":
		return core.AdapterVC
	case "drive.file.created_v1", "drive.file.updated_v1", "drive.file.comment_add_v1":
		return core.AdapterDocs
	case "calendar.event.created", "calendar.event.updated", "calendar.event.deleted":
		return core.AdapterCalendar
	case "task.task.updated":
		return core.AdapterTask
	case "wiki.node.created", "wiki.node.updated":
		return core.AdapterWiki
	default:
		return core.AdapterIM
	}
}

// ---- 内置信号发射器 ----

// IM Emitter
type IMEmitter struct{}

func (e *IMEmitter) AdapterType() core.AdapterType { return core.AdapterIM }
func (e *IMEmitter) EmitSignal(event *core.LarkEvent) *StateChangeSignal {
	// 简化实现
	return &StateChangeSignal{
		SignalID:      "sig_" + event.Header.EventID,
		Adapter:       core.AdapterIM,
		Timestamp:     now(),
		EventType:     event.Type,
		ChangeType:    core.ChangeCreated,
		Strength:      core.SignalMedium,
		ChangeSummary: "群聊消息",
	}
}

// VC Emitter
type VCEmitter struct{}

func (e *VCEmitter) AdapterType() core.AdapterType { return core.AdapterVC }
func (e *VCEmitter) EmitSignal(event *core.LarkEvent) *StateChangeSignal {
	return &StateChangeSignal{
		SignalID:      "sig_" + event.Header.EventID,
		Adapter:       core.AdapterVC,
		Timestamp:     now(),
		EventType:     event.Type,
		ChangeType:    core.ChangeCreated,
		Strength:      core.SignalStrong,
		ChangeSummary: "会议结束",
	}
}

// Docs Emitter
type DocsEmitter struct{}

func (e *DocsEmitter) AdapterType() core.AdapterType { return core.AdapterDocs }
func (e *DocsEmitter) EmitSignal(event *core.LarkEvent) *StateChangeSignal {
	return &StateChangeSignal{
		SignalID:      "sig_" + event.Header.EventID,
		Adapter:       core.AdapterDocs,
		Timestamp:     now(),
		EventType:     event.Type,
		ChangeType:    core.ChangeUpdated,
		Strength:      core.SignalMedium,
		ChangeSummary: "文档变更",
	}
}

func now() time.Time {
	return time.Now()
}
