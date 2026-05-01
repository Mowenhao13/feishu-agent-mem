// internal/llm/agent.go
// LLM 模块主入口

package llm

import (
	"context"
	"fmt"
	"time"

	"feishu-mem/internal/llm/budget"
	"feishu-mem/internal/llm/prompts"
	"feishu-mem/internal/llm/tools"
)

// MemoryAgent 记忆系统专用 Agent
type MemoryAgent struct {
	promptMgr *prompts.PromptManager
	tools     *tools.ToolRegistry
	budget    *budget.BudgetTracker
	fallback  *Fallback
	llmClient *Client
}

// NewMemoryAgent 创建 MemoryAgent
func NewMemoryAgent() *MemoryAgent {
	return &MemoryAgent{
		promptMgr: prompts.NewPromptManager(),
		tools:     tools.NewToolRegistry(),
		budget: budget.NewBudgetTracker(budget.DefaultBudget()),
		fallback:  NewFallback(),
		llmClient: NewClient(),
	}
}

// ========== 核心工作流 ==========

// ProcessSignal 处理信号（主入口）
func (a *MemoryAgent) ProcessSignal(sig any) (*DecisionResult, error) {
	return &DecisionResult{
		CreatedAt: time.Now(),
	}, nil
}

// ExtractDecision 提取决策
func (a *MemoryAgent) ExtractDecision(content string, topics []string) (*ExtractionResult, error) {
	// 检查 LLM 是否可用
	if !a.llmClient.IsAvailable() {
		return &ExtractionResult{
			HasDecision:    false,
			Confidence:     0.0,
			ExtractedFrom: content,
		}, fmt.Errorf("ARK_API_KEY is not set")
	}

	// 构建提示词
	systemPrompt, userPrompt, err := a.buildExtractionPrompts(content, topics)
	if err != nil {
		return &ExtractionResult{
			HasDecision:    false,
			Confidence:     0.0,
			ExtractedFrom: content,
		}, err
	}

	// 调用 LLM
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	llmResponse, err := a.llmClient.Call(ctx, systemPrompt, userPrompt)
	if err != nil {
		return &ExtractionResult{
			HasDecision:    false,
			Confidence:     0.0,
			ExtractedFrom: content,
		}, err
	}

	// 解析 LLM 响应
	result, err := ParseExtractionResult(llmResponse)
	if err != nil {
		return &ExtractionResult{
			HasDecision:    false,
			Confidence:     0.0,
			ExtractedFrom: content,
		}, err
	}

	result.ExtractedFrom = content
	return result, nil
}

// ClassifyTopic 分类议题
func (a *MemoryAgent) ClassifyTopic(decision string, topics []string) (*ClassificationResult, error) {
	// 快速路径：关键词匹配
	quickResult := a.fallback.ClassifyTopic(decision, topics)
	if quickResult.Topic != "" {
		return quickResult, nil
	}

	// 检查 LLM 是否可用
	if !a.llmClient.IsAvailable() {
		return quickResult, nil
	}

	// 构建提示词
	systemPrompt, userPrompt, err := a.buildClassificationPrompts(decision, topics)
	if err != nil {
		return quickResult, err
	}

	// 调用 LLM
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	llmResponse, err := a.llmClient.Call(ctx, systemPrompt, userPrompt)
	if err != nil {
		return quickResult, nil
	}

	// 解析 LLM 响应
	result, err := ParseClassificationResult(llmResponse)
	if err != nil {
		return quickResult, nil
	}

	return result, nil
}

// DetectCrossTopic 检测跨议题
func (a *MemoryAgent) DetectCrossTopic(node any) (*CrossTopicResult, error) {
	// 快速路径：降级策略
	quickResult := a.fallback.DetectCrossTopic(node)

	// 检查 LLM 是否可用
	if !a.llmClient.IsAvailable() {
		return quickResult, nil
	}

	// 构建提示词
	systemPrompt, userPrompt, err := a.buildCrossTopicPrompts(node)
	if err != nil {
		return quickResult, err
	}

	// 调用 LLM
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	llmResponse, err := a.llmClient.Call(ctx, systemPrompt, userPrompt)
	if err != nil {
		return quickResult, nil
	}

	// 解析 LLM 响应
	result, err := ParseCrossTopicResult(llmResponse)
	if err != nil {
		return quickResult, nil
	}

	return result, nil
}

// ResolveConflict 解决冲突
func (a *MemoryAgent) ResolveConflict(nodeA, nodeB any) (*ConflictResult, error) {
	// 默认结果
	result := &ConflictResult{
		ContradictionScore: 0.0,
		ContradictionType:  "none",
		Description:        "",
		Action:             "no_conflict",
		NeedsUser:          false,
	}

	// 检查 LLM 是否可用
	if !a.llmClient.IsAvailable() {
		return result, nil
	}

	// 构建提示词
	systemPrompt, userPrompt, err := a.buildConflictPrompts(nodeA, nodeB)
	if err != nil {
		return result, nil
	}

	// 调用 LLM
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	llmResponse, err := a.llmClient.Call(ctx, systemPrompt, userPrompt)
	if err != nil {
		return result, nil
	}

	// 解析 LLM 响应
	llmResult, err := ParseConflictResult(llmResponse)
	if err != nil {
		return result, nil
	}

	return llmResult, nil
}

// IsAvailable 检查 LLM 是否可用
func (a *MemoryAgent) IsAvailable() bool {
	return a.llmClient.IsAvailable()
}

// GetTools 获取工具列表
func (a *MemoryAgent) GetTools() []*tools.ToolHint {
	return a.tools.GetAllHints()
}

// SearchTools 搜索工具
func (a *MemoryAgent) SearchTools(query string) []*tools.ToolHint {
	return a.tools.SearchTools(query)
}

// ========== 内部方法 ==========

func (a *MemoryAgent) buildExtractionPrompts(content string, topics []string) (string, string, error) {
	systemPrompt := prompts.ExtractionStaticPrompt

	userPrompt, err := a.promptMgr.BuildPrompt("extraction", map[string]any{
		"content": content,
		"topics":  topics,
	})
	if err != nil {
		return "", "", err
	}

	return systemPrompt, userPrompt, nil
}

func (a *MemoryAgent) buildClassificationPrompts(decision string, topics []string) (string, string, error) {
	systemPrompt := prompts.ClassificationStaticPrompt

	userPrompt, err := a.promptMgr.BuildPrompt("classification", map[string]any{
		"decision": decision,
		"topics":   topics,
	})
	if err != nil {
		return "", "", err
	}

	return systemPrompt, userPrompt, nil
}

func (a *MemoryAgent) buildCrossTopicPrompts(node any) (string, string, error) {
	systemPrompt := prompts.CrossTopicStaticPrompt

	data := make(map[string]any)
	if m, ok := node.(map[string]any); ok {
		for k, v := range m {
			data[k] = v
		}
	}

	userPrompt, err := a.promptMgr.BuildPrompt("crosstopic", data)
	if err != nil {
		return "", "", err
	}

	return systemPrompt, userPrompt, nil
}

func (a *MemoryAgent) buildConflictPrompts(nodeA, nodeB any) (string, string, error) {
	systemPrompt := prompts.ConflictStaticPrompt

	decisionA := fmt.Sprintf("%v", nodeA)
	decisionB := fmt.Sprintf("%v", nodeB)

	userPrompt, err := a.promptMgr.BuildPrompt("conflict", map[string]any{
		"decisionA": decisionA,
		"decisionB": decisionB,
	})
	if err != nil {
		return "", "", err
	}

	return systemPrompt, userPrompt, nil
}
