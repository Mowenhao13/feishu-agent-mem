package larkadapter

import (
	"encoding/json"
	"fmt"
	"time"
)

// MinutesExtractor 妙记提取器
type MinutesExtractor struct {
	config *Config
	cli    *LarkCLI
}

// NewMinutesExtractor 创建妙记提取器
func NewMinutesExtractor(cfg *Config) *MinutesExtractor {
	return &MinutesExtractor{
		config: cfg,
		cli:    NewLarkCLI(),
	}
}

// Name 实现 Extractor 接口
func (e *MinutesExtractor) Name() string {
	return "lark_minutes"
}

// Detect 检测新增的妙记
func (e *MinutesExtractor) Detect(lastCheck time.Time) (*DetectResult, error) {
	changes := []Change{}

	// 按时间范围搜索新妙记
	startTs := lastCheck.Unix()
	endTs := time.Now().Unix()

	if !lastCheck.IsZero() {
		output, err := e.cli.RunCommand(
			"minutes", "+search",
			"--query", "",
			"--start-time", fmt.Sprintf("%d", startTs),
			"--end-time", fmt.Sprintf("%d", endTs),
		)
		if err != nil {
			result := &DetectResult{
				Source:     e.Name(),
				HasChanges: false,
				DetectedAt: time.Now(),
				LastCheck:  lastCheck,
			}
			_ = SaveDetectResult(result)
			return result, nil
		}

		minutes := e.parseNewMinutes(output)
		for _, m := range minutes {
			changes = append(changes, Change{
				Type:       "new",
				EntityType: "minutes",
				EntityID:   m["minute_token"].(string),
				Summary:    fmt.Sprintf("新妙记: %s", m["title"].(string)),
			})
		}
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

// Extract 提取妙记决策信息
func (e *MinutesExtractor) Extract() error {
	rawData := make(map[string]any)

	if minutes, err := e.searchMinutes(); err == nil {
		rawData["minutes"] = minutes
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

func (e *MinutesExtractor) searchMinutes() ([]any, error) {
	output, err := e.cli.RunCommand("minutes", "+search", "--query", "")
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

func (e *MinutesExtractor) parseNewMinutes(output []byte) []map[string]any {
	var minutes []map[string]any

	var result []any
	if err := json.Unmarshal(output, &result); err != nil {
		var single any
		if err := json.Unmarshal(output, &single); err != nil {
			return minutes
		}
		result = []any{single}
	}

	for _, item := range result {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		data, ok := itemMap["data"].(map[string]any)
		if !ok {
			continue
		}
		items, ok := data["items"].([]any)
		if !ok {
			continue
		}
		for _, m := range items {
			mMap, ok := m.(map[string]any)
			if !ok {
				continue
			}
			token, _ := mMap["minute_token"].(string)
			title, _ := mMap["title"].(string)
			if token == "" {
				continue
			}
			minutes = append(minutes, map[string]any{
				"minute_token": token,
				"title":        title,
			})
		}
	}

	return minutes
}
