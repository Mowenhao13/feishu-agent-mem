package signal

import (
	"github.com/openclaw/internal/core"
)

// ---- 8 个适配器的信号发射器（signal-activation-engine.md §1 信号强度规则）----

// IMEmitter IM 消息发射器
type IMEmitter struct{}

func (e *IMEmitter) AdapterType() core.AdapterType { return core.AdapterIM }

// EmitSignal 根据 IM 消息内容判定信号强度
func (e *IMEmitter) EmitSignal(event *core.LarkEvent) *StateChangeSignal {
	text := extractMessageText(event)

	// 检测决策关键词
	matched, keyword := ContainsDecisionKeyword(text)
	if !matched {
		return nil
	}

	// 统计匹配的关键词数量
	keywordCount := countDecisionKeywords(text)

	strength := core.SignalWeak
	if keywordCount >= 2 || isThreadApproval(text) {
		strength = core.SignalStrong
	} else if keywordCount >= 1 {
		strength = core.SignalMedium
	}

	return &StateChangeSignal{
		SignalID:      "sig_im_" + event.Header.EventID,
		Adapter:       core.AdapterIM,
		Timestamp:     now(),
		EventType:     event.Type,
		ChangeType:    core.ChangeCreated,
		Strength:      strength,
		ChangeSummary: "群聊消息包含决策关键词: " + keyword,
		Context: SignalContext{
			Keywords:        []string{keyword},
			DecisionSignals: []string{keyword},
			ContentSnippet:  truncate(text, 500),
		},
	}
}

// VCEmitter 会议发射器
type VCEmitter struct{}

func (e *VCEmitter) AdapterType() core.AdapterType { return core.AdapterVC }

// EmitSignal 会议结束信号（强信号）
func (e *VCEmitter) EmitSignal(event *core.LarkEvent) *StateChangeSignal {
	return &StateChangeSignal{
		SignalID:      "sig_vc_" + event.Header.EventID,
		Adapter:       core.AdapterVC,
		Timestamp:     now(),
		EventType:     event.Type,
		ChangeType:    core.ChangeCreated,
		Strength:      core.SignalStrong, // 会议结束始终是强信号
		ChangeSummary: "会议结束",
	}
}

// DocsEmitter 文档发射器
type DocsEmitter struct{}

func (e *DocsEmitter) AdapterType() core.AdapterType { return core.AdapterDocs }

// EmitSignal 根据文档变更类型判定信号强度
func (e *DocsEmitter) EmitSignal(event *core.LarkEvent) *StateChangeSignal {
	switch event.Type {
	case "drive.file.comment_add_v1":
		// 审批评论
		return &StateChangeSignal{
			SignalID:      "sig_docs_" + event.Header.EventID,
			Adapter:       core.AdapterDocs,
			Timestamp:     now(),
			EventType:     event.Type,
			ChangeType:    core.ChangeUpdated,
			Strength:      core.SignalStrong,
			ChangeSummary: "文档审批评论",
		}
	case "drive.file.created_v1":
		return &StateChangeSignal{
			SignalID:      "sig_docs_" + event.Header.EventID,
			Adapter:       core.AdapterDocs,
			Timestamp:     now(),
			EventType:     event.Type,
			ChangeType:    core.ChangeCreated,
			Strength:      core.SignalWeak,
			ChangeSummary: "新文档创建",
		}
	default:
		return &StateChangeSignal{
			SignalID:      "sig_docs_" + event.Header.EventID,
			Adapter:       core.AdapterDocs,
			Timestamp:     now(),
			EventType:     event.Type,
			ChangeType:    core.ChangeUpdated,
			Strength:      core.SignalWeak,
			ChangeSummary: "文档更新",
		}
	}
}

// CalendarEmitter 日程发射器
type CalendarEmitter struct{}

func (e *CalendarEmitter) AdapterType() core.AdapterType { return core.AdapterCalendar }

// EmitSignal 检测日程中是否含"评审"/"决策"关键词
func (e *CalendarEmitter) EmitSignal(event *core.LarkEvent) *StateChangeSignal {
	text := extractMessageText(event)

	if containsDecisionKeyword(text) {
		return &StateChangeSignal{
			SignalID:      "sig_cal_" + event.Header.EventID,
			Adapter:       core.AdapterCalendar,
			Timestamp:     now(),
			EventType:     event.Type,
			ChangeType:    core.ChangeCreated,
			Strength:      core.SignalMedium,
			ChangeSummary: "含决策关键词的日程",
		}
	}

	return &StateChangeSignal{
		SignalID:      "sig_cal_" + event.Header.EventID,
		Adapter:       core.AdapterCalendar,
		Timestamp:     now(),
		EventType:     event.Type,
		ChangeType:    core.ChangeUpdated,
		Strength:      core.SignalWeak,
		ChangeSummary: "日程变更",
	}
}

// TaskEmitter 任务发射器
type TaskEmitter struct{}

func (e *TaskEmitter) AdapterType() core.AdapterType { return core.AdapterTask }

// EmitSignal 任务完成是强信号
func (e *TaskEmitter) EmitSignal(event *core.LarkEvent) *StateChangeSignal {
	strength := core.SignalMedium
	changeSummary := "任务变更"

	// 检测是否为 completed 状态
	if isTaskCompleted(event) {
		strength = core.SignalStrong
		changeSummary = "任务完成"
	}

	return &StateChangeSignal{
		SignalID:      "sig_task_" + event.Header.EventID,
		Adapter:       core.AdapterTask,
		Timestamp:     now(),
		EventType:     event.Type,
		ChangeType:    core.ChangeStatus,
		Strength:      strength,
		ChangeSummary: changeSummary,
	}
}

// OKREmitter OKR 发射器
type OKREmitter struct{}

func (e *OKREmitter) AdapterType() core.AdapterType { return core.AdapterOKR }

// EmitSignal OKR 变更
func (e *OKREmitter) EmitSignal(event *core.LarkEvent) *StateChangeSignal {
	return &StateChangeSignal{
		SignalID:      "sig_okr_" + event.Header.EventID,
		Adapter:       core.AdapterOKR,
		Timestamp:     now(),
		EventType:     event.Type,
		ChangeType:    core.ChangeUpdated,
		Strength:      core.SignalWeak,
		ChangeSummary: "OKR 变更",
	}
}

// ContactEmitter 通讯录发射器
type ContactEmitter struct{}

func (e *ContactEmitter) AdapterType() core.AdapterType { return core.AdapterContact }

// EmitSignal 人员信息变更是弱信号
func (e *ContactEmitter) EmitSignal(event *core.LarkEvent) *StateChangeSignal {
	return &StateChangeSignal{
		SignalID:      "sig_contact_" + event.Header.EventID,
		Adapter:       core.AdapterContact,
		Timestamp:     now(),
		EventType:     event.Type,
		ChangeType:    core.ChangeUpdated,
		Strength:      core.SignalWeak,
		ChangeSummary: "人员信息变更",
	}
}

// WikiEmitter 知识库发射器
type WikiEmitter struct{}

func (e *WikiEmitter) AdapterType() core.AdapterType { return core.AdapterWiki }

// EmitSignal 知识库变更
func (e *WikiEmitter) EmitSignal(event *core.LarkEvent) *StateChangeSignal {
	return &StateChangeSignal{
		SignalID:      "sig_wiki_" + event.Header.EventID,
		Adapter:       core.AdapterWiki,
		Timestamp:     now(),
		EventType:     event.Type,
		ChangeType:    core.ChangeCreated,
		Strength:      core.SignalMedium,
		ChangeSummary: "知识库新增决策相关节点",
	}
}

// ---- 辅助函数 ----

// extractMessageText 从事件中提取消息文本
func extractMessageText(event *core.LarkEvent) string {
	if event.Raw != nil {
		return string(event.Raw)
	}
	return ""
}

// countDecisionKeywords 统计文本中决策关键词数量
func countDecisionKeywords(text string) int {
	count := 0
	for _, kw := range DecisionKeywords {
		if contains(text, kw) {
			count++
		}
	}
	return count
}

// isThreadApproval 判断是否为线程内审批回复
func isThreadApproval(text string) bool {
	approvalSignals := []string{"/approve", "LGTM", "同意", "批准", "通过"}
	for _, s := range approvalSignals {
		if contains(text, s) {
			return true
		}
	}
	return false
}

// isTaskCompleted 判断任务是否为完成事件
func isTaskCompleted(event *core.LarkEvent) bool {
	// 实际需要解析事件中的 status 字段
	return false
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
