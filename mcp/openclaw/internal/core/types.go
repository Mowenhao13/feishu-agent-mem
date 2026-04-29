package core

import "time"

// ImpactLevel 影响深度（decision-tree.md §3.1）
type ImpactLevel int

const (
	ImpactAdvisory ImpactLevel = iota // advisory — 影响单个 Topic，单 Phase
	ImpactMinor                       // minor — 影响单个 Topic，可能跨 Phase
	ImpactMajor                       // major — 影响一个或多个 Topic，可能跨 Phase
	ImpactCritical                    // critical — 影响整个项目，多个 Topic
)

func (l ImpactLevel) String() string {
	switch l {
	case ImpactAdvisory:
		return "advisory"
	case ImpactMinor:
		return "minor"
	case ImpactMajor:
		return "major"
	case ImpactCritical:
		return "critical"
	default:
		return "unknown"
	}
}

func ParseImpactLevel(s string) ImpactLevel {
	switch s {
	case "advisory":
		return ImpactAdvisory
	case "minor":
		return ImpactMinor
	case "major":
		return ImpactMajor
	case "critical":
		return ImpactCritical
	default:
		return ImpactAdvisory
	}
}

// DecisionStatus 决策状态（decision-tree.md §1.3）
type DecisionStatus int

const (
	StatusPending      DecisionStatus = iota // 待决策
	StatusInDiscussion                       // 决策中
	StatusDecided                            // 已决策
	StatusExecuting                          // 执行中
	StatusCompleted                          // 已完成
	StatusShelved                            // 搁置
	StatusRejected                           // 否决
	StatusSuperseded                         // 已被替代
	StatusDeprecated                         // 已废弃
)

func (s DecisionStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusInDiscussion:
		return "in_discussion"
	case StatusDecided:
		return "decided"
	case StatusExecuting:
		return "executing"
	case StatusCompleted:
		return "completed"
	case StatusShelved:
		return "shelved"
	case StatusRejected:
		return "rejected"
	case StatusSuperseded:
		return "superseded"
	case StatusDeprecated:
		return "deprecated"
	default:
		return "unknown"
	}
}

func ParseDecisionStatus(s string) DecisionStatus {
	switch s {
	case "pending":
		return StatusPending
	case "in_discussion":
		return StatusInDiscussion
	case "decided":
		return StatusDecided
	case "executing":
		return StatusExecuting
	case "completed":
		return StatusCompleted
	case "shelved":
		return StatusShelved
	case "rejected":
		return StatusRejected
	case "superseded":
		return StatusSuperseded
	case "deprecated":
		return StatusDeprecated
	default:
		return StatusPending
	}
}

// PhaseScope 阶段范围（decision-tree.md §2.1）
type PhaseScope int

const (
	PhaseScopePoint       PhaseScope = iota // Point — 单阶段
	PhaseScopeSpan                          // Span — 跨阶段
	PhaseScopeRetroactive                   // Retroactive — 追溯生效
)

func (s PhaseScope) String() string {
	switch s {
	case PhaseScopePoint:
		return "Point"
	case PhaseScopeSpan:
		return "Span"
	case PhaseScopeRetroactive:
		return "Retroactive"
	default:
		return "Point"
	}
}

func ParsePhaseScope(s string) PhaseScope {
	switch s {
	case "Point":
		return PhaseScopePoint
	case "Span":
		return PhaseScopeSpan
	case "Retroactive":
		return PhaseScopeRetroactive
	default:
		return PhaseScopePoint
	}
}

// RelationType 关系类型（decision-tree.md §2.1）
type RelationType int

const (
	RelationDependsOn     RelationType = iota // DEPENDS_ON
	RelationSupersedes                        // SUPERSEDES
	RelationRefines                           // REFINES
	RelationConflictsWith                     // CONFLICTS_WITH
)

func (r RelationType) String() string {
	switch r {
	case RelationDependsOn:
		return "DEPENDS_ON"
	case RelationSupersedes:
		return "SUPERSEDES"
	case RelationRefines:
		return "REFINES"
	case RelationConflictsWith:
		return "CONFLICTS_WITH"
	default:
		return "UNKNOWN"
	}
}

// VersionRange 版本范围（decision-tree.md §2.1）
type VersionRange struct {
	From string `yaml:"from" json:"from"`
	To   string `yaml:"to,omitempty" json:"to,omitempty"` // 空值表示当前生效
}

// Relation 关系边（decision-tree.md §2.1）
type Relation struct {
	Type      RelationType `yaml:"type" json:"type"`
	TargetSDR string       `yaml:"target_sdr_id" json:"target_sdr_id"`
}

// FeishuLinks 飞书实体关联（decision-extraction.md §2 / openclaw-architecture.md §2.1.3）
type FeishuLinks struct {
	RelatedChatIDs      []string `yaml:"related_chat_ids,omitempty" json:"related_chat_ids,omitempty"`
	RelatedMessageIDs   []string `yaml:"related_message_ids,omitempty" json:"related_message_ids,omitempty"`
	RelatedDocTokens    []string `yaml:"related_doc_tokens,omitempty" json:"related_doc_tokens,omitempty"`
	RelatedEventIDs     []string `yaml:"related_event_ids,omitempty" json:"related_event_ids,omitempty"`
	RelatedMeetingIDs   []string `yaml:"related_meeting_ids,omitempty" json:"related_meeting_ids,omitempty"`
	RelatedTaskGUIDs    []string `yaml:"related_task_guids,omitempty" json:"related_task_guids,omitempty"`
	RelatedMinuteTokens []string `yaml:"related_minute_tokens,omitempty" json:"related_minute_tokens,omitempty"`
}

// AdapterType 适配器类型（signal-activation-engine.md §1）
type AdapterType string

const (
	AdapterIM       AdapterType = "IM"
	AdapterVC       AdapterType = "VC"
	AdapterDocs     AdapterType = "Docs"
	AdapterCalendar AdapterType = "Calendar"
	AdapterTask     AdapterType = "Task"
	AdapterOKR      AdapterType = "OKR"
	AdapterContact  AdapterType = "Contact"
	AdapterWiki     AdapterType = "Wiki"
)

// ChangeType 变更类型（signal-activation-engine.md §1）
type ChangeType string

const (
	ChangeCreated ChangeType = "created"
	ChangeUpdated ChangeType = "updated"
	ChangeDeleted ChangeType = "deleted"
	ChangeStatus  ChangeType = "status_change"
)

// SignalStrength 信号强度（signal-activation-engine.md §1）
type SignalStrength int

const (
	SignalWeak   SignalStrength = iota // Weak
	SignalMedium                       // Medium
	SignalStrong                       // Strong
)

func (s SignalStrength) String() string {
	switch s {
	case SignalWeak:
		return "weak"
	case SignalMedium:
		return "medium"
	case SignalStrong:
		return "strong"
	default:
		return "unknown"
	}
}

// ---- 辅助类型 ----

// SearchOpts 搜索选项
type SearchOpts struct {
	Project     string
	Topic       string
	Keywords    string
	Status      string
	ImpactLevel string
	Limit       int
}

// SearchHit 搜索结果条目
type SearchHit struct {
	FilePath string
	SDRID    string
	Title    string
	Topic    string
	Score    float64
}

// Conflict 决策冲突（decision-tree.md §5.3）
type Conflict struct {
	ConflictID         string     `json:"conflict_id"`
	DecisionA          string     `json:"decision_a"` // SDR ID
	DecisionB          string     `json:"decision_b"` // SDR ID
	ContradictionScore float64    `json:"contradiction_score"`
	Description        string     `json:"description"`
	Status             string     `json:"status"` // pending | resolved
	CreatedAt          time.Time  `json:"created_at"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`
}

// BlameEntry Git blame 条目
type BlameEntry struct {
	Line   int       `json:"line"`
	Commit string    `json:"commit"`
	Author string    `json:"author"`
	Date   time.Time `json:"date"`
}

// PipelineReport 流水线执行报告
type PipelineReport struct {
	DecisionID string    `json:"decision_id"`
	Action     string    `json:"action"` // created | updated | deleted
	CommitHash string    `json:"commit_hash"`
	Error      string    `json:"error,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// Priority 优先级（signal-activation-engine.md §2.2）
type Priority int

const (
	PriorityMust   Priority = 0
	PriorityShould Priority = 1
	PriorityMay    Priority = 2
	PrioritySkip   Priority = 3
)
