package larkadapter

import (
	"encoding/json"
	"fmt"
	"time"
)

// DocExtractor 云文档提取器
type DocExtractor struct {
	config *Config
	cli    *LarkCLI
}

// NewDocExtractor 创建云文档提取器
func NewDocExtractor(cfg *Config) *DocExtractor {
	return &DocExtractor{
		config: cfg,
		cli:    NewLarkCLI(),
	}
}

// Name 实现 Extractor 接口
func (e *DocExtractor) Name() string {
	return "lark_doc"
}

// Detect 检测文档变化（新增/更新的文档）
func (e *DocExtractor) Detect(lastCheck time.Time) (*DetectResult, error) {
	changes := []Change{}

	// 尝试获取文档列表
	output, err := e.cli.RunCommand("docs", "+search", "--query", "")
	if err != nil {
		// 如果失败返回无变化
		result := &DetectResult{
			Source:     e.Name(),
			HasChanges: false,
			DetectedAt: time.Now(),
			LastCheck:  lastCheck,
		}
		_ = SaveDetectResult(result)
		return result, nil
	}

	var docsList []any
	if err := json.Unmarshal(output, &docsList); err != nil {
		var single any
		if err := json.Unmarshal(output, &single); err != nil {
			result := &DetectResult{
				Source:     e.Name(),
				HasChanges: false,
				DetectedAt: time.Now(),
				LastCheck:  lastCheck,
			}
			_ = SaveDetectResult(result)
			return result, nil
		}
		docsList = []any{single}
	}

	cutoff := lastCheck.Unix()
	for _, doc := range docsList {
		docMap, ok := doc.(map[string]any)
		if !ok {
			continue
		}
		if e.isDocChanged(docMap, cutoff, lastCheck.IsZero()) {
			docToken, _ := docMap["doc_token"].(string)
			title, _ := docMap["title"].(string)
			if docToken == "" && title == "" {
				continue
			}
			if docToken == "" {
				docToken = title
			}
			changes = append(changes, Change{
				Type:       "new",
				EntityType: "doc",
				EntityID:   docToken,
				Summary:    fmt.Sprintf("文档更新: %s", title),
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

// Extract 提取云文档决策信息
func (e *DocExtractor) Extract() error {
	rawData := make(map[string]any)

	if docs, err := e.searchDocs(); err == nil {
		rawData["documents"] = docs
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

func (e *DocExtractor) searchDocs() ([]any, error) {
	output, err := e.cli.RunCommand("docs", "+search", "--query", "")
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

func (e *DocExtractor) isDocChanged(doc map[string]any, cutoff int64, isFirstCheck bool) bool {
	// 如果是第一次检测，跳过初始扫描（避免大量初始变化）
	if isFirstCheck {
		return false
	}

	// 检查更新时间
	for _, key := range []string{"updated_at", "update_time", "modified_at"} {
		if v, ok := doc[key]; ok {
			switch val := v.(type) {
			case float64:
				if int64(val) > cutoff {
					return true
				}
			case int64:
				if val > cutoff {
					return true
				}
			case string:
				t, err := time.Parse(time.RFC3339, val)
				if err == nil && t.Unix() > cutoff {
					return true
				}
			}
		}
	}
	return false
}
