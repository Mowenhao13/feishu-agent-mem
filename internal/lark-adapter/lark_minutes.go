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

// Detect 检测妙记变化（新增、内容更新、AI 摘要等）
func (e *MinutesExtractor) Detect(lastCheck time.Time) (*DetectResult, error) {
	changes := []Change{}
	cutoff := lastCheck.Unix()

	if !lastCheck.IsZero() {
		output, err := e.cli.RunCommand("minutes", "+search", "--query", "")
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

		// 分析妙记的详细变化
		minutesChanges := e.parseMinutesWithDetails(output, cutoff)
		changes = append(changes, minutesChanges...)
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

// parseMinutesWithDetails 解析妙记并识别各种变化类型
func (e *MinutesExtractor) parseMinutesWithDetails(output []byte, cutoff int64) []Change {
	var changes []Change

	var result []any
	if err := json.Unmarshal(output, &result); err != nil {
		var single any
		if err := json.Unmarshal(output, &single); err != nil {
			return changes
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

			// 获取时间戳
			var minuteTs int64
			if createTime, ok := mMap["create_time"].(string); ok {
				minuteTs = parseMessageTime(createTime)
			} else if updateTime, ok := mMap["update_time"].(string); ok {
				minuteTs = parseMessageTime(updateTime)
			}

			if minuteTs <= cutoff {
				continue
			}

			// 检查是否是新创建的
			if createTime, ok := mMap["create_time"].(string); ok {
				createTs := parseMessageTime(createTime)
				if createTs > cutoff {
					changes = append(changes, Change{
						Type:       "minutes_created",
						EntityType: "minutes",
						EntityID:   token,
						Summary:    fmt.Sprintf("新妙记生成: %s", title),
						Timestamp:  createTs,
					})
				}
			}

			// 检查内容更新
			if updateTime, ok := mMap["update_time"].(string); ok {
				updateTs := parseMessageTime(updateTime)
				if createTime, ok := mMap["create_time"].(string); ok {
					createTs := parseMessageTime(createTime)
					if updateTs > cutoff && updateTs > createTs {
						changes = append(changes, Change{
							Type:       "minutes_updated",
							EntityType: "minutes",
							EntityID:   token,
							Summary:    fmt.Sprintf("妙记内容更新: %s", title),
							Timestamp:  updateTs,
						})
					}
				}
			}

			// 检查 AI 摘要状态
			if aiSummary, ok := mMap["ai_summary"].(map[string]any); ok && aiSummary != nil {
				changes = append(changes, Change{
					Type:       "minutes_ai_summary_ready",
					EntityType: "minutes",
					EntityID:   token,
					Summary:    fmt.Sprintf("妙记 AI 摘要就绪: %s", title),
					Timestamp:  minuteTs,
				})
			}

			// 检查是否有录制内容
			if hasRecording, ok := mMap["has_recording"].(bool); ok && hasRecording {
				changes = append(changes, Change{
					Type:       "minutes_recording_available",
					EntityType: "minutes",
					EntityID:   token,
					Summary:    fmt.Sprintf("妙记有录制内容: %s", title),
					Timestamp:  minuteTs,
				})
			}
		}
	}

	return changes
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
