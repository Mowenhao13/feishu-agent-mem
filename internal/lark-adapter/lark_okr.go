package larkadapter

import (
	"encoding/json"
	"fmt"
	"time"
)

// OKRExtractor OKR 提取器
type OKRExtractor struct {
	config *Config
	cli    *LarkCLI
}

// NewOKRExtractor 创建 OKR 提取器
func NewOKRExtractor(cfg *Config) *OKRExtractor {
	return &OKRExtractor{
		config: cfg,
		cli:    NewLarkCLI(),
	}
}

// Name 实现 Extractor 接口
func (e *OKRExtractor) Name() string {
	return "lark_okr"
}

// Detect 检测 OKR 变化（新周期、进度更新）
func (e *OKRExtractor) Detect(lastCheck time.Time) (*DetectResult, error) {
	changes := []Change{}

	// 检测 OKR 周期变化
	cycleResult, err := e.getCycleList()
	if err == nil {
		newCycles := e.parseNewCycles(cycleResult, lastCheck)
		changes = append(changes, newCycles...)
	}

	result := &DetectResult{
		Source:     e.Name(),
		HasChanges: len(changes) > 0,
		DetectedAt: time.Now(),
		LastCheck:  lastCheck,
		Changes:    changes,
	}

	_ = SaveDetectResult(result)
	return result, nil
}

// Extract 提取 OKR 决策信息
func (e *OKRExtractor) Extract() error {
	rawData := make(map[string]any)

	if cycles, err := e.getCycleList(); err == nil {
		rawData["cycles"] = cycles
	}

	formatted := map[string]any{
		"extracted": true,
	}

	result := &ExtractionResult{
		Source:      e.Name(),
		ExtractedAt: time.Now(),
		RawData:     rawData,
		Formatted:   formatted,
	}

	if err := SaveToJSON(e.Name(), result); err != nil {
		return fmt.Errorf("save result failed: %w", err)
	}

	return nil
}

func (e *OKRExtractor) getCycleList() ([]any, error) {
	output, err := e.cli.RunCommand("okr", "+cycle-list")
	if err != nil {
		return nil, err
	}

	var result []any
	if err := json.Unmarshal(output, &result); err != nil {
		var single any
		if err := json.Unmarshal(output, &single); err != nil {
			return nil, err
		}
		result = []any{single}
	}
	return result, nil
}

func (e *OKRExtractor) parseNewCycles(cycles []any, lastCheck time.Time) []Change {
	var changes []Change

	for _, cycle := range cycles {
		cycleMap, ok := cycle.(map[string]any)
		if !ok {
			continue
		}
		data, ok := cycleMap["data"].(map[string]any)
		if !ok {
			continue
		}
		items, ok := data["items"].([]any)
		if !ok {
			continue
		}
		for _, item := range items {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}

			cycleID, _ := itemMap["cycle_id"].(string)
			cycleName, _ := itemMap["name"].(string)

			// 没有时间戳字段，首次检测不报告
			if lastCheck.IsZero() {
				continue
			}

			changes = append(changes, Change{
				Type:       "new",
				EntityType: "okr_cycle",
				EntityID:   cycleID,
				Summary:    fmt.Sprintf("OKR周期变化: %s", cycleName),
			})
		}
	}

	return changes
}
