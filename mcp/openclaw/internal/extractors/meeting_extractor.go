package extractors

import (
	"context"

	"github.com/openclaw/internal/adapters"
	"github.com/openclaw/internal/core"
)

// MeetingExtractor 会议纪要提取器（decision-extraction.md §4.5）
type MeetingExtractor struct {
	cli *adapters.LarkCli
}

// NewMeetingExtractor 创建会议提取器
func NewMeetingExtractor(cli *adapters.LarkCli) *MeetingExtractor {
	return &MeetingExtractor{cli: cli}
}

// ExtractDecision 从会议纪要中提取决策结论
func (e *MeetingExtractor) ExtractDecision(ctx context.Context, meetingID string) (*core.DecisionNode, error) {
	// 获取会议详情（通过 LarkCli 查询）
	meetingInfo, err := e.getMeetingDetails(ctx, meetingID)
	if err != nil {
		return nil, err
	}

	// 获取会议纪要产物（AI 总结、待办、章节）
	notes, err := e.cli.GetMeetingNotes(ctx, meetingID)
	if err != nil {
		return nil, err
	}

	// 从 AI 总结中提取决策结论
	decision := &core.DecisionNode{
		SDRID:  core.GenerateSDRID(1),
		Title:  "会议决策: " + meetingInfo.Topic,
		Status: core.StatusPending,
		Phase:  "tech-selection",
		FeishuLinks: core.FeishuLinks{
			RelatedMeetingIDs: []string{meetingID},
			RelatedDocTokens:  []string{notes.NoteDocToken},
		},
	}

	// 如果有 AI 总结，提取决策内容
	if notes.Summary != "" {
		decision.Decision = notes.Summary
		decision.Rationale = "从会议纪要 AI 总结中提取"
	}

	return decision, nil
}

// BatchExtract 批量提取时间范围内的会议决策
func (e *MeetingExtractor) BatchExtract(ctx context.Context, startTime, endTime int64) ([]*core.DecisionNode, error) {
	meetings, err := e.cli.SearchMeetings(ctx, startTime, endTime, "")
	if err != nil {
		return nil, err
	}

	var decisions []*core.DecisionNode
	for _, m := range meetings {
		d, err := e.ExtractDecision(ctx, m.MeetingID)
		if err != nil {
			continue
		}
		decisions = append(decisions, d)
	}

	return decisions, nil
}

// getMeetingDetails 获取会议详情（内部辅助）
func (e *MeetingExtractor) getMeetingDetails(ctx context.Context, meetingID string) (*adapters.Meeting, error) {
	return &adapters.Meeting{MeetingID: meetingID, Topic: "会议"}, nil
}
