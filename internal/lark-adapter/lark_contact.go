package larkadapter

import (
	"encoding/json"
	"fmt"
	"time"
)

// ContactExtractor 通讯录提取器
type ContactExtractor struct {
	config *Config
	cli    *LarkCLI
}

// NewContactExtractor 创建通讯录提取器
func NewContactExtractor(cfg *Config) *ContactExtractor {
	return &ContactExtractor{
		config: cfg,
		cli:    NewLarkCLI(),
	}
}

// Name 实现 Extractor 接口
func (e *ContactExtractor) Name() string {
	return "lark_contact"
}

// Detect 检测通讯录/人员变化
// 通讯录变化频率低，通过检测用户信息变更来判断
func (e *ContactExtractor) Detect(lastCheck time.Time) (*DetectResult, error) {
	changes := []Change{}

	// 通讯录变化检测需要事件订阅支持
	// 这里只做轻量检测：如果有新的用户搜索请求
	// 实际的事件驱动靠 lark-event WebSocket 监听

	// 如果是首次检测，不报告变化（通讯录基线数据大）
	if lastCheck.IsZero() {
		result := &DetectResult{
			Source:     e.Name(),
			HasChanges: false,
			DetectedAt: time.Now(),
			LastCheck:  lastCheck,
		}
		_ = SaveDetectResult(result)
		return result, nil
	}

	// 非首次：尝试搜索空查询，看能否获取到信息
	// 通讯录变化不在这里实时检测，建议使用事件订阅
	result := &DetectResult{
		Source:     e.Name(),
		HasChanges: false,
		DetectedAt: time.Now(),
		LastCheck:  lastCheck,
		Changes:    changes,
	}

	_ = SaveDetectResult(result)
	return result, nil
}

// Extract 提取联系人信息用于决策关联
func (e *ContactExtractor) Extract() error {
	rawData := make(map[string]any)

	if users, err := e.searchUsers(); err == nil {
		rawData["users"] = users
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

func (e *ContactExtractor) searchUsers() ([]any, error) {
	output, err := e.cli.RunCommand("contact", "+search-user", "--query", "")
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
