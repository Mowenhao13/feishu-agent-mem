package extractors

import (
	"context"

	"github.com/openclaw/internal/adapters"
	"github.com/openclaw/internal/core"
)

// DocExtractor 云文档提取器（decision-extraction.md §4.3）
type DocExtractor struct {
	cli *adapters.LarkCli
}

// NewDocExtractor 创建文档提取器
func NewDocExtractor(cli *adapters.LarkCli) *DocExtractor {
	return &DocExtractor{cli: cli}
}

// ExtractFromComments 从文档评论中提取决策信号
func (e *DocExtractor) ExtractFromComments(ctx context.Context, docToken string) ([]*core.DecisionNode, error) {
	comments, err := e.cli.GetDocComments(ctx, docToken, "docx")
	if err != nil {
		return nil, err
	}

	var signals []*core.DecisionNode
	for _, c := range comments {
		// 检测审批信号
		if containsDecisionKeyword(c.Content) || isApprovalComment(c.Content) {
			signals = append(signals, &core.DecisionNode{
				SDRID:  core.GenerateSDRID(len(signals) + 1),
				Title:  "文档评论: " + truncateString(c.Content, 60),
				Status: core.StatusPending,
				FeishuLinks: core.FeishuLinks{
					RelatedDocTokens: []string{docToken},
				},
			})
		}
	}

	return signals, nil
}

// SearchDecisionDocs 搜索决策相关文档
func (e *DocExtractor) SearchDecisionDocs(ctx context.Context, query string) ([]*adapters.DocRef, error) {
	return e.cli.SearchDocs(ctx, query)
}

// isApprovalComment 检测审批评论
func isApprovalComment(text string) bool {
	approvalSignals := []string{"/approve", "同意", "批准", "通过", "不通过", "需要修改"}
	for _, s := range approvalSignals {
		if contains(text, s) {
			return true
		}
	}
	return false
}
