package git_test

import (
	"testing"

	"feishu-mem/internal/decision"
	"feishu-mem/internal/storage/git"
)

func TestRenderDecisionFile(t *testing.T) {
	tests := []struct {
		name string
		node *decision.DecisionNode
		wantContains []string
	}{
		{
			name: "完整决策节点渲染",
			node: func() *decision.DecisionNode {
				d := decision.NewDecisionNode("DEC-20260501-001", "数据库架构选择", "feishu-mem", "database")
				d.Decision = "选择 PostgreSQL 作为主数据库"
				d.Rationale = "PostgreSQL 支持复杂查询和 JSON 数据类型，适合我们的业务场景"
				d.ImpactLevel = decision.ImpactMajor
				d.Status = decision.StatusDecided
				d.Proposer = "张三"
				d.Executor = "李四"
				return d
			}(),
			wantContains: []string{
				"DEC-20260501-001",
				"数据库架构选择",
				"选择 PostgreSQL 作为主数据库",
				"PostgreSQL 支持复杂查询",
				"major",
				"decided",
				"张三",
				"李四",
			},
		},
		{
			name: "最小化决策节点",
			node: func() *decision.DecisionNode {
				d := decision.NewDecisionNode("DEC-001", "简单决策", "test", "topic")
				d.Decision = "简单内容"
				return d
			}(),
			wantContains: []string{
				"DEC-001",
				"简单决策",
				"简单内容",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := git.RenderDecisionFile(tt.node)

			if content == "" {
				t.Error("期望非空内容")
			}

			// 检查包含 YAML frontmatter
			if len(content) < 8 || content[:4] != "---\n" {
				t.Error("期望以 YAML frontmatter 开始")
			}

			// 检查包含 Markdown 正文
			for _, substr := range tt.wantContains {
				if !contains(content, substr) {
					t.Errorf("期望包含 %q", substr)
				}
			}

			t.Logf("渲染内容长度: %d 字符", len(content))
		})
	}
}

func TestRenderAndParseRoundTrip(t *testing.T) {
	d := decision.NewDecisionNode("DEC-001", "往返测试", "test", "test-topic")
	d.Decision = "这是决策内容"
	d.Rationale = "这是决策依据"
	d.ImpactLevel = decision.ImpactCritical
	d.Status = decision.StatusDecided
	d.Proposer = "测试用户"
	d.CrossTopicRefs = []string{"topic1", "topic2"}

	// 渲染
	content := git.RenderDecisionFile(d)

	// 解析
	parsed, err := git.ParseDecisionFile([]byte(content))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 验证字段
	if parsed.SDRID != d.SDRID {
		t.Errorf("SDRID 不匹配: 期望 %q, 得到 %q", d.SDRID, parsed.SDRID)
	}
	if parsed.Title != d.Title {
		t.Errorf("Title 不匹配: 期望 %q, 得到 %q", d.Title, parsed.Title)
	}
	if parsed.Decision != d.Decision {
		t.Errorf("Decision 不匹配: 期望 %q, 得到 %q", d.Decision, parsed.Decision)
	}
	if parsed.Rationale != d.Rationale {
		t.Errorf("Rationale 不匹配: 期望 %q, 得到 %q", d.Rationale, parsed.Rationale)
	}
	if parsed.ImpactLevel != d.ImpactLevel {
		t.Errorf("ImpactLevel 不匹配: 期望 %q, 得到 %q", d.ImpactLevel, parsed.ImpactLevel)
	}
	if parsed.Status != d.Status {
		t.Errorf("Status 不匹配: 期望 %q, 得到 %q", d.Status, parsed.Status)
	}
	if parsed.Proposer != d.Proposer {
		t.Errorf("Proposer 不匹配: 期望 %q, 得到 %q", d.Proposer, parsed.Proposer)
	}

	t.Log("往返测试通过")
}

func TestFormatDecisionNode(t *testing.T) {
	d := decision.NewDecisionNode("DEC-001", "格式化测试", "test", "topic")
	d.Status = decision.StatusDecided
	d.ImpactLevel = decision.ImpactMajor
	d.GitCommitHash = "abcdef1234567890abcdef1234567890"

	formatted := git.FormatDecisionNode(d)

	if !contains(formatted, "DEC-001") {
		t.Error("格式化结果应包含 SDRID")
	}
	if !contains(formatted, "格式化测试") {
		t.Error("格式化结果应包含标题")
	}
	if !contains(formatted, "abcdef12") {
		t.Error("格式化结果应包含短 commit hash")
	}

	t.Logf("格式化结果:\n%s", formatted)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
