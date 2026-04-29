package larkadapter

import (
	"encoding/json"
	"fmt"
	"time"
)

// TaskExtractor 任务提取器
type TaskExtractor struct {
	config *Config
	cli    *LarkCLI
}

// NewTaskExtractor 创建任务提取器
func NewTaskExtractor(cfg *Config) *TaskExtractor {
	return &TaskExtractor{
		config: cfg,
		cli:    NewLarkCLI(),
	}
}

// Name 实现 Extractor 接口
func (e *TaskExtractor) Name() string {
	return "lark_task"
}

// Detect 检测任务变化（新增、完成、状态变化）
func (e *TaskExtractor) Detect(lastCheck time.Time) (*DetectResult, error) {
	changes := []Change{}

	// 1. 检测新任务
	myTasks, err := e.getMyTasks()
	if err == nil {
		newTasks := e.parseTaskChanges(myTasks, lastCheck)
		changes = append(changes, newTasks...)
	}

	// 2. 检测相关任务变化
	relatedTasks, err := e.getRelatedTasks()
	if err == nil {
		relatedChanges := e.parseTaskChanges(relatedTasks, lastCheck)
		// 合并去重（根据 entity_id）
		seen := make(map[string]bool)
		for _, c := range changes {
			seen[c.EntityID] = true
		}
		for _, c := range relatedChanges {
			if !seen[c.EntityID] {
				changes = append(changes, c)
			}
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

// Extract 提取任务决策信息
func (e *TaskExtractor) Extract() error {
	rawData := make(map[string]any)
	errors := make(map[string]string)

	if myTasks, err := e.getMyTasks(); err == nil {
		rawData["my_tasks"] = myTasks
	} else {
		errors["my_tasks"] = err.Error()
	}

	if relatedTasks, err := e.getRelatedTasks(); err == nil {
		rawData["related_tasks"] = relatedTasks
	} else {
		errors["related_tasks"] = err.Error()
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

func (e *TaskExtractor) getMyTasks() ([]any, error) {
	output, err := e.cli.RunCommand("task", "+get-my-tasks")
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

func (e *TaskExtractor) getRelatedTasks() ([]any, error) {
	output, err := e.cli.RunCommand("task", "+get-related-tasks")
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

func (e *TaskExtractor) parseTaskChanges(tasks []any, lastCheck time.Time) []Change {
	var changes []Change
	cutoff := lastCheck.Unix()

	for _, task := range tasks {
		taskMap, ok := task.(map[string]any)
		if !ok {
			continue
		}
		data, ok := taskMap["data"].(map[string]any)
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

			guid, _ := itemMap["guid"].(string)
			summary, _ := itemMap["summary"].(string)
			if guid == "" {
				continue
			}

			// 检查状态变化
			status, _ := itemMap["status"].(string)

			// 检查创建时间
			createdAt, _ := itemMap["created_at"].(string)
			createdTs := parseMessageTime(createdAt)

			if !lastCheck.IsZero() && createdTs > cutoff {
				changes = append(changes, Change{
					Type:       "new",
					EntityType: "task",
					EntityID:   guid,
					Summary:    fmt.Sprintf("新任务: %s", summary),
					Timestamp:  createdTs,
				})
			} else if status == "done" {
				// 检查完成时间
				completedAt, _ := itemMap["completed_at"].(string)
				completedTs := parseMessageTime(completedAt)
				if !lastCheck.IsZero() && completedTs > cutoff && completedTs > 0 {
					changes = append(changes, Change{
						Type:       "updated",
						EntityType: "task",
						EntityID:   guid,
						Summary:    fmt.Sprintf("任务已完成: %s", summary),
						Timestamp:  completedTs,
					})
				}
			}
		}
	}

	return changes
}
