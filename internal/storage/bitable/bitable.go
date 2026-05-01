package bitable

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"feishu-mem/internal/decision"
	larkadapter "feishu-mem/internal/lark-adapter"
)

// Config Bitable 配置
type Config struct {
	BaseToken string       `json:"base_token"`
	Tables    TablesConfig `json:"tables"`
}

// TablesConfig 表配置
type TablesConfig struct {
	Decision string `json:"decision"`
	Topic    string `json:"topic"`
	Phase    string `json:"phase"`
	Relation string `json:"relation"`
}

// BitableStore 飞书多维表格存储
type BitableStore struct {
	cli    *larkadapter.LarkCLI
	config Config
}

// NewBitableStore 创建 Bitable 存储
func NewBitableStore(config Config, cli *larkadapter.LarkCLI) *BitableStore {
	return &BitableStore{
		config: config,
		cli:    cli,
	}
}

// UpsertDecision 插入或更新决策记录
func (bs *BitableStore) UpsertDecision(node *decision.DecisionNode) error {
	if bs.config.BaseToken == "" || bs.config.Tables.Decision == "" {
		return fmt.Errorf("bitable config not set: base_token or decision table missing")
	}

	fields := bs.nodeToFieldValues(node)
	payload, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	log.Printf("[Bitable] Upsert payload: %s", string(payload))

	// 使用 lark-cli 执行 upsert
	_, err = bs.cli.RunCommand(
		"base", "+record-upsert",
		"--base-token", bs.config.BaseToken,
		"--table-id", bs.config.Tables.Decision,
		"--json", string(payload),
	)
	return err
}

// QueryByTopic 按主题查询
func (bs *BitableStore) QueryByTopic(topic, status string) ([]*decision.DecisionNode, error) {
	if bs.config.BaseToken == "" || bs.config.Tables.Decision == "" {
		// 返回空结果而不是错误，方便降级处理
		return []*decision.DecisionNode{}, nil
	}

	// 构造过滤条件
	filter := fmt.Sprintf(`CurrentValue.[Topic] = "%s"`, topic)
	if status != "" {
		filter = fmt.Sprintf(`%s && CurrentValue.[Status] = "%s"`, filter, status)
	}

	var result BitableQueryResponse
	err := bs.cli.RunCommandJSON(&result,
		"base", "+record-list",
		"--base-token", bs.config.BaseToken,
		"--table-id", bs.config.Tables.Decision,
		"--filter", filter,
	)
	if err != nil {
		return []*decision.DecisionNode{}, err
	}

	return bs.recordsToDecisions(result.Items), nil
}

// QueryCrossTopic 跨主题查询
func (bs *BitableStore) QueryCrossTopic(topic string) ([]*decision.DecisionNode, error) {
	if bs.config.BaseToken == "" || bs.config.Tables.Decision == "" {
		return []*decision.DecisionNode{}, nil
	}

	// 查询引用了该主题的决策
	filter := fmt.Sprintf(`CurrentValue.[CrossTopicRefs] includes "%s"`, topic)

	var result BitableQueryResponse
	err := bs.cli.RunCommandJSON(&result,
		"base", "+record-list",
		"--base-token", bs.config.BaseToken,
		"--table-id", bs.config.Tables.Decision,
		"--filter", filter,
	)
	if err != nil {
		return []*decision.DecisionNode{}, err
	}

	return bs.recordsToDecisions(result.Items), nil
}

// QueryByPhase 按阶段查询
func (bs *BitableStore) QueryByPhase(phase string) ([]*decision.DecisionNode, error) {
	if bs.config.BaseToken == "" || bs.config.Tables.Decision == "" {
		return []*decision.DecisionNode{}, nil
	}

	filter := fmt.Sprintf(`CurrentValue.[Phase] = "%s"`, phase)

	var result BitableQueryResponse
	err := bs.cli.RunCommandJSON(&result,
		"base", "+record-list",
		"--base-token", bs.config.BaseToken,
		"--table-id", bs.config.Tables.Decision,
		"--filter", filter,
	)
	if err != nil {
		return []*decision.DecisionNode{}, err
	}

	return bs.recordsToDecisions(result.Items), nil
}

// ListTopics 列出所有主题
func (bs *BitableStore) ListTopics() ([]TopicDef, error) {
	if bs.config.BaseToken == "" || bs.config.Tables.Topic == "" {
		return []TopicDef{}, nil
	}

	var result BitableQueryResponse
	err := bs.cli.RunCommandJSON(&result,
		"base", "+record-list",
		"--base-token", bs.config.BaseToken,
		"--table-id", bs.config.Tables.Topic,
	)
	if err != nil {
		return []TopicDef{}, err
	}

	var topics []TopicDef
	for _, item := range result.Items {
		topics = append(topics, TopicDef{
			Name:        getStringField(item.Fields, "Name"),
			Description: getStringField(item.Fields, "Description"),
		})
	}
	return topics, nil
}

// SearchContent 全文搜索 Bitable 记录
func (bs *BitableStore) SearchContent(query string, topic string) ([]*decision.DecisionNode, error) {
	if bs.config.BaseToken == "" || bs.config.Tables.Decision == "" {
		return []*decision.DecisionNode{}, nil
	}

	var filter string
	if topic != "" {
		filter = fmt.Sprintf(`CurrentValue.[Topic] = "%s"`, topic)
	}

	var result BitableQueryResponse
	args := []string{
		"base", "+record-list",
		"--base-token", bs.config.BaseToken,
		"--table-id", bs.config.Tables.Decision,
	}
	if filter != "" {
		args = append(args, "--filter", filter)
	}

	err := bs.cli.RunCommandJSON(&result, args...)
	if err != nil {
		return []*decision.DecisionNode{}, err
	}

	// 在内存中进行关键词过滤
	var filtered []*decision.DecisionNode
	for _, d := range bs.recordsToDecisions(result.Items) {
		if query == "" {
			filtered = append(filtered, d)
			continue
		}
		// 简单关键词匹配
		if containsIgnoreCase(d.Title, query) ||
			containsIgnoreCase(d.Decision, query) ||
			containsIgnoreCase(d.Rationale, query) ||
			containsIgnoreCase(d.Topic, query) {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}

// ListAllDecisions 列出所有决策
func (bs *BitableStore) ListAllDecisions() ([]*decision.DecisionNode, error) {
	if bs.config.BaseToken == "" || bs.config.Tables.Decision == "" {
		return []*decision.DecisionNode{}, nil
	}

	var result BitableQueryResponse
	err := bs.cli.RunCommandJSON(&result,
		"base", "+record-list",
		"--base-token", bs.config.BaseToken,
		"--table-id", bs.config.Tables.Decision,
	)
	if err != nil {
		return []*decision.DecisionNode{}, err
	}

	return bs.recordsToDecisions(result.Items), nil
}

// TopicDef 主题定义
type TopicDef struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// BitableQueryResponse Bitable 查询响应
type BitableQueryResponse struct {
	Items []BitableRecord `json:"items"`
}

// BitableRecord Bitable 记录
type BitableRecord struct {
	RecordID string         `json:"record_id"`
	Fields   map[string]any `json:"fields"`
}

// nodeToFieldValues 将决策节点转换为字段值（使用表中存在的字段名）
func (bs *BitableStore) nodeToFieldValues(node *decision.DecisionNode) map[string]any {
	fields := map[string]any{
		"sdr_id":          node.SDRID,
		"title":           node.Title,
		"topic":           node.Topic,
		"status":          string(node.Status),
		"impact_level":    string(node.ImpactLevel),
		"decision":        node.Decision,
		"proposer":        node.Proposer,
		"executor":        node.Executor,
		"git_commit_hash": node.GitCommitHash,
		"created_at":      node.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	if len(node.CrossTopicRefs) > 0 {
		fields["CrossTopicRefs"] = node.CrossTopicRefs
	}

	if node.DecidedAt != nil {
		fields["DecidedAt"] = node.DecidedAt.Format(time.RFC3339)
	}

	// 飞书关联
	if len(node.FeishuLinks.RelatedChatIDs) > 0 {
		fields["RelatedChatIDs"] = node.FeishuLinks.RelatedChatIDs
	}
	if len(node.FeishuLinks.RelatedDocTokens) > 0 {
		fields["RelatedDocTokens"] = node.FeishuLinks.RelatedDocTokens
	}

	return fields
}

// recordsToDecisions 将 Bitable 记录转换为决策节点
func (bs *BitableStore) recordsToDecisions(records []BitableRecord) []*decision.DecisionNode {
	var decisions []*decision.DecisionNode
	for _, rec := range records {
		d := bs.recordToDecision(rec)
		if d != nil {
			decisions = append(decisions, d)
		}
	}
	return decisions
}

// recordToDecision 将单条记录转换为决策节点
func (bs *BitableStore) recordToDecision(rec BitableRecord) *decision.DecisionNode {
	fields := rec.Fields

	d := &decision.DecisionNode{
		SDRID:         getStringField(fields, "sdr_id"),
		GitCommitHash: getStringField(fields, "git_commit_hash"),
		Title:         getStringField(fields, "title"),
		Decision:      getStringField(fields, "decision"),
		Topic:         getStringField(fields, "topic"),
		ImpactLevel:   decision.ImpactLevel(getStringField(fields, "impact_level")),
		Status:        decision.DecisionStatus(getStringField(fields, "status")),
		Proposer:      getStringField(fields, "proposer"),
		Executor:      getStringField(fields, "executor"),
	}

	// 解析时间
	if createdAt := getStringField(fields, "created_at"); createdAt != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
			d.CreatedAt = t
		}
	}

	// 解析数组字段
	d.Stakeholders = getStringArrayField(fields, "Stakeholders")
	d.CrossTopicRefs = getStringArrayField(fields, "CrossTopicRefs")

	// 飞书链接
	d.FeishuLinks = decision.FeishuLinks{
		RelatedChatIDs:    getStringArrayField(fields, "RelatedChatIDs"),
		RelatedDocTokens:  getStringArrayField(fields, "RelatedDocTokens"),
		RelatedEventIDs:   getStringArrayField(fields, "RelatedEventIDs"),
		RelatedMeetingIDs: getStringArrayField(fields, "RelatedMeetingIDs"),
	}

	return d
}

func getStringField(fields map[string]any, key string) string {
	if v, ok := fields[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getStringArrayField(fields map[string]any, key string) []string {
	if v, ok := fields[key]; ok {
		if arr, ok := v.([]any); ok {
			var result []string
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return []string{}
}

// ===== 接口定义 =====

// GitReader Git 读取接口
type GitReader interface {
	ListDecisions(project, topic string) ([]*decision.DecisionNode, error)
	ListTopics(project string) ([]string, error)
}

// GitWriter Git 写入接口
type GitWriter interface {
	WriteDecision(node *decision.DecisionNode) (string, error)
}

// GitHashReader Git 哈希读取接口
type GitHashReader interface {
	GetFileHash(project, topic, sdrID string) (string, error)
}

// SyncDrift 同步漂移记录
type SyncDrift struct {
	SDRID       string    `json:"sdr_id"`
	BitableHash string    `json:"bitable_hash"`
	GitHash     string    `json:"git_hash"`
	DetectedAt  time.Time `json:"detected_at"`
}

// ===== 辅助函数 =====

func stringsToLower(s string) string {
	var res []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		res = append(res, c)
	}
	return string(res)
}

func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	ls := stringsToLower(s)
	lsub := stringsToLower(substr)
	return stringsContains(ls, lsub)
}

func stringsContains(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
