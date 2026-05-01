package llm_module_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromptTemplateCompleteness(t *testing.T) {
	t.Log("测试提示词模板的完整性")

	testCases := []struct {
		name      string
		template  string
		checkKeys []string
	}{
		{
			name:     "extraction_prompt",
			template: extractionStaticPrompt,
			checkKeys: []string{
				"决策提取器", "决定", "确认", "has_decision", "confidence", "impact_level",
			},
		},
		{
			name:     "classification_prompt",
			template: classificationStaticPrompt,
			checkKeys: []string{
				"议题分类器", "topic", "confidence", "reasoning", "alternative_topics",
			},
		},
		{
			name:     "crosstopic_prompt",
			template: crosstopicStaticPrompt,
			checkKeys: []string{
				"跨议题影响检测器", "is_cross_topic", "cross_topic_refs", "reasons",
			},
		},
		{
			name:     "conflict_prompt",
			template: conflictStaticPrompt,
			checkKeys: []string{
				"决策冲突评估器", "contradiction_score", "contradiction_type", "直接", "参数", "时序",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, len(tc.template) > 0, "template should not be empty")
			for _, key := range tc.checkKeys {
				assert.Contains(t, tc.template, key, "template should contain key: "+key)
			}
			t.Logf("✓ %s template is complete, length: %d chars", tc.name, len(tc.template))
		})
	}
}

func TestExtractionPromptConstruction(t *testing.T) {
	t.Log("测试决策提取提示词的动态构建")

	pm := NewPromptManager()

	testContext := map[string]any{
		"content": "我们决定使用 PostgreSQL，执行人是张三",
		"topics":  []string{"数据库架构", "缓存架构", "API网关"},
	}

	prompt, err := pm.BuildPrompt("extraction", testContext)
	assert.NoError(t, err)
	assert.Contains(t, prompt, "PostgreSQL")
	assert.Contains(t, prompt, "数据库架构")
	assert.Contains(t, prompt, "待分析内容")
	assert.Contains(t, prompt, "候选议题")

	t.Logf("✓ Extraction prompt constructed, length: %d", len(prompt))
}

func TestClassificationPromptConstruction(t *testing.T) {
	t.Log("测试议题分类提示词的动态构建")

	pm := NewPromptManager()

	testContext := map[string]any{
		"decision": "决定使用 Redis 作为缓存",
		"topics": []string{
			"数据库架构", "缓存架构", "用户服务", "前端框架", "日志监控",
		},
	}

	prompt, err := pm.BuildPrompt("classification", testContext)
	assert.NoError(t, err)
	assert.Contains(t, prompt, "Redis")
	assert.Contains(t, prompt, "缓存架构")
	assert.Contains(t, prompt, "决策内容")
	assert.Contains(t, prompt, "候选议题列表")

	t.Logf("✓ Classification prompt constructed, length: %d", len(prompt))
}

func TestCrossTopicPromptConstruction(t *testing.T) {
	t.Log("测试跨议题检测提示词的动态构建")

	pm := NewPromptManager()

	testContext := map[string]any{
		"topic":          "数据库架构",
		"title":          "用户表字段变更",
		"decision":       "将 user_profile 表的 session_token 字段从 256 字节扩展到 512 字节",
		"rationale":      "新的 JWT 格式需要更长的 token 字段",
		"impact_level":   "major",
		"candidate_topics": []string{"用户认证", "前端框架", "日志监控", "API网关"},
	}

	prompt, err := pm.BuildPrompt("crosstopic", testContext)
	assert.NoError(t, err)
	assert.Contains(t, prompt, "user_profile 表")
	assert.Contains(t, prompt, "session_token")
	assert.Contains(t, prompt, "所属议题")
	assert.Contains(t, prompt, "决策结论")
	assert.Contains(t, prompt, "判定维度")

	t.Logf("✓ CrossTopic prompt constructed, length: %d", len(prompt))
}

func TestConflictPromptConstruction(t *testing.T) {
	t.Log("测试冲突评估提示词的动态构建")

	pm := NewPromptManager()

	testContext := map[string]any{
		"decisionA": "决定使用 PostgreSQL 作为主数据库，主要因为它支持复杂查询和 JSON 类型",
		"decisionB": "之前决定使用 MySQL 作为主数据库，主要因为团队更熟悉 MySQL",
	}

	prompt, err := pm.BuildPrompt("conflict", testContext)
	assert.NoError(t, err)
	assert.Contains(t, prompt, "PostgreSQL")
	assert.Contains(t, prompt, "MySQL")
	assert.Contains(t, prompt, "决策 A")
	assert.Contains(t, prompt, "决策 B")
	assert.Contains(t, prompt, "直接矛盾")
	assert.Contains(t, prompt, "contradiction_score")

	t.Logf("✓ Conflict prompt constructed, length: %d", len(prompt))
}

func TestPromptTokenEstimates(t *testing.T) {
	t.Log("测试提示词的 Token 消耗估算（简单按字符估算）")

	testCases := []struct {
		name     string
		template string
		maxToken int
	}{
		{"extraction", extractionStaticPrompt, 4000},
		{"classification", classificationStaticPrompt, 2000},
		{"crosstopic", crosstopicStaticPrompt, 3000},
		{"conflict", conflictStaticPrompt, 3000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			estimatedTokens := len(tc.template) / 4
			t.Logf("  %s: estimated %d tokens (max: %d)", tc.name, estimatedTokens, tc.maxToken)
			assert.True(t, len(tc.template) > 100, "template should be long enough")
		})
	}
}

func TestPromptMarkdownStructure(t *testing.T) {
	t.Log("测试提示词的 Markdown 结构完整性")

	testCases := map[string]string{
		"extraction":     extractionStaticPrompt,
		"classification": classificationStaticPrompt,
		"crosstopic":     crosstopicStaticPrompt,
		"conflict":       conflictStaticPrompt,
	}

	for name, template := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.True(t, strings.Contains(template, "# ") || strings.Contains(template, "## "), "should have markdown headers")
			assert.True(t, strings.Contains(template, "## "), "should have section structure")
			assert.Contains(t, template, "{", "should contain JSON example")
			assert.Contains(t, template, "}", "should contain JSON example")
		})
	}
}

func TestPromptChineseContent(t *testing.T) {
	t.Log("测试提示词中的中文内容质量")

	testCases := []struct {
		name      string
		template  string
		keyWords []string
	}{
		{
			"extraction", extractionStaticPrompt,
			[]string{"决定", "确认", "结论", "通过", "定下来"},
		},
		{
			"classification", classificationStaticPrompt,
			[]string{"议题", "分类", "匹配", "备选"},
		},
		{
			"crosstopic", crosstopicStaticPrompt,
			[]string{"跨议题", "影响", "依赖", "模块"},
		},
		{
			"conflict", conflictStaticPrompt,
			[]string{"矛盾", "冲突", "直接", "参数", "时序"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			foundCount := 0
			for _, kw := range tc.keyWords {
				if strings.Contains(tc.template, kw) {
					foundCount++
				}
			}
			assert.True(t, foundCount >= len(tc.keyWords)/2, fmt.Sprintf("should contain at least half of keywords, found %d/%d", foundCount, len(tc.keyWords)))
		})
	}
}
