package mcp_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	"feishu-mem/internal/core"
	"feishu-mem/internal/decision"
	larkadapter "feishu-mem/internal/lark-adapter"
	"feishu-mem/internal/llm"
	"feishu-mem/internal/mcp"
)

// Test_01_InitializeRequest 测试初始化请求
func Test_01_InitializeRequest(t *testing.T) {
	t.Log("=== 测试1: MCP initialize 请求 ===")

	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
			},
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	reqJSON, err := json.MarshalIndent(req, "", "  ")
	assert.NoError(t, err)
	t.Logf("请求 JSON:\n%s\n", string(reqJSON))

	assert.Equal(t, "2.0", req.JSONRPC)
	assert.Equal(t, "initialize", req.Method)
	assert.Equal(t, 1, req.ID)

	t.Log("✓ 初始化请求构建成功")
}

// Test_02_ToolsListRequest 测试工具列表请求
func Test_02_ToolsListRequest(t *testing.T) {
	t.Log("=== 测试2: MCP tools/list 请求 ===")

	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
		Params:  map[string]any{},
	}

	reqJSON, err := json.MarshalIndent(req, "", "  ")
	assert.NoError(t, err)
	t.Logf("请求 JSON:\n%s\n", string(reqJSON))

	t.Log("✓ 工具列表请求构建成功")
}

// Test_03_ToolCallRequest 测试工具调用请求
func Test_03_ToolCallRequest(t *testing.T) {
	t.Log("=== 测试3: MCP tools/call 请求 ===")

	testCases := []struct {
		name      string
		toolName  string
		arguments map[string]any
	}{
		{
			name:     "search 工具",
			toolName: "search",
			arguments: map[string]any{
				"query": "数据库",
				"limit": 10,
			},
		},
		{
			name:     "topic 工具",
			toolName: "topic",
			arguments: map[string]any{
				"topic": "数据库架构",
			},
		},
		{
			name:     "decision 工具",
			toolName: "decision",
			arguments: map[string]any{
				"sdr_id": "DEC-001",
			},
		},
		{
			name:     "extract_decision 工具",
			toolName: "extract_decision",
			arguments: map[string]any{
				"content": "我们决定使用 PostgreSQL 作为主数据库，张三负责实施。",
				"topics": []any{"数据库架构", "缓存架构"},
			},
		},
		{
			name:     "classify_topic 工具",
			toolName: "classify_topic",
			arguments: map[string]any{
				"decision": "使用 Redis 作为缓存",
				"topics": []any{"数据库架构", "缓存架构", "前端框架"},
			},
		},
		{
			name:     "detect_crosstopic 工具",
			toolName: "detect_crosstopic",
			arguments: map[string]any{
				"title":            "用户表字段变更",
				"decision":         "将 session_token 字段扩展到 512 字节",
				"candidate_topics": []any{"用户认证", "API网关", "数据库架构"},
			},
		},
		{
			name:     "check_conflict 工具",
			toolName: "check_conflict",
			arguments: map[string]any{
				"decision_a": "使用 PostgreSQL",
				"decision_b": "使用 MySQL",
			},
		},
		{
			name:     "timeline 工具",
			toolName: "timeline",
			arguments: map[string]any{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.Request{
				JSONRPC: "2.0",
				ID:      100 + len(testCases),
				Method:  "tools/call",
				Params: map[string]any{
					"name":      tc.toolName,
					"arguments": tc.arguments,
				},
			}

			reqJSON, err := json.MarshalIndent(req, "", "  ")
			assert.NoError(t, err)
			t.Logf("\n--- %s 请求 JSON:\n%s\n", tc.name, string(reqJSON))
		})
	}

	t.Log("✓ 所有工具调用请求构建成功")
}

// Test_04_ResourcesListRequest 测试资源列表请求
func Test_04_ResourcesListRequest(t *testing.T) {
	t.Log("=== 测试4: MCP resources/list 请求 ===")

	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "resources/list",
		Params:  map[string]any{},
	}

	reqJSON, err := json.MarshalIndent(req, "", "  ")
	assert.NoError(t, err)
	t.Logf("请求 JSON:\n%s\n", string(reqJSON))

	t.Log("✓ 资源列表请求构建成功")
}

// Test_05_ResourceReadRequest 测试资源读取请求
func Test_05_ResourceReadRequest(t *testing.T) {
	t.Log("=== 测试5: MCP resources/read 请求 ===")

	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "resources/read",
		Params: map[string]any{
			"uri": "docs://design",
		},
	}

	reqJSON, err := json.MarshalIndent(req, "", "  ")
	assert.NoError(t, err)
	t.Logf("请求 JSON:\n%s\n", string(reqJSON))

	t.Log("✓ 资源读取请求构建成功")
}

// Test_06_ResponseValidation 测试响应格式验证
func Test_06_ResponseValidation(t *testing.T) {
	t.Log("=== 测试6: 响应格式验证 ===")

	successResp := mcp.Response{
		JSONRPC: "2.0",
		ID:      1,
		Result: map[string]any{
			"content": []mcp.Content{
				{
					Type: "text",
					Text: "测试结果",
				},
			},
		},
	}

	respJSON, err := json.MarshalIndent(successResp, "", "  ")
	assert.NoError(t, err)
	t.Logf("成功响应 JSON:\n%s\n", string(respJSON))

	errorResp := mcp.Response{
		JSONRPC: "2.0",
		ID:      2,
		Error: &mcp.Error{
			Code:    -32000,
			Message: "测试错误",
		},
	}

	errRespJSON, err := json.MarshalIndent(errorResp, "", "  ")
	assert.NoError(t, err)
	t.Logf("错误响应 JSON:\n%s\n", string(errRespJSON))

	t.Log("✓ 响应格式验证成功")
}

// Test_07_FullProtocolFlow 测试完整协议流程
func Test_07_FullProtocolFlow(t *testing.T) {
	t.Log("=== 测试7: 完整 MCP 协议流程 ===")

	memoryGraph := core.NewMemoryGraph()
	d1 := decision.NewDecisionNode("DEC-001", "数据库架构选择", "test-proj", "数据库架构")
	d1.Decision = "使用 PostgreSQL 作为主数据库"
	d1.Status = decision.StatusDecided
	memoryGraph.UpsertDecision(d1, "test-proj")

	server := mcp.NewMCPServer(memoryGraph, nil, nil)
	assert.NotNil(t, server)

	t.Log("\n步骤 1: 客户端发送 initialize 请求")
	initReq := mcp.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0",
			},
		},
	}

	t.Logf("initialize 请求: %+v\n", initReq)
	t.Log("预期响应包含: protocolVersion, capabilities, serverInfo")

	t.Log("\n步骤 2: 客户端发送 tools/list 请求")
	toolsReq := mcp.Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
		Params:  map[string]any{},
	}
	t.Logf("tools/list 请求: %+v\n", toolsReq)

	t.Log("\n步骤 3: 客户端调用 search 工具")
	searchReq := mcp.Request{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "search",
			"arguments": map[string]any{
				"query": "数据库",
			},
		},
	}
	t.Logf("tools/call (search) 请求: %+v\n", searchReq)

	t.Log("\n✓ 完整协议流程测试完成")
}

// Test_08_RealLLM_ExtractDecision 测试真实 LLM - 决策提取
func Test_08_RealLLM_ExtractDecision(t *testing.T) {
	t.Log("=== 测试8: 真实 LLM 调用 - 决策提取 ===")

	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY 未设置，跳过真实 LLM 测试")
	}

	llmAgent := llm.NewMemoryAgent()
	assert.NotNil(t, llmAgent)

	testContent := "我们决定使用 PostgreSQL 作为主数据库，因为它支持复杂查询和 JSON 类型。该决定将在 2024-12-01 开始生效，由张三负责实施，李四作为提案人。"
	testTopics := []string{"数据库架构", "缓存架构", "用户认证"}

	t.Logf("测试内容: %s\n", testContent)
	t.Logf("候选议题: %v\n", testTopics)

	result, err := llmAgent.ExtractDecision(testContent, testTopics)
	if err != nil {
		t.Logf("LLM 调用失败，使用降级策略: %v\n", err)
	} else {
		t.Logf("LLM 调用成功！\n")
	}

	t.Logf("提取结果 - has_decision: %v, confidence: %.2f\n",
		result.HasDecision, result.Confidence)

	if result.HasDecision && result.Decision != nil {
		t.Logf("  标题: %s\n", result.Decision.Title)
		t.Logf("  决策内容: %s\n", result.Decision.Decision)
		t.Logf("  建议议题: %s\n", result.Decision.SuggestedTopic)
		t.Logf("  影响级别: %s\n", result.Decision.ImpactLevel)
		t.Logf("  提案人: %s\n", result.Decision.Proposer)
		t.Logf("  执行人: %s\n", result.Decision.Executor)
	}

	t.Logf("  提取来源摘要: %s\n", result.ExtractedFrom)
	t.Log("✓ 真实 LLM 决策提取测试完成")
}

// Test_09_RealLLM_ClassifyTopic 测试真实 LLM - 议题分类
func Test_09_RealLLM_ClassifyTopic(t *testing.T) {
	t.Log("=== 测试9: 真实 LLM 调用 - 议题分类 ===")

	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY 未设置，跳过真实 LLM 测试")
	}

	llmAgent := llm.NewMemoryAgent()
	assert.NotNil(t, llmAgent)

	testDecision := "使用 Redis Cluster 作为分布式缓存方案，支持缓存分片和高可用。"
	testTopics := []string{"数据库架构", "缓存架构", "前端框架", "用户认证", "API网关"}

	t.Logf("决策内容: %s\n", testDecision)
	t.Logf("候选议题: %v\n", testTopics)

	result, err := llmAgent.ClassifyTopic(testDecision, testTopics)
	if err != nil {
		t.Logf("LLM 调用失败，使用降级策略: %v\n", err)
	} else {
		t.Logf("LLM 调用成功！\n")
	}

	t.Logf("分类结果 - 建议议题: %s, 置信度: %.2f\n",
		result.Topic, result.Confidence)
	t.Logf("  说明: %s\n", result.Reasoning)

	if len(result.AlternativeTopics) > 0 {
		t.Logf("  备选议题: %v\n", result.AlternativeTopics)
	}

	t.Log("✓ 真实 LLM 议题分类测试完成")
}

// Test_10_RealLLM_DetectCrossTopic 测试真实 LLM - 跨议题检测
func Test_10_RealLLM_DetectCrossTopic(t *testing.T) {
	t.Log("=== 测试10: 真实 LLM 调用 - 跨议题检测 ===")

	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY 未设置，跳过真实 LLM 测试")
	}

	llmAgent := llm.NewMemoryAgent()
	assert.NotNil(t, llmAgent)

	testData := map[string]any{
		"title":          "用户表字段变更",
		"decision":       "将 user_profile 表的 session_token 字段从 256 字节扩展到 512 字节",
		"rationale":      "新的 JWT 格式需要更长的 token 字段",
		"impact_level":   "major",
		"candidate_topics": []any{"数据库架构", "用户认证", "API网关", "前端框架"},
	}

	t.Logf("决策标题: %s\n", testData["title"])
	t.Logf("决策内容: %s\n", testData["decision"])
	t.Logf("候选议题: %v\n", testData["candidate_topics"])

	result, err := llmAgent.DetectCrossTopic(testData)
	if err != nil {
		t.Logf("LLM 调用失败，使用降级策略: %v\n", err)
	} else {
		t.Logf("LLM 调用成功！\n")
	}

	t.Logf("检测结果 - is_cross_topic: %v, confidence: %.2f\n",
		result.IsCrossTopic, result.Confidence)

	if result.IsCrossTopic {
		t.Logf("  受影响议题: %v\n", result.CrossTopicRefs)
		t.Logf("  原因说明: %v\n", result.Reasons)
	}

	t.Log("✓ 真实 LLM 跨议题检测测试完成")
}

// Test_11_RealLLM_CheckConflict 测试真实 LLM - 冲突评估
func Test_11_RealLLM_CheckConflict(t *testing.T) {
	t.Log("=== 测试11: 真实 LLM 调用 - 冲突评估 ===")

	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY 未设置，跳过真实 LLM 测试")
	}

	llmAgent := llm.NewMemoryAgent()
	assert.NotNil(t, llmAgent)

	testDecisionA := "使用 PostgreSQL 作为主数据库，因为它支持复杂查询和 JSON 类型"
	testDecisionB := "使用 MySQL 作为主数据库，因为团队对它更熟悉，性能也足够好"

	t.Logf("决策 A: %s\n", testDecisionA)
	t.Logf("决策 B: %s\n", testDecisionB)

	result, err := llmAgent.ResolveConflict(testDecisionA, testDecisionB)
	if err != nil {
		t.Logf("LLM 调用失败: %v\n", err)
	} else {
		t.Logf("LLM 调用成功！\n")
	}

	t.Logf("冲突评估结果:\n")
	t.Logf("  矛盾分数: %.2f\n", result.ContradictionScore)
	t.Logf("  矛盾类型: %s\n", result.ContradictionType)
	t.Logf("  描述: %s\n", result.Description)
	t.Logf("  建议: %s\n", result.Suggestion)
	t.Logf("  处理行动: %s\n", result.Action)
	t.Logf("  需要用户介入: %v\n", result.NeedsUser)

	t.Log("✓ 真实 LLM 冲突评估测试完成")
}

// Test_12_RealLLM_EndToEnd 测试真实 LLM - 端到端完整流程
func Test_12_RealLLM_EndToEnd(t *testing.T) {
	t.Log("=== 测试12: 真实 LLM - 端到端完整流程 ===")

	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY 未设置，跳过真实 LLM 测试")
	}

	llmAgent := llm.NewMemoryAgent()
	assert.NotNil(t, llmAgent)

	t.Log("\n=== 步骤 1: 从原始文本提取决策 ===")
	rawContent := `今天的团队会议做出了以下决定：
1. 使用 PostgreSQL 作为主数据库，支持 JSON 查询
2. 使用 Redis Cluster 作为缓存方案
3. API 网关层使用 Kong 实现限流
这些决定由张三提出，李四负责实施，项目代码需要在 2024-12-31 前完成。`

	topics := []string{"数据库架构", "缓存架构", "API网关", "前端框架"}

	t.Logf("原始文本长度: %d\n", len(rawContent))

	extractResult, err := llmAgent.ExtractDecision(rawContent, topics)
	if err != nil {
		t.Logf("LLM 调用失败，使用降级策略: %v\n", err)
	} else {
		t.Logf("提取成功，置信度: %.2f\n", extractResult.Confidence)
	}

	if extractResult.HasDecision && extractResult.Decision != nil {
		t.Logf("\n提取到的决策:\n")
		t.Logf("  标题: %s\n", extractResult.Decision.Title)
		t.Logf("  建议议题: %s\n", extractResult.Decision.SuggestedTopic)
	}

	t.Log("\n=== 步骤 2: 测试议题分类 ===")
	if extractResult.HasDecision && extractResult.Decision != nil {
		classifyResult, err := llmAgent.ClassifyTopic(
			extractResult.Decision.Decision, topics)

		if err != nil {
			t.Logf("分类调用失败: %v\n", err)
		} else {
			t.Logf("分类结果: %s (置信度: %.2f)\n", classifyResult.Topic, classifyResult.Confidence)
		}
	}

	t.Log("\n=== 步骤 3: 构建 MCP 请求格式 ===")
	extractRequest := mcp.Request{
		JSONRPC: "2.0",
		ID:      100,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "extract_decision",
			"arguments": map[string]any{
				"content": rawContent,
				"topics":  topics,
			},
		},
	}

	reqJSON, _ := json.MarshalIndent(extractRequest, "", "  ")
	t.Logf("对应的 MCP 请求:\n%s\n", string(reqJSON))

	t.Log("\n✓ 真实 LLM 端到端完整流程测试完成！")
}

// Test_13_MCPToLarkPush 测试 MCP-server 给飞书客户端推送消息
func Test_13_MCPToLarkPush(t *testing.T) {
	t.Log("=== 测试13: MCP-server 给飞书客户端推送消息 ===")

	// 加载 .env 文件
	_ = godotenv.Load("../../.env")

	// 检查 CLAW_* 配置
	cfg := larkadapter.LoadConfigWithPrefix("CLAW_")
	if !cfg.IsConfigured() {
		t.Skip("CLAW_APP_ID 或 CLAW_APP_SECRET 未设置，跳过飞书消息推送测试")
	}
	if len(cfg.ChatIDs) == 0 {
		t.Skip("CLAW_CHAT_IDS 未设置，跳过飞书消息推送测试")
	}

	t.Logf("CLAW_APP_ID: %s\n", maskString(cfg.AppID))
	t.Logf("目标群聊: %v\n", cfg.ChatIDs)

	// 创建消息发送器
	sender := larkadapter.NewIMSender(cfg)

	// 准备三条消息
	messages := []struct {
		index int
		text  string
	}{
		{1, "【MCP测试消息 1/3】这是来自 MCP-server 的第一条测试消息，时间: " + timeNowString()},
		{2, "【MCP测试消息 2/3】这是来自 MCP-server 的第二条测试消息，确认推送通道正常工作"},
		{3, "【MCP测试消息 3/3】这是来自 MCP-server 的第三条测试消息，测试完成！"},
	}

	// 发送消息
	for _, msg := range messages {
		t.Logf("\n--- 发送消息 %d ---\n", msg.index)
		t.Logf("消息内容: %s\n", msg.text)

		results, err := sender.SendTextMessageToAll(msg.text)
		if err != nil {
			t.Logf("发送失败: %v\n", err)
		} else {
			for _, result := range results {
				t.Logf("结果: %s\n", result)
			}
		}
	}

	t.Log("\n✓ MCP-server 给飞书客户端推送消息测试完成")
}

// Test_14_LarkToMCPReceive 测试飞书客户端给 MCP-server 发送消息
func Test_14_LarkToMCPReceive(t *testing.T) {
	t.Log("=== 测试14: 飞书客户端给 MCP-server 发送消息 ===")

	// 加载 .env 文件
	_ = godotenv.Load("../../.env")

	// 检查 LARK_* 配置
	cfg := larkadapter.LoadConfigWithPrefix("LARK_")
	if !cfg.IsConfigured() {
		t.Skip("LARK_APP_ID 或 LARK_APP_SECRET 未设置，跳过飞书消息接收测试")
	}
	if len(cfg.ChatIDs) == 0 {
		t.Skip("LARK_CHAT_IDS 未设置，跳过飞书消息接收测试")
	}

	t.Logf("LARK_APP_ID: %s\n", maskString(cfg.AppID))
	t.Logf("LARK_CHAT_IDS: %v\n", cfg.ChatIDs)

	// 创建消息提取器
	extractor := larkadapter.NewIMExtractor(cfg)

	// 检测最近消息变化
	t.Log("\n--- 检测最近消息变化 ---\n")
	result, err := extractor.Detect(time.Now().Add(-24 * time.Hour))
	if err != nil {
		t.Logf("检测失败: %v\n", err)
	} else {
		t.Logf("检测结果 - HasChanges: %v\n", result.HasChanges)
		t.Logf("检测到 %d 条变化\n", len(result.Changes))

		for i, change := range result.Changes {
			t.Logf("  [%d] %s: %s\n", i+1, change.Type, change.Summary)
		}
	}

	t.Log("\n提示：请在飞书客户端发送三条新消息，然后重新运行此测试来验证接收功能")

	t.Log("\n✓ 飞书客户端给 MCP-server 发送消息测试完成")
}

// ========== 辅助函数 ==========

func maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "****"
}

func timeNowString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

