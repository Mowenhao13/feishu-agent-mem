package core

import (
	"log"
	"sync"
)

// Extractor 提取器接口 — 从飞书提取原始数据
type Extractor interface {
	Extract(event *LarkEvent) (interface{}, error)
}

// Processor 处理器接口 — 将原始数据处理为决策变更
type Processor interface {
	Process(raw interface{}) (*DecisionMutation, error)
}

// Syncer 同步器接口 — 持久化决策变更到 Git/Bitable
type Syncer interface {
	Sync(mutation *DecisionMutation) error
}

// DecisionMutation 决策变更操作（signal-activation-engine.md §14.1）
type DecisionMutation struct {
	Type          string                 // "create" | "update" | "status_change" | "conflict"
	SDRID         string                 // 目标决策 ID
	Fields        map[string]interface{} // 变更的字段
	CommitMessage string                 // Git commit message
	Decision      *DecisionNode          // 完整决策（create 时）
}

// PipelineEngine 流程引擎 — 编排提取 → 处理 → 同步（openclaw-architecture.md §2.1.1）
type PipelineEngine struct {
	extractors []Extractor
	processors []Processor
	syncers    []Syncer
	mu         sync.Mutex
}

// NewPipelineEngine 创建流程引擎
func NewPipelineEngine() *PipelineEngine {
	return &PipelineEngine{
		extractors: make([]Extractor, 0),
		processors: make([]Processor, 0),
		syncers:    make([]Syncer, 0),
	}
}

// RegisterExtractor 注册提取器
func (pe *PipelineEngine) RegisterExtractor(e Extractor) {
	pe.extractors = append(pe.extractors, e)
}

// RegisterProcessor 注册处理器
func (pe *PipelineEngine) RegisterProcessor(p Processor) {
	pe.processors = append(pe.processors, p)
}

// RegisterSyncer 注册同步器
func (pe *PipelineEngine) RegisterSyncer(s Syncer) {
	pe.syncers = append(pe.syncers, s)
}

// OnEvent 事件驱动入口 — 收到飞书事件后触发增量流水线（openclaw-architecture.md §2.1.1）
func (pe *PipelineEngine) OnEvent(event *LarkEvent) *PipelineReport {
	report := &PipelineReport{Timestamp: Now()}

	// Stage 1: 提取原始数据
	var rawResults []interface{}
	for _, ext := range pe.extractors {
		raw, err := ext.Extract(event)
		if err != nil {
			log.Printf("Extractor error: %v", err)
			continue
		}
		if raw != nil {
			rawResults = append(rawResults, raw)
		}
	}

	if len(rawResults) == 0 {
		return nil
	}

	// Stage 2: 处理为决策变更
	var mutations []*DecisionMutation
	for _, raw := range rawResults {
		for _, proc := range pe.processors {
			mutation, err := proc.Process(raw)
			if err != nil {
				log.Printf("Processor error: %v", err)
				continue
			}
			if mutation != nil {
				mutations = append(mutations, mutation)
			}
		}
	}

	if len(mutations) == 0 {
		return nil
	}

	// Stage 3: 同步
	for _, mutation := range mutations {
		report.DecisionID = mutation.SDRID
		report.Action = mutation.Type

		for _, syncer := range pe.syncers {
			if err := syncer.Sync(mutation); err != nil {
				report.Error = err.Error()
				return report
			}
		}
	}

	return report
}

// Poll 主动轮询入口 — 定时任务批量提取（openclaw-architecture.md §2.1.1）
func (pe *PipelineEngine) Poll() []*PipelineReport {
	return nil
}

// LarkEvent 简化的飞书事件结构
type LarkEvent struct {
	Type   string `json:"type"`
	Schema string `json:"schema"`
	Header *EventHeader
	Event  interface{} `json:"event"`
	Raw    []byte      `json:"-"`
}

// EventHeader 事件头
type EventHeader struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	AppID      string `json:"app_id"`
	TenantKey  string `json:"tenant_key"`
	CreateTime string `json:"create_time"`
}

// Now 返回当前时间（可 mock）
var Now = func() Timestamp {
	return Timestamp{Unix: 0}
}

// Timestamp 时间戳
type Timestamp struct {
	Unix int64
}
