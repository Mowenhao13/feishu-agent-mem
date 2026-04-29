package adapters

import (
	"context"
	"github.com/openclaw/internal/core"
)

// BitableStore Bitable 封装 — 结构化查询接口（openclaw-architecture.md §2.4）
type BitableStore struct {
	cli       *LarkCli
	baseToken string
	tables    BitableTableIDs
}

// BitableTableIDs 表 ID 配置
type BitableTableIDs struct {
	Decision string
	Topic    string
	Phase    string
	Relation string
}

// NewBitableStore 创建 BitableStore
func NewBitableStore(cli *LarkCli, baseToken string, tables BitableTableIDs) *BitableStore {
	return &BitableStore{
		cli:       cli,
		baseToken: baseToken,
		tables:    tables,
	}
}

// UpsertDecision 写入/更新决策记录（openclaw-architecture.md §2.4）
func (bs *BitableStore) UpsertDecision(ctx context.Context, node *core.DecisionNode) error {
	fieldValues := map[string]interface{}{
		"sdr_id":           node.SDRID,
		"title":            node.Title,
		"topic":            node.Topic,
		"status":           node.Status.String(),
		"impact_level":     node.ImpactLevel.String(),
		"phase":            node.Phase,
		"phase_scope":      node.PhaseScope.String(),
		"proposer":         node.Proposer,
		"executor":         node.Executor,
		"cross_topic_refs": joinStrings(node.CrossTopicRefs),
		"git_commit_hash":  node.GitCommitHash,
		"created_at":       node.CreatedAt.Format("2006-01-02"),
	}

	if node.DecidedAt != nil {
		fieldValues["decided_at"] = node.DecidedAt.Format("2006-01-02")
	}

	// 飞书关联字段
	fieldValues["related_chats"] = joinStrings(node.FeishuLinks.RelatedChatIDs)
	fieldValues["related_docs"] = joinStrings(node.FeishuLinks.RelatedDocTokens)
	fieldValues["related_meetings"] = joinStrings(node.FeishuLinks.RelatedMeetingIDs)
	fieldValues["related_tasks"] = joinStrings(node.FeishuLinks.RelatedTaskGUIDs)

	// 实际实现是通过 lark-cli 调用 bitable API:
	// lark-cli base +record-upsert \
	//   --base-token <token> \
	//   --table-id <decision_table> \
	//   --field-values 'json...'
	_ = fieldValues
	_ = ctx

	return nil
}

// QueryByTopic 按议题查询（openclaw-architecture.md §2.4）
func (bs *BitableStore) QueryByTopic(ctx context.Context, topic, status string) ([]*core.DecisionNode, error) {
	// 实际实现:
	// lark-cli base +record-list \
	//   --base-token <token> \
	//   --table-id <decision_table> \
	//   --filter '{"conjunction":"and","conditions":[...]}'
	_ = ctx
	return nil, nil
}

// QueryCrossTopic 跨议题查询（含 cross_topic_refs CONTAINS）（openclaw-architecture.md §2.4）
func (bs *BitableStore) QueryCrossTopic(ctx context.Context, topic string) ([]*core.DecisionNode, error) {
	// Step 1: 同议题查询
	sameTopic, err := bs.QueryByTopic(ctx, topic, "active")
	if err != nil {
		return nil, err
	}

	// Step 2: cross_topic_refs CONTAINS topic
	// lark-cli base +record-search \
	//   --base-token <token> \
	//   --table-id <decision_table> \
	//   --field-name cross_topic_refs \
	//   --contains <topic>
	_ = sameTopic

	return nil, nil
}

// QueryByPhase 按阶段过滤
func (bs *BitableStore) QueryByPhase(ctx context.Context, phase string) ([]*core.DecisionNode, error) {
	_ = ctx
	return nil, nil
}

// ListTopics 获取所有议题列表
func (bs *BitableStore) ListTopics(ctx context.Context) ([]string, error) {
	_ = ctx
	return nil, nil
}

// VerifyConsistency Git ↔ Bitable 一致性校验（openclaw-architecture.md §3.3）
func (bs *BitableStore) VerifyConsistency(ctx context.Context, gitStorage *GitStorage) ([]SyncDrift, error) {
	_ = ctx
	return nil, nil
}

// SyncDrift 同步漂移记录
type SyncDrift struct {
	SDRID       string `json:"sdr_id"`
	BitableHash string `json:"bitable_hash"`
	GitHash     string `json:"git_hash"`
	DriftType   string `json:"drift_type"` // git_ahead | bitable_ahead | conflicted
}

// joinStrings 将字符串切片合并为逗号分隔
func joinStrings(s []string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ", "
		}
		result += v
	}
	return result
}
