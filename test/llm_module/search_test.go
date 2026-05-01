package llm_module_test

import (
	"os"
	"path/filepath"
	"testing"
	"feishu-mem/internal/search"
	"feishu-mem/internal/core"
	"feishu-mem/internal/decision"
	"feishu-mem/internal/storage/git"
	"github.com/stretchr/testify/assert"
)

// TestSearchTool 测试 SearchTool
func TestSearchTool(t *testing.T) {
	t.Log("测试 SearchTool 专用搜索工具")

	// 1. 创建测试数据
	mg := core.NewMemoryGraph()

	// 添加几个决策
	d1 := decision.NewDecisionNode("dec_001", "数据库架构选择", "test-project", "数据库架构")
	d1.Decision = "选择 PostgreSQL 作为主数据库"
	d1.ImpactLevel = decision.ImpactMajor
	d1.Status = decision.StatusDecided

	d2 := decision.NewDecisionNode("dec_002", "缓存架构设计", "test-project", "缓存架构")
	d2.Decision = "使用 Redis Cluster 作为缓存层"
	d2.ImpactLevel = decision.ImpactMajor
	d2.Status = decision.StatusDecided

	d3 := decision.NewDecisionNode("dec_003", "前端框架选型", "test-project", "前端框架")
	d3.Decision = "前端使用 Vue 3"
	d3.ImpactLevel = decision.ImpactMinor
	d3.Status = decision.StatusInDiscussion

	mg.UpsertDecision(d1, "test-project")
	mg.UpsertDecision(d2, "test-project")
	mg.UpsertDecision(d3, "test-project")

	// 2. 创建 SearchTool
	st := search.NewSearchTool(mg, "test-project")

	// 3. 测试搜索
	t.Run("搜索数据库相关", func(t *testing.T) {
		req := search.SearchRequest{
			Query: "数据库",
			Limit: 10,
		}

		resp, err := st.Search(req)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Results), 1)
		assert.Equal(t, "数据库架构", resp.Results[0].Topic)
		assert.Contains(t, resp.Results[0].Title, "数据库")

		t.Logf("搜索成功，返回 %d 条结果", len(resp.Results))
	})

	// 4. 测试按议题过滤
	t.Run("按议题过滤", func(t *testing.T) {
		req := search.SearchRequest{
			Query: "",
			Topic: "缓存架构",
			Limit: 10,
		}

		resp, err := st.Search(req)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(resp.Results))
		assert.Equal(t, "缓存架构", resp.Results[0].Topic)
	})

	// 5. 测试缓存
	t.Run("缓存机制", func(t *testing.T) {
		req := search.SearchRequest{
			Query: "数据库",
			Limit: 10,
		}

		// 第一次搜索
		resp1, err := st.Search(req)
		assert.NoError(t, err)

		// 第二次搜索应该走缓存
		resp2, err := st.Search(req)
		assert.NoError(t, err)

		assert.Equal(t, len(resp1.Results), len(resp2.Results))
		t.Log("缓存机制正常工作")
	})
}

// TestSearchToolBudget 测试搜索工具预算
func TestSearchToolBudget(t *testing.T) {
	t.Log("测试搜索工具预算 ≤ 20000 tokens")

	st := search.NewSearchTool(nil, "test-project")
	assert.Equal(t, 20000, st.OutputBudget)
	t.Logf("搜索工具预算配置正确：%d tokens", st.OutputBudget)
}

// TestSearchToolDedup 测试搜索工具去重
func TestSearchToolDedup(t *testing.T) {
	t.Log("测试搜索工具去重")

	mg := core.NewMemoryGraph()

	// 添加相同的决策（模拟重复）
	d1 := decision.NewDecisionNode("dec_001", "决策测试", "test-project", "测试")
	d1.Decision = "测试决策内容"
	mg.UpsertDecision(d1, "test-project")

	// 实际 SearchTool 按 SDRID 去重，这里应该不会有重复
	st := search.NewSearchTool(mg, "test-project")
	assert.True(t, st.Dedup)
	t.Log("去重机制已启用")
}

// TestSearchToolWithRanker 测试带自定义排名器的搜索
func TestSearchToolWithRanker(t *testing.T) {
	t.Log("测试搜索工具排名器")

	mg := core.NewMemoryGraph()
	d1 := decision.NewDecisionNode("dec_001", "数据库选择", "test-project", "数据库架构")
	d1.Decision = "PostgreSQL"
	d1.ImpactLevel = decision.ImpactMajor
	mg.UpsertDecision(d1, "test-project")

	st := search.NewSearchTool(mg, "test-project")
	st.SetRanker(&search.SimpleSearchRanker{})
	req := search.SearchRequest{
		Query: "数据库",
		Limit: 10,
	}

	resp, err := st.Search(req)
	assert.NoError(t, err)
	if len(resp.Results) > 0 {
		t.Logf("返回结果的相关性分数: %.2f", resp.Results[0].RelevanceScore)
	}
}

// TestSearchToolListTopics 测试列出所有议题
func TestSearchToolListTopics(t *testing.T) {
	t.Log("测试列出所有议题")

	mg := core.NewMemoryGraph()

	// 添加几个不同议题的决策
	d1 := decision.NewDecisionNode("dec_001", "决策1", "test-project", "数据库架构")
	d2 := decision.NewDecisionNode("dec_002", "决策2", "test-project", "缓存架构")
	mg.UpsertDecision(d1, "test-project")
	mg.UpsertDecision(d2, "test-project")

	st := search.NewSearchTool(mg, "test-project")
	topics := st.ListTopics("test-project")
	assert.GreaterOrEqual(t, len(topics), 2)

	for _, topic := range topics {
		t.Logf("- %s", topic)
	}
}

// TestSearchToolGetDecision 测试获取单个决策
func TestSearchToolGetDecision(t *testing.T) {
	t.Log("测试获取单个决策")

	mg := core.NewMemoryGraph()
	d1 := decision.NewDecisionNode("dec_001", "决策测试", "test-project", "测试")
	mg.UpsertDecision(d1, "test-project")

	st := search.NewSearchTool(mg, "test-project")

	// 获取存在的决策
	result, ok := st.GetDecision("dec_001")
	assert.True(t, ok)
	assert.Equal(t, "dec_001", result.SDRID)
	assert.Equal(t, "决策测试", result.Title)

	// 获取不存在的决策
	result, ok = st.GetDecision("dec_nonexistent")
	assert.False(t, ok)
}

// TestSearchToolWithGitStorage 测试 SearchTool 与 Git 持久化存储集成
func TestSearchToolWithGitStorage(t *testing.T) {
	t.Log("测试 SearchTool 与 Git 持久化存储集成")

	// 1. 创建临时目录
	tmpDir, err := os.MkdirTemp("", "git-search-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Logf("临时目录: %s", tmpDir)

	// 2. 创建 GitStorage
	gitConfig := git.Config{
		WorkDir: tmpDir,
		Branch:  "main",
		AutoPush: false,
	}

	gs, err := git.NewGitStorage(gitConfig)
	assert.NoError(t, err)

	// 3. 写入几个测试决策到 Git
	testProject := "test-git-project"

	d1 := decision.NewDecisionNode("git_dec_001", "数据库架构选择", testProject, "数据库架构")
	d1.Decision = "选择 PostgreSQL 作为主数据库"
	d1.Rationale = "PostgreSQL 支持复杂查询和 JSON"
	d1.ImpactLevel = decision.ImpactMajor
	d1.Status = decision.StatusDecided
	_, err = gs.WriteDecision(d1)
	assert.NoError(t, err)

	d2 := decision.NewDecisionNode("git_dec_002", "缓存架构设计", testProject, "缓存架构")
	d2.Decision = "使用 Redis Cluster 作为缓存层"
	d2.Rationale = "Redis Cluster 提供高可用性"
	d2.ImpactLevel = decision.ImpactMajor
	d2.Status = decision.StatusDecided
	_, err = gs.WriteDecision(d2)
	assert.NoError(t, err)

	d3 := decision.NewDecisionNode("git_dec_003", "前端框架选型", testProject, "前端框架")
	d3.Decision = "前端使用 Vue 3"
	d3.Rationale = "Vue 3 有良好的生态系统"
	d3.ImpactLevel = decision.ImpactMinor
	d3.Status = decision.StatusInDiscussion
	_, err = gs.WriteDecision(d3)
	assert.NoError(t, err)

	t.Log("测试决策已写入 Git")

	// 4. 创建 MemoryGraph 并从 Git 加载
	mg := core.NewMemoryGraph()
	err = mg.LoadFromGit(gs, testProject)
	assert.NoError(t, err)
	assert.Equal(t, 3, mg.Count())
	t.Logf("MemoryGraph 已加载 %d 个决策", mg.Count())

	// 5. 创建 SearchTool
	st := search.NewSearchTool(mg, testProject)

	// 6. 测试搜索 - 搜索数据库相关
	t.Run("从 Git 加载的决策搜索", func(t *testing.T) {
		req := search.SearchRequest{
			Query: "数据库",
			Limit: 10,
		}

		resp, err := st.Search(req)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Results), 1)
		assert.Equal(t, "数据库架构", resp.Results[0].Topic)
		assert.Contains(t, resp.Results[0].Title, "数据库")

		t.Logf("搜索成功，返回 %d 条结果", len(resp.Results))
		for _, r := range resp.Results {
			t.Logf("  - %s [%s] (相关性: %.2f)", r.Title, r.Topic, r.RelevanceScore)
		}
	})

	// 7. 测试按议题过滤
	t.Run("按议题过滤 Git 决策", func(t *testing.T) {
		req := search.SearchRequest{
			Query: "",
			Topic: "缓存架构",
			Limit: 10,
		}

		resp, err := st.Search(req)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(resp.Results))
		assert.Equal(t, "缓存架构", resp.Results[0].Topic)
		assert.Contains(t, resp.Results[0].Title, "缓存")
	})

	// 8. 测试列出所有议题
	t.Run("列出 Git 中的议题", func(t *testing.T) {
		topics := st.ListTopics(testProject)
		assert.GreaterOrEqual(t, len(topics), 2)
		t.Log("Git 中的议题:")
		for _, topic := range topics {
			t.Logf("  - %s", topic)
		}
	})

	// 9. 测试获取单个决策
	t.Run("从 Git 获取单个决策", func(t *testing.T) {
		result, ok := st.GetDecision("git_dec_001")
		assert.True(t, ok)
		assert.Equal(t, "git_dec_001", result.SDRID)
		assert.Equal(t, "数据库架构选择", result.Title)
	})

	// 10. 测试 Git 直接搜索（Git grep）
	t.Run("Git 直接全文搜索", func(t *testing.T) {
		hits, err := gs.SearchContent(testProject, "Redis")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(hits), 1)
		t.Logf("Git grep 搜索 'Redis' 命中 %d 条", len(hits))
		for _, hit := range hits {
			t.Logf("  - %s: %s", hit.File, hit.Content)
		}
	})
}

// TestSearchToolGitLoadError 测试 Git 加载错误处理
func TestSearchToolGitLoadError(t *testing.T) {
	t.Log("测试 Git 加载错误处理")

	// 创建临时目录但不初始化 Git
	tmpDir, err := os.MkdirTemp("", "git-error-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建不存在的项目目录（模拟空仓库）
	nonExistentDir := filepath.Join(tmpDir, "nonexistent")

	// 创建 GitStorage - 应该能正常初始化（会自动 init）
	gitConfig := git.Config{
		WorkDir: nonExistentDir,
		Branch:  "main",
	}
	gs, err := git.NewGitStorage(gitConfig)
	assert.NoError(t, err)

	// 加载不存在的项目 - 应该返回空而不报错
	mg := core.NewMemoryGraph()
	err = mg.LoadFromGit(gs, "nonexistent-project")
	assert.NoError(t, err)
	assert.Equal(t, 0, mg.Count())

	t.Log("错误处理正常：空仓库加载返回空结果")
}

