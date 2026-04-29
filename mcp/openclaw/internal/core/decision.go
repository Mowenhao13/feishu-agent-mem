package core

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"strings"
	"time"
)

// DecisionNode 决策节点 — 与 decision-tree.md §2.1 / openclaw-architecture.md §2.1.3 严格对齐
type DecisionNode struct {
	// === 标识与内容 ===
	SDRID         string `yaml:"sdr_id" json:"sdr_id"`
	GitCommitHash string `yaml:"git_commit_hash" json:"git_commit_hash"`
	Title         string `yaml:"title" json:"title"`
	Decision      string `yaml:"decision" json:"decision"`
	Rationale     string `yaml:"rationale" json:"rationale"`

	// === 所属（树位置） ===
	Project string `yaml:"project" json:"project"`
	Topic   string `yaml:"topic" json:"topic"` // 唯一位置锚点

	// === 时态标签 ===
	Phase       string     `yaml:"phase" json:"phase"`
	PhaseScope  PhaseScope `yaml:"phase_scope" json:"phase_scope"`
	VersionFrom string     `yaml:"version_from" json:"version_from"`
	VersionTo   string     `yaml:"version_to,omitempty" json:"version_to,omitempty"`

	// === 影响级别 ===
	ImpactLevel ImpactLevel `yaml:"impact_level" json:"impact_level"`

	// === 跨议题影响 ===
	CrossTopicRefs []string `yaml:"cross_topic_refs,omitempty" json:"cross_topic_refs,omitempty"`

	// === 树内父子关系 ===
	ParentDecision string `yaml:"parent_decision,omitempty" json:"parent_decision,omitempty"`
	ChildrenCount  int    `yaml:"children_count" json:"children_count"`

	// === 人员 ===
	Proposer     string   `yaml:"proposer" json:"proposer"`
	Executor     string   `yaml:"executor,omitempty" json:"executor,omitempty"`
	Stakeholders []string `yaml:"stakeholders,omitempty" json:"stakeholders,omitempty"`

	// === 关系图谱 ===
	Relations []Relation `yaml:"relations,omitempty" json:"relations,omitempty"`

	// === 飞书关联 ===
	FeishuLinks FeishuLinks `yaml:"feishu_links,omitempty" json:"feishu_links,omitempty"`

	// === 状态 ===
	Status    DecisionStatus `yaml:"status" json:"status"`
	CreatedAt time.Time      `yaml:"created_at" json:"created_at"`
	DecidedAt *time.Time     `yaml:"decided_at,omitempty" json:"decided_at,omitempty"`

	// 内容 body（非 frontmatter 部分）
	Body string `yaml:"-" json:"body,omitempty"`
}

// YAMLFrontmatter 用于序列化 frontmatter 的中间结构
type decisionYAML struct {
	SDRID          string      `yaml:"sdr_id"`
	GitCommitHash  string      `yaml:"git_commit_hash"`
	Title          string      `yaml:"title"`
	Decision       string      `yaml:"decision"`
	Rationale      string      `yaml:"rationale"`
	Project        string      `yaml:"project"`
	Topic          string      `yaml:"topic"`
	Phase          string      `yaml:"phase"`
	PhaseScope     string      `yaml:"phase_scope"`
	VersionFrom    string      `yaml:"version_from"`
	VersionTo      string      `yaml:"version_to,omitempty"`
	ImpactLevel    string      `yaml:"impact_level"`
	CrossTopicRefs []string    `yaml:"cross_topic_refs,omitempty"`
	ParentDecision string      `yaml:"parent_decision,omitempty"`
	ChildrenCount  int         `yaml:"children_count"`
	Proposer       string      `yaml:"proposer"`
	Executor       string      `yaml:"executor,omitempty"`
	Stakeholders   []string    `yaml:"stakeholders,omitempty"`
	Relations      []Relation  `yaml:"relations,omitempty"`
	FeishuLinks    FeishuLinks `yaml:"feishu_links,omitempty"`
	Status         string      `yaml:"status"`
	CreatedAt      string      `yaml:"created_at"`
	DecidedAt      string      `yaml:"decided_at,omitempty"`
}

// ToYAML 将 DecisionNode 转换为 YAML frontmatter
func (d *DecisionNode) ToYAML() ([]byte, error) {
	dy := decisionYAML{
		SDRID:          d.SDRID,
		GitCommitHash:  d.GitCommitHash,
		Title:          d.Title,
		Decision:       d.Decision,
		Rationale:      d.Rationale,
		Project:        d.Project,
		Topic:          d.Topic,
		Phase:          d.Phase,
		PhaseScope:     d.PhaseScope.String(),
		VersionFrom:    d.VersionFrom,
		VersionTo:      d.VersionTo,
		ImpactLevel:    d.ImpactLevel.String(),
		CrossTopicRefs: d.CrossTopicRefs,
		ParentDecision: d.ParentDecision,
		ChildrenCount:  d.ChildrenCount,
		Proposer:       d.Proposer,
		Executor:       d.Executor,
		Stakeholders:   d.Stakeholders,
		Relations:      d.Relations,
		FeishuLinks:    d.FeishuLinks,
		Status:         d.Status.String(),
		CreatedAt:      d.CreatedAt.Format(time.RFC3339),
	}
	if d.DecidedAt != nil {
		dy.DecidedAt = d.DecidedAt.Format(time.RFC3339)
	}
	return yaml.Marshal(dy)
}

// FromYAML 从 YAML 解析 DecisionNode
func (d *DecisionNode) FromYAML(data []byte) error {
	var dy decisionYAML
	if err := yaml.Unmarshal(data, &dy); err != nil {
		return fmt.Errorf("parse decision YAML: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, dy.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	d.SDRID = dy.SDRID
	d.GitCommitHash = dy.GitCommitHash
	d.Title = dy.Title
	d.Decision = dy.Decision
	d.Rationale = dy.Rationale
	d.Project = dy.Project
	d.Topic = dy.Topic
	d.Phase = dy.Phase
	d.PhaseScope = ParsePhaseScope(dy.PhaseScope)
	d.VersionFrom = dy.VersionFrom
	d.VersionTo = dy.VersionTo
	d.ImpactLevel = ParseImpactLevel(dy.ImpactLevel)
	d.CrossTopicRefs = dy.CrossTopicRefs
	d.ParentDecision = dy.ParentDecision
	d.ChildrenCount = dy.ChildrenCount
	d.Proposer = dy.Proposer
	d.Executor = dy.Executor
	d.Stakeholders = dy.Stakeholders
	d.Relations = dy.Relations
	d.FeishuLinks = dy.FeishuLinks
	d.Status = ParseDecisionStatus(dy.Status)
	d.CreatedAt = createdAt

	if dy.DecidedAt != "" {
		t, err := time.Parse(time.RFC3339, dy.DecidedAt)
		if err == nil {
			d.DecidedAt = &t
		}
	}

	return nil
}

// RenderFile 渲染决策的完整 Markdown 文件（YAML frontmatter + body）
func (d *DecisionNode) RenderFile() ([]byte, error) {
	fm, err := d.ToYAML()
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(fm)
	b.WriteString("---\n\n")
	if d.Body != "" {
		b.WriteString(d.Body)
	} else {
		b.WriteString(d.Decision + "\n")
	}

	return []byte(b.String()), nil
}

// ParseDecisionFile 解析完整的决策 Markdown 文件
func ParseDecisionFile(content []byte) (*DecisionNode, error) {
	text := string(content)

	// 查找 frontmatter 边界
	if !strings.HasPrefix(text, "---\n") {
		return nil, fmt.Errorf("missing YAML frontmatter delimiter")
	}

	rest := text[4:] // skip "---\n"
	endIdx := strings.Index(rest, "\n---\n")
	if endIdx < 0 {
		return nil, fmt.Errorf("missing closing YAML frontmatter delimiter")
	}

	fmYAML := rest[:endIdx]
	body := strings.TrimSpace(rest[endIdx+5:])

	node := &DecisionNode{}
	if err := node.FromYAML([]byte(fmYAML)); err != nil {
		return nil, err
	}
	node.Body = body

	return node, nil
}

// DecisionPath 返回 Git 中决策文件的相对路径: decisions/{project}/{topic}/{sdr_id}.md
func DecisionPath(project, topic, sdrID string) string {
	return fmt.Sprintf("decisions/%s/%s/%s.md", project, topic, sdrID)
}

// SDRID 生成唯一决策 ID: dec_{YYYYMMDD}_{seq}
func GenerateSDRID(seq int) string {
	now := time.Now()
	return fmt.Sprintf("dec_%s_%03d", now.Format("20060102"), seq)
}
