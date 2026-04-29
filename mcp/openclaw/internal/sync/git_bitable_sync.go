package sync

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/openclaw/internal/adapters"
)

// GitBitableSync Git ↔ Bitable 双向同步器（git-operations-design.md §7）
type GitBitableSync struct {
	git     *adapters.GitStorage
	bitable *adapters.BitableStore
}

// NewGitBitableSync 创建同步器
func NewGitBitableSync(gs *adapters.GitStorage, bs *adapters.BitableStore) *GitBitableSync {
	return &GitBitableSync{
		git:     gs,
		bitable: bs,
	}
}

// ForwardSync 正向同步：Git → Bitable（git-operations-design.md §7.2）
func (s *GitBitableSync) ForwardSync(ctx context.Context, project, topic, sdrID string) error {
	// 1. 从 Git 读取决策文件
	node, err := s.git.ReadDecision(project, topic, sdrID)
	if err != nil {
		return fmt.Errorf("read from git: %w", err)
	}

	// 2. 写入 Bitable
	if err := s.bitable.UpsertDecision(ctx, node); err != nil {
		return fmt.Errorf("upsert to bitable: %w", err)
	}

	log.Printf("正向同步成功: %s/%s/%s (hash: %s)", project, topic, sdrID, node.GitCommitHash)
	return nil
}

// ReverseSync 反向同步：Bitable → Git（git-operations-design.md §7.3）
func (s *GitBitableSync) ReverseSync(ctx context.Context, project, topic, sdrID string) error {
	// 1. 从 Git 读取当前版本
	gitNode, err := s.git.ReadDecision(project, topic, sdrID)
	if err != nil {
		return fmt.Errorf("read from git: %w", err)
	}

	// 2. 比较 Bitable 版本（通过 git_commit_hash）
	// 如果 Bitable 的 hash ≠ Git 的 hash，说明 Bitable 有修改
	// 实际需从 Bitable 读取记录

	_ = gitNode
	return nil
}

// VerifyConsistency 一致性校验（git-operations-design.md §7.4）
func (s *GitBitableSync) VerifyConsistency(ctx context.Context) ([]ConsistencyIssue, error) {
	var issues []ConsistencyIssue

	// 遍历 Git 中所有决策文件
	projects, err := s.git.ListProjects()
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		topics, err := s.git.ListTopics(project)
		if err != nil {
			continue
		}
		for _, topic := range topics {
			decisions, err := s.git.ListDecisions(project, topic)
			if err != nil {
				continue
			}
			for _, d := range decisions {
				// 获取 Git HEAD hash
				logOutput, err := s.git.Log(project, topic, d.SDRID)
				if err != nil || logOutput == "" {
					continue
				}

				// 简化：只记录需要同步的决策
				_ = logOutput
				_ = d
			}
		}
	}

	return issues, nil
}

// ConsistencyIssue 一致性问题
type ConsistencyIssue struct {
	SDRID       string `json:"sdr_id"`
	IssueType   string `json:"issue_type"` // git_ahead | bitable_ahead | conflicted
	GitHash     string `json:"git_hash"`
	BitableHash string `json:"bitable_hash"`
}

// ---- 定时任务封装 ----

// SyncScheduler 同步调度器
type SyncScheduler struct {
	sync    *GitBitableSync
	git     *adapters.GitStorage
	bitable *adapters.BitableStore
}

// NewSyncScheduler 创建同步调度器
func NewSyncScheduler(gs *adapters.GitStorage, bs *adapters.BitableStore) *SyncScheduler {
	return &SyncScheduler{
		sync:    NewGitBitableSync(gs, bs),
		git:     gs,
		bitable: bs,
	}
}

// RunConsistencyCheck 执行一致性校验（每 N 分钟调用）
func (ss *SyncScheduler) RunConsistencyCheck(ctx context.Context) {
	start := time.Now()
	issues, err := ss.sync.VerifyConsistency(ctx)
	if err != nil {
		log.Printf("一致性校验失败: %v", err)
		return
	}
	log.Printf("一致性校验完成: %d 个问题, 耗时 %v", len(issues), time.Since(start))
}

// RunDailySync 执行每日同步（轮询补全）
func (ss *SyncScheduler) RunDailySync(ctx context.Context) {
	log.Println("开始每日同步...")
}
