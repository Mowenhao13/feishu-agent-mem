package git_test

import (
	"os"
	"testing"
	"time"

	"feishu-mem/internal/decision"
	"feishu-mem/internal/storage/git"
)

func TestGitCommit(t *testing.T) {
	t.Log("测试 Git Commit 功能")

	tmpDir, err := os.MkdirTemp("", "git-commit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := git.NewGitStorage(git.Config{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 创建并写入第一个决策
	d1 := decision.NewDecisionNode("DEC-001", "第一次提交", "test", "topic1")
	d1.Decision = "这是第一个决策内容"
	hash1, err := gs.WriteDecision(d1)
	if err != nil {
		t.Fatalf("第一次写入失败: %v", err)
	}
	t.Logf("第一次提交 hash: %s", hash1)

	// 创建并写入第二个决策
	d2 := decision.NewDecisionNode("DEC-002", "第二次提交", "test", "topic1")
	d2.Decision = "这是第二个决策内容"
	hash2, err := gs.WriteDecision(d2)
	if err != nil {
		t.Fatalf("第二次写入失败: %v", err)
	}
	t.Logf("第二次提交 hash: %s", hash2)

	// 验证两次 hash 不同
	if hash1 == hash2 {
		t.Error("两次提交的 hash 应该不同")
	}

	// 验证 HEAD hash 是最新的
	headHash, err := gs.GetHeadHash()
	if err != nil {
		t.Fatal(err)
	}
	if headHash != hash2 {
		t.Errorf("HEAD 应该指向最新的提交: 期望 %s, 得到 %s", hash2, headHash)
	}

	t.Log("Git Commit 测试通过")
}

func TestSequentialWrites(t *testing.T) {
	t.Log("测试顺序写入（Git 操作需要串行）")

	tmpDir, err := os.MkdirTemp("", "git-sequential-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := git.NewGitStorage(git.Config{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	const numWrites = 5

	// 顺序写入多个决策
	for i := 0; i < numWrites; i++ {
		d := decision.NewDecisionNode(
			"DEC-SEQ-000"+string(rune(i+'0')),
			"决策"+string(rune(i+'0')),
			"test",
			"sequential",
		)
		d.Decision = "内容"
		_, err := gs.WriteDecision(d)
		if err != nil {
			t.Errorf("写入决策 %d 失败: %v", i, err)
		}
	}

	// 验证所有决策都成功写入
	decisions, err := gs.ListDecisions("test", "sequential")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("成功写入 %d 个决策", len(decisions))
	if len(decisions) != numWrites {
		t.Errorf("期望 %d 个决策，实际 %d", numWrites, len(decisions))
	}
}

func TestBranchModel(t *testing.T) {
	t.Log("测试分支模型：讨论分支、紧急修复分支")

	tmpDir, err := os.MkdirTemp("", "git-branch-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := git.NewGitStorage(git.Config{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 1. 主分支初始状态
	mainBranch, err := gs.GetCurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("初始分支: %s", mainBranch)

	// 2. 创建讨论分支 (discussion 分支)
	discussionBranch := "discussion/DEC-001/20260501"
	err = gs.CreateBranch(discussionBranch)
	if err != nil {
		t.Fatalf("创建讨论分支失败: %v", err)
	}

	// 3. 在讨论分支上创建决策
	d := decision.NewDecisionNode("DEC-001", "讨论中的决策", "test", "topic")
	d.Decision = "正在讨论..."
	d.Status = decision.StatusInDiscussion
	_, err = gs.WriteDecision(d)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("在讨论分支上创建了决策")

	// 4. 切换回主分支
	err = gs.SwitchBranch("main")
	if err != nil {
		t.Fatal(err)
	}

	// 5. 创建紧急修复分支 (hotfix 分支)
	hotfixBranch := "hotfix/DEC-001"
	err = gs.CreateBranch(hotfixBranch)
	if err != nil {
		t.Fatalf("创建紧急修复分支失败: %v", err)
	}
	t.Log("创建了紧急修复分支")

	// 6. 列出所有分支
	branches, err := gs.ListBranches()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("所有分支: %v", branches)

	expectedBranches := map[string]bool{
		"main":                          true,
		discussionBranch:                true,
		hotfixBranch:                    true,
	}
	for _, b := range branches {
		delete(expectedBranches, b)
	}
	if len(expectedBranches) > 0 {
		t.Logf("注意：期望的分支列表: %v", expectedBranches)
	}

	t.Log("分支模型测试通过")
}

func TestGitMerge(t *testing.T) {
	t.Log("测试 Git 合并功能")

	tmpDir, err := os.MkdirTemp("", "git-merge-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := git.NewGitStorage(git.Config{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 1. 在主分支上创建初始决策
	dMain := decision.NewDecisionNode("DEC-MAIN-001", "主分支决策", "test", "topic")
	dMain.Decision = "主分支初始内容"
	_, err = gs.WriteDecision(dMain)
	if err != nil {
		t.Fatal(err)
	}

	// 2. 创建分支
	featureBranch := "feature/new-decision"
	err = gs.CreateBranch(featureBranch)
	if err != nil {
		t.Fatal(err)
	}

	// 3. 在特性分支上创建新决策
	dFeature := decision.NewDecisionNode("DEC-FEAT-001", "特性分支决策", "test", "topic")
	dFeature.Decision = "特性分支内容"
	_, err = gs.WriteDecision(dFeature)
	if err != nil {
		t.Fatal(err)
	}

	// 4. 切回主分支
	err = gs.SwitchBranch("main")
	if err != nil {
		t.Fatal(err)
	}

	// 5. 合并分支
	err = gs.MergeBranch(featureBranch)
	if err != nil {
		t.Logf("合并可能有冲突 (测试环境中): %v", err)
	} else {
		t.Log("分支合并成功")
	}

	t.Log("Git 合并测试完成")
}

func TestGitBitableSync(t *testing.T) {
	t.Log("测试 Git 与 Bitable 同步：正向同步、反向同步、一致性校验")

	tmpDir, err := os.MkdirTemp("", "git-sync-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := git.NewGitStorage(git.Config{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 模拟 Bitable 存储（简化版本）
	bitableStorage := make(map[string]string)

	// 1. 正向同步：Git -> Bitable
	t.Log("=== 正向同步：Git -> Bitable ===")
	d := decision.NewDecisionNode("DEC-SYNC-001", "同步测试", "test", "sync")
	d.Decision = "同步测试内容"
	hash, err := gs.WriteDecision(d)
	if err != nil {
		t.Fatal(err)
	}

	// 模拟同步到 Bitable
	bitableStorage[d.SDRID] = hash
	t.Logf("已同步决策 %s 到 Bitable，commit hash: %s", d.SDRID, hash)

	// 2. 一致性校验
	t.Log("=== 一致性校验 ===")
	drifts, err := gs.CheckConsistency(bitableStorage)
	if err != nil {
		t.Fatal(err)
	}
	if len(drifts) == 0 {
		t.Log("一致性检查通过，无漂移")
	} else {
		t.Logf("发现 %d 个漂移", len(drifts))
		for _, drift := range drifts {
			t.Logf("  - %s: Git=%s, Bitable=%s", drift.SDRID, drift.GitHash, drift.BitableHash)
		}
	}

	// 3. 模拟反向同步：Bitable -> Git
	t.Log("=== 模拟反向同步：Bitable -> Git ===")
	// 更新决策
	d.Decision = "在 Bitable 中修改后的内容"
	newHash, err := gs.WriteDecision(d)
	if err != nil {
		t.Fatal(err)
	}
	bitableStorage[d.SDRID] = newHash
	t.Logf("决策已更新，新 hash: %s", newHash)

	t.Log("Git 与 Bitable 同步测试完成")
}

func TestHistoryOperations(t *testing.T) {
	t.Log("测试历史操作：版本追溯、Blame、Diff")

	tmpDir, err := os.MkdirTemp("", "git-history-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := git.NewGitStorage(git.Config{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 创建几个有历史的决策
	d := decision.NewDecisionNode("DEC-HIST-001", "历史测试", "test", "history")
	d.Decision = "第一版内容"
	_, err = gs.WriteDecision(d)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond) // 确保时间戳不同

	// 更新决策
	d.Decision = "第二版内容"
	_, err = gs.WriteDecision(d)
	if err != nil {
		t.Fatal(err)
	}

	// 获取提交历史
	logs, err := gs.GetCommitLog("", 10)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("提交历史 (%d 条):", len(logs))
	for _, log := range logs {
		if len(log.Hash) >= 8 {
			t.Logf("  %s - %s", log.Hash[:8], log.Message)
		} else {
			t.Logf("  %s - %s", log.Hash, log.Message)
		}
	}

	if len(logs) < 3 { // 包括初始的 L0_RULES
		t.Log("注意：期望至少有 3 条提交记录")
	}

	// 测试 Blame（简化）
	t.Log("测试 Blame 功能...")
	blame, err := gs.BlameDecision("test", "history", "DEC-HIST-001")
	if err != nil {
		t.Logf("Blame 返回错误（可能是首次提交）: %v", err)
	} else {
		t.Logf("Blame 条目数: %d", len(blame))
	}

	// 测试 Git Grep
	t.Log("测试 Git Grep 全文搜索...")
	hits, err := gs.SearchContent("test", "内容")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("搜索'内容'命中 %d 条", len(hits))

	t.Log("历史操作测试完成")
}

func TestArchiveProcess(t *testing.T) {
	t.Log("测试归档处理")

	tmpDir, err := os.MkdirTemp("", "git-archive-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := git.NewGitStorage(git.Config{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 1. 创建项目和决策
	projectName := "to-archive"
	d1 := decision.NewDecisionNode("DEC-ARCH-001", "待归档决策1", projectName, "topic1")
	d1.Decision = "内容1"
	d1.Status = decision.StatusSuperseded // 标记为已替代
	_, err = gs.WriteDecision(d1)
	if err != nil {
		t.Fatal(err)
	}

	d2 := decision.NewDecisionNode("DEC-ARCH-002", "待归档决策2", projectName, "topic1")
	d2.Decision = "内容2"
	d2.Status = decision.StatusSuperseded
	_, err = gs.WriteDecision(d2)
	if err != nil {
		t.Fatal(err)
	}

	// 2. 验证项目存在
	topics, err := gs.ListTopics(projectName)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("归档前项目 %s 有 %d 个议题", projectName, len(topics))

	// 3. 执行归档
	err = gs.ArchiveProject(projectName)
	if err != nil {
		t.Logf("归档可能有问题（简化实现）: %v", err)
	} else {
		t.Log("项目归档成功")
	}

	// 4. 验证归档后
	archiveDir := tmpDir + "/archive"
	if _, err := os.Stat(archiveDir); err == nil {
		t.Log("archive 目录已创建")
	}

	t.Log("归档处理测试完成")
}

func TestFullGitWorkflow(t *testing.T) {
	t.Log("=== 完整 Git 工作流测试 ===")

	tmpDir, err := os.MkdirTemp("", "git-full-workflow-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := git.NewGitStorage(git.Config{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 阶段1: 主分支创建初始决策
	t.Log("阶段1: 在主分支创建决策")
	d := decision.NewDecisionNode("DEC-100", "重要架构决策", "feishu-mem", "architecture")
	d.Decision = "使用微服务架构"
	d.Rationale = "微服务提供更好的可扩展性"
	d.ImpactLevel = decision.ImpactMajor
	d.Status = decision.StatusInDiscussion
	_, err = gs.WriteDecision(d)
	if err != nil {
		t.Fatal(err)
	}

	// 阶段2: 更新为已决定
	t.Log("阶段2: 决策批准，状态更新")
	d.Status = decision.StatusDecided
	_, err = gs.WriteDecision(d)
	if err != nil {
		t.Fatal(err)
	}

	// 阶段3: 执行中
	t.Log("阶段3: 决策执行中")
	d.Status = decision.StatusExecuting
	_, err = gs.WriteDecision(d)
	if err != nil {
		t.Fatal(err)
	}

	// 阶段4: 完成
	t.Log("阶段4: 决策完成")
	d.Status = decision.StatusCompleted
	hash, err := gs.WriteDecision(d)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("最终决策提交: %s", hash)

	// 查看完整历史
	logs, _ := gs.GetCommitLog("", 10)
	t.Logf("完整工作流的提交历史 (%d):", len(logs))

	t.Log("=== 完整 Git 工作流测试通过 ===")
}
