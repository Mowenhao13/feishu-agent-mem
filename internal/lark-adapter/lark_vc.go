package larkadapter

import (
	"encoding/json"
	"fmt"
	"time"
)

// VCExtractor 视频会议提取器
type VCExtractor struct {
	config *Config
	cli    *LarkCLI
}

// NewVCExtractor 创建视频会议提取器
func NewVCExtractor(cfg *Config) *VCExtractor {
	return &VCExtractor{
		config: cfg,
		cli:    NewLarkCLI(),
	}
}

// Name 实现 Extractor 接口
func (e *VCExtractor) Name() string {
	return "lark_vc"
}

// Detect 检测新结束的会议
func (e *VCExtractor) Detect(lastCheck time.Time) (*DetectResult, error) {
	changes := []Change{}

	// 搜索自 lastCheck 以来结束的会议
	startTs := lastCheck.Unix()
	endTs := time.Now().Unix()

	if !lastCheck.IsZero() {
		output, err := e.cli.RunCommand(
			"vc", "+search",
			"--start-time", fmt.Sprintf("%d", startTs),
			"--end-time", fmt.Sprintf("%d", endTs),
		)
		if err != nil {
			// +search 不支持时间范围则获取全部再过滤
			output, err = e.cli.RunCommand("vc", "+search")
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
		}

		meetings := e.parseEndedMeetings(output)
		for _, m := range meetings {
			changes = append(changes, Change{
				Type:       "new",
				EntityType: "meeting",
				EntityID:   m["meeting_id"].(string),
				Summary:    fmt.Sprintf("新结束会议: %s", m["topic"].(string)),
			})
		}
	}

	// 检测会议纪要（note_doc）
	// 如果有新结束的会议，标记为需提取纪要

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

// Extract 提取会议决策信息
func (e *VCExtractor) Extract() error {
	rawData := make(map[string]any)
	errors := make(map[string]string)

	if meetings, err := e.searchMeetings(); err == nil {
		rawData["meetings"] = meetings
	} else {
		errors["meetings"] = err.Error()
	}

	if len(errors) > 0 {
		rawData["_errors"] = errors
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

func (e *VCExtractor) searchMeetings() ([]any, error) {
	output, err := e.cli.RunCommand("vc", "+search", "--page-all")
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

func (e *VCExtractor) parseEndedMeetings(output []byte) []map[string]any {
	var meetings []map[string]any

	var result []any
	if err := json.Unmarshal(output, &result); err != nil {
		var single any
		if err := json.Unmarshal(output, &single); err != nil {
			return meetings
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
			meetingID, _ := mMap["meeting_id"].(string)
			topic, _ := mMap["topic"].(string)
			if meetingID == "" {
				continue
			}
			meetings = append(meetings, map[string]any{
				"meeting_id": meetingID,
				"topic":      topic,
			})
		}
	}

	return meetings
}
