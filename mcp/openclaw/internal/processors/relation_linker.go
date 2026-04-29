package processors

import (
	"github.com/openclaw/internal/core"
)

// RelationLinker 关联推理器（openclaw-architecture.md §2.1.1 Stage 2c）
// 根据决策内容推测关联的飞书实体（文档、人员、日程）
type RelationLinker struct {
}

// NewRelationLinker 创建关联推理器
func NewRelationLinker() *RelationLinker {
	return &RelationLinker{}
}

// LinkRelations 从决策内容中提取关联实体
func (rl *RelationLinker) LinkRelations(decision *core.DecisionNode, rawData interface{}) {
	// 提取文档链接（根据 URL 模式）
	// 提取 @提及的人员
	// 提取关联日程关键词
	// 这些由 LLM 在外部完成，框架中预留接口

	_ = rawData
}

// CrossTopicRefs 判断决策是否需要跨议题引用（decision-tree.md §4.5）
// 返回受影响的其他议题列表
func (rl *RelationLinker) CrossTopicRefs(decision *core.DecisionNode, allTopics []string) []string {
	// 默认不跨议题
	// LLM 在外部根据以下信号判定：
	// 1. 决策类型（基础设施/架构类 → 倾向跨议题）
	// 2. 内容中提及的其他议题模块名
	// 3. 受影响的人员（stakeholders 包含其他议题负责人）
	// 4. 技术依赖方向（API 变更 → 所有调用方）

	return nil
}
