package adapters

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/openclaw/internal/core"
)

// GitStorage Git 存储封装 — 决策文件的读写、版本控制（openclaw-architecture.md §2.3 / git-operations-design.md）
type GitStorage struct {
	workDir  string
	remote   string
	autoPush bool
	branch   string

	mu    sync.Mutex
	queue chan CommitRequest
}

// CommitRequest Git 提交请求（git-operations-design.md §4.4）
type CommitRequest struct {
	FilePath string
	Content  []byte
	Message  string
	ResultCh chan CommitResult
}

// CommitResult 提价结果
type CommitResult struct {
	Hash string
	Err  error
}

// NewGitStorage 初始化 Git 仓库并创建 GitStorage
func NewGitStorage(workDir, remote string, autoPush bool) (*GitStorage, error) {
	gs := &GitStorage{
		workDir:  workDir,
		remote:   remote,
		autoPush: autoPush,
		branch:   "main",
		queue:    make(chan CommitRequest, 100),
	}

	// 确保目录存在
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}

	// 检查是否已有仓库
	gitDir := filepath.Join(workDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		if err := gs.initRepo(); err != nil {
			return nil, fmt.Errorf("init repo: %w", err)
		}
	}

	// 启动队列处理器
	go gs.processQueue()

	return gs, nil
}

// initRepo 初始化 Git 仓库（git-operations-design.md §12.1）
func (gs *GitStorage) initRepo() error {
	// git init
	cmd := exec.Command("git", "init", "-b", gs.branch, gs.workDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init failed: %s", output)
	}

	// 创建基础目录
	dirs := []string{
		filepath.Join(gs.workDir, "decisions"),
		filepath.Join(gs.workDir, "conflicts"),
		filepath.Join(gs.workDir, "archive"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", d, err)
		}
	}

	// 写入 L0_RULES.md
	l0Rules := `# L0 核心规则

> 本文件中的规则不可被决策覆盖。违反 L0 规则的决策将被阻断。

## 规则列表

### L0-001: 数据安全
- 不得在决策文件中存储密码、密钥、Token 等敏感信息
- 数据库连接字符串使用环境变量引用

### L0-002: 向后兼容
- 涉及 API 变更的决策必须包含迁移方案
- 废弃 API 需保留至少一个版本周期

### L0-003: 审批流程
- impact_level 为 critical 的决策必须经过两人以上审批
- 涉及生产环境的决策必须包含回滚方案
`
	if err := os.WriteFile(filepath.Join(gs.workDir, "L0_RULES.md"), []byte(l0Rules), 0644); err != nil {
		return err
	}

	// 写入 .gitignore
	gitignore := ".DS_Store\n__pycache__/\n*.pyc\n"
	if err := os.WriteFile(filepath.Join(gs.workDir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		return err
	}

	// 初始 commit
	if err := gs.gitCommitAll("init: 初始化 OpenClaw 决策仓库"); err != nil {
		return err
	}

	// 设置远程仓库
	if gs.remote != "" {
		cmd = exec.Command("git", "remote", "add", "origin", gs.remote)
		cmd.Dir = gs.workDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git remote add failed: %s", output)
		}
	}

	return nil
}

// WriteDecision 写决策文件（原子写入 + git commit）（openclaw-architecture.md §2.3）
func (gs *GitStorage) WriteDecision(node *core.DecisionNode) (string, error) {
	path := core.DecisionPath(node.Project, node.Topic, node.SDRID)
	fullPath := filepath.Join(gs.workDir, path)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("create decision dir: %w", err)
	}

	// 渲染文件
	content, err := node.RenderFile()
	if err != nil {
		return "", fmt.Errorf("render decision file: %w", err)
	}

	// 构建 commit message
	msg := fmt.Sprintf("decision(%s): %s — %s\n\n决策: %s\n依据: %s",
		node.Topic, node.SDRID, node.Title, node.Decision, node.Rationale)

	// 通过队列提交
	resultCh := make(chan CommitResult, 1)
	gs.queue <- CommitRequest{
		FilePath: fullPath,
		Content:  content,
		Message:  msg,
		ResultCh: resultCh,
	}

	result := <-resultCh
	if result.Err != nil {
		return "", result.Err
	}

	return result.Hash, nil
}

// ReadDecision 读决策文件（openclaw-architecture.md §2.3）
func (gs *GitStorage) ReadDecision(project, topic, sdrID string) (*core.DecisionNode, error) {
	path := core.DecisionPath(project, topic, sdrID)
	fullPath := filepath.Join(gs.workDir, path)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read decision file %s: %w", fullPath, err)
	}

	return core.ParseDecisionFile(content)
}

// ListDecisions 按议题列出所有决策文件（openclaw-architecture.md §2.3）
func (gs *GitStorage) ListDecisions(project, topic string) ([]*core.DecisionNode, error) {
	dir := filepath.Join(gs.workDir, "decisions", project, topic)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list decisions dir: %w", err)
	}

	var decisions []*core.DecisionNode
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if entry.Name() == "topic-index.md" {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		node, err := core.ParseDecisionFile(content)
		if err != nil {
			continue
		}
		decisions = append(decisions, node)
	}

	return decisions, nil
}

// SearchContent Git grep 全文搜索（git-operations-design.md §10）
func (gs *GitStorage) SearchContent(project, query string) ([]core.SearchHit, error) {
	args := []string{"grep", "-n", "--", query}
	if project != "" {
		args = append(args, filepath.Join("decisions", project))
	} else {
		args = append(args, "decisions/")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = gs.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		// git grep 返回 1 表示无结果
		return nil, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var hits []core.SearchHit
	for _, line := range lines {
		if line == "" {
			continue
		}
		hits = append(hits, core.SearchHit{
			FilePath: line,
		})
	}
	return hits, nil
}

// BlameDecision 获取决策文件的版本历史（git-operations-design.md §9.2）
func (gs *GitStorage) BlameDecision(project, topic, sdrID string) ([]core.BlameEntry, error) {
	path := core.DecisionPath(project, topic, sdrID)
	cmd := exec.Command("git", "blame", "--porcelain", "--", path)
	cmd.Dir = gs.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git blame: %w", err)
	}

	return parseBlameOutput(string(output)), nil
}

// Diff 对比决策的两个版本（git-operations-design.md §9.3）
func (gs *GitStorage) Diff(commit1, commit2, project, topic, sdrID string) (string, error) {
	path := core.DecisionPath(project, topic, sdrID)
	cmd := exec.Command("git", "diff", commit1+".."+commit2, "--", path)
	cmd.Dir = gs.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(output), nil
}

// Log 查看某决策的修改历史（git-operations-design.md §9.1）
func (gs *GitStorage) Log(project, topic, sdrID string) (string, error) {
	path := core.DecisionPath(project, topic, sdrID)
	cmd := exec.Command("git", "log", "--follow", "--format=%H %ai %s", "--", path)
	cmd.Dir = gs.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git log: %w", err)
	}
	return string(output), nil
}

// WriteConflict 写入冲突文件（git-operations-design.md §6.2）
func (gs *GitStorage) WriteConflict(conflict *core.Conflict) (string, error) {
	path := filepath.Join("conflicts", conflict.ConflictID+".md")
	fullPath := filepath.Join(gs.workDir, path)

	content := fmt.Sprintf(`# 冲突: %s

- 决策 A: %s
- 决策 B: %s
- 矛盾评分: %.2f
- 描述: %s
- 状态: %s
- 创建时间: %s
`,
		conflict.ConflictID,
		conflict.DecisionA,
		conflict.DecisionB,
		conflict.ContradictionScore,
		conflict.Description,
		conflict.Status,
		conflict.CreatedAt.Format(time.RFC3339),
	)

	resultCh := make(chan CommitResult, 1)
	gs.queue <- CommitRequest{
		FilePath: fullPath,
		Content:  []byte(content),
		Message:  fmt.Sprintf("resolve: 发现冲突 %s", conflict.ConflictID),
		ResultCh: resultCh,
	}

	result := <-resultCh
	return result.Hash, result.Err
}

// ListProjects 列出所有项目
func (gs *GitStorage) ListProjects() ([]string, error) {
	dir := filepath.Join(gs.workDir, "decisions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var projects []string
	for _, e := range entries {
		if e.IsDir() {
			projects = append(projects, e.Name())
		}
	}
	sort.Strings(projects)
	return projects, nil
}

// ListTopics 列出某项目的所有议题
func (gs *GitStorage) ListTopics(project string) ([]string, error) {
	dir := filepath.Join(gs.workDir, "decisions", project)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var topics []string
	for _, e := range entries {
		if e.IsDir() {
			topics = append(topics, e.Name())
		}
	}
	sort.Strings(topics)
	return topics, nil
}

// ---- 内部方法 ----

// processQueue 处理提交队列（git-operations-design.md §4.4）
func (gs *GitStorage) processQueue() {
	for req := range gs.queue {
		gs.mu.Lock()

		// 写入文件
		if err := os.WriteFile(req.FilePath, req.Content, 0644); err != nil {
			req.ResultCh <- CommitResult{Err: fmt.Errorf("write file: %w", err)}
			gs.mu.Unlock()
			continue
		}

		// git add
		relPath, _ := filepath.Rel(gs.workDir, req.FilePath)
		addCmd := exec.Command("git", "add", relPath)
		addCmd.Dir = gs.workDir
		if output, err := addCmd.CombinedOutput(); err != nil {
			req.ResultCh <- CommitResult{Err: fmt.Errorf("git add failed: %s", output)}
			gs.mu.Unlock()
			continue
		}

		// git commit
		hash, err := gs.gitCommit(req.Message)
		if err != nil {
			req.ResultCh <- CommitResult{Err: err}
			gs.mu.Unlock()
			continue
		}

		// auto push
		if gs.autoPush && gs.remote != "" {
			pushCmd := exec.Command("git", "push", "origin", gs.branch)
			pushCmd.Dir = gs.workDir
			if output, err := pushCmd.CombinedOutput(); err != nil {
				_ = fmt.Sprintf("git push failed (non-fatal): %s", output)
			}
		}

		gs.mu.Unlock()
		req.ResultCh <- CommitResult{Hash: hash}
	}
}

func (gs *GitStorage) gitCommit(message string) (string, error) {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = gs.workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git commit failed: %s", output)
	}

	// 获取 commit hash
	hashCmd := exec.Command("git", "rev-parse", "HEAD")
	hashCmd.Dir = gs.workDir
	hashOutput, err := hashCmd.Output()
	if err != nil {
		return "", fmt.Errorf("get commit hash: %w", err)
	}

	return strings.TrimSpace(string(hashOutput)), nil
}

func (gs *GitStorage) gitCommitAll(message string) error {
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = gs.workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add -A failed: %s", output)
	}

	_, err := gs.gitCommit(message)
	return err
}

// parseBlameOutput 解析 git blame --porcelain 输出
func parseBlameOutput(output string) []core.BlameEntry {
	var entries []core.BlameEntry
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		entries = append(entries, core.BlameEntry{
			Line:   i + 1,
			Commit: parts[0],
		})
	}
	return entries
}
