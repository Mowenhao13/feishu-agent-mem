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

// Detect 检测文档变化（新增/更新/评论/权限变更等）
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

		docChanges := e.analyzeDocChanges(docMap, cutoff, lastCheck.IsZero())
		changes = append(changes, docChanges...)
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

// analyzeDocChanges 分析文档的详细变化类型
func (e *DocExtractor) analyzeDocChanges(doc map[string]any, cutoff int64, isFirstCheck bool) []Change {
	var changes []Change

	// 如果是第一次检测，跳过初始扫描
	if isFirstCheck {
		return changes
	}

	docToken, _ := doc["doc_token"].(string)
	title, _ := doc["title"].(string)
	docType, _ := doc["obj_type"].(string)

	if docToken == "" && title == "" {
		return changes
	}
	if docToken == "" {
		docToken = title
	}

	// 获取文档时间戳
	docTime := e.getDocTimestamp(doc)
	if docTime == 0 {
		return changes
	}

	// 确定文档类型标签
	typeLabel := e.getDocTypeLabel(docType)

	// 根据不同的时间字段判断变化类型
	changeTypes := e.detectChangeTypes(doc, cutoff)

	for _, ct := range changeTypes {
		var changeType, summary string
		switch ct {
		case "created":
			changeType = "doc_created"
			summary = fmt.Sprintf("新建%s: %s", typeLabel, title)
		case "content_updated":
			changeType = "doc_content_updated"
			summary = fmt.Sprintf("%s内容更新: %s", typeLabel, title)
		case "comment_added":
			changeType = "doc_comment_added"
			summary = fmt.Sprintf("%s新评论: %s", typeLabel, title)
		case "permission_changed":
			changeType = "doc_permission_changed"
			summary = fmt.Sprintf("%s权限变更: %s", typeLabel, title)
		default:
			changeType = "doc_updated"
			summary = fmt.Sprintf("%s更新: %s", typeLabel, title)
		}

		changes = append(changes, Change{
			Type:       changeType,
			EntityType: "doc",
			EntityID:   docToken,
			Summary:    summary,
			Timestamp:  docTime,
		})
	}

	return changes
}

// getDocTypeLabel 获取文档类型的可读标签
func (e *DocExtractor) getDocTypeLabel(objType string) string {
	switch objType {
	case "doc":
		return "文档"
	case "sheet":
		return "表格"
	case "bitable":
		return "多维表格"
	case "mindnote":
		return "思维导图"
	case "file":
		return "文件"
	case "docx":
		return "Word文档"
	case "pptx":
		return "PPT"
	case "pdf":
		return "PDF"
	default:
		return "文档"
	}
}

// getDocTimestamp 获取文档的时间戳
func (e *DocExtractor) getDocTimestamp(doc map[string]any) int64 {
	// 尝试不同的时间字段
	for _, key := range []string{"updated_at", "update_time", "modified_at", "created_at", "create_time"} {
		if v, ok := doc[key]; ok {
			switch val := v.(type) {
			case float64:
				return int64(val)
			case int64:
				return val
			case string:
				t, err := time.Parse(time.RFC3339, val)
				if err == nil {
					return t.Unix()
				}
			}
		}
	}
	return 0
}

// detectChangeTypes 检测文档的变化类型
func (e *DocExtractor) detectChangeTypes(doc map[string]any, cutoff int64) []string {
	var types []string

	// 检查不同的时间戳来判断变化类型
	for _, key := range []string{"updated_at", "update_time", "modified_at"} {
		if v, ok := doc[key]; ok {
			var ts int64
			switch val := v.(type) {
			case float64:
				ts = int64(val)
			case int64:
				ts = val
			case string:
				t, err := time.Parse(time.RFC3339, val)
				if err == nil {
					ts = t.Unix()
				}
			}
			if ts > cutoff {
				types = append(types, "content_updated")
				break
			}
		}
	}

	// 检查创建时间
	if len(types) == 0 {
		for _, key := range []string{"created_at", "create_time"} {
			if v, ok := doc[key]; ok {
				var ts int64
				switch val := v.(type) {
				case float64:
					ts = int64(val)
				case int64:
					ts = val
				case string:
					t, err := time.Parse(time.RFC3339, val)
					if err == nil {
						ts = t.Unix()
					}
				}
				if ts > cutoff {
					types = append(types, "created")
					break
				}
			}
		}
	}

	// 如果没有检测到具体变化，使用默认
	if len(types) == 0 {
		types = append(types, "default")
	}

	return types
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
