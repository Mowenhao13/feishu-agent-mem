// internal/llm/tools/parse_tool.go

package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseTool JSON 解析工具（容错处理）
type ParseTool struct{}

func NewParseTool() *ParseTool {
	return &ParseTool{}
}

// ParseJSON 解析 JSON（容错）
func (t *ParseTool) ParseJSON(input string) (map[string]any, error) {
	// 1. 尝试直接解析
	var result map[string]any
	if err := json.Unmarshal([]byte(input), &result); err == nil {
		return result, nil
	}

	// 2. 尝试提取 JSON 块
	jsonBlock := t.extractJSONBlock(input)
	if jsonBlock != "" {
		if err := json.Unmarshal([]byte(jsonBlock), &result); err == nil {
			return result, nil
		}
	}

	// 3. 尝试修复常见格式问题
	fixed := t.fixCommonIssues(input)
	if err := json.Unmarshal([]byte(fixed), &result); err == nil {
		return result, nil
	}

	return nil, fmt.Errorf("JSON parse failed: %s", input)
}

// extractJSONBlock 提取 JSON 块
func (t *ParseTool) extractJSONBlock(input string) string {
	// 从 Markdown 代码块中提取 JSON
	if idx := strings.Index(input, "```json"); idx != -1 {
		start := idx + 7
		if end := strings.Index(input[start:], "```"); end != -1 {
			return strings.TrimSpace(input[start : start+end])
		}
	}
	if idx := strings.Index(input, "```"); idx != -1 {
		start := idx + 3
		if end := strings.Index(input[start:], "```"); end != -1 {
			return strings.TrimSpace(input[start : start+end])
		}
	}
	// 尝试找到第一个 { 到最后一个 }
	if start := strings.Index(input, "{"); start != -1 {
		if end := strings.LastIndex(input, "}"); end != -1 && end > start {
			return input[start : end+1]
		}
	}
	return ""
}

// fixCommonIssues 修复常见格式问题
func (t *ParseTool) fixCommonIssues(input string) string {
	// 简化实现，实际需要更复杂的逻辑
	return input
}
