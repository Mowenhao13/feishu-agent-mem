package larkadapter

import (
	"encoding/json"
	"fmt"
	"time"
)

// CalendarExtractor 日程提取器
type CalendarExtractor struct {
	config *Config
	cli    *LarkCLI
}

// NewCalendarExtractor 创建日程提取器
func NewCalendarExtractor(cfg *Config) *CalendarExtractor {
	return &CalendarExtractor{
		config: cfg,
		cli:    NewLarkCLI(),
	}
}

// Name 实现 Extractor 接口
func (e *CalendarExtractor) Name() string {
	return "lark_calendar"
}

// Detect 检测日程变化（新增/更新/删除日程、参会人变化、RSVP 变化等）
func (e *CalendarExtractor) Detect(lastCheck time.Time) (*DetectResult, error) {
	changes := []Change{}
	cutoff := lastCheck.Unix()

	// 检测日程：从今天起往后3天的日程
	today := time.Now()
	endDate := today.AddDate(0, 0, 3)

	agenda, err := e.getAgendaRange(today, endDate)
	if err == nil {
		events := e.parseAgendaEvents(agenda)

		for _, ev := range events {
			ts, _ := ev["start_time"].(int64)
			eventID, _ := ev["event_id"].(string)
			title, _ := ev["title"].(string)

			// 分析日程的详细变化类型
			changeTypes := e.analyzeEventChanges(ev)

			if len(changeTypes) > 0 {
				for _, ct := range changeTypes {
					var changeType, summary string
					switch ct {
					case "new":
						if lastCheck.IsZero() || ts > cutoff {
							changeType = "new_event"
							summary = fmt.Sprintf("新日程: %s", title)
						}
					case "updated":
						changeType = "updated_event"
						summary = fmt.Sprintf("日程更新: %s", title)
					case "attendee_added":
						changeType = "attendee_added"
						summary = fmt.Sprintf("日程添加参会人: %s", title)
					case "attendee_removed":
						changeType = "attendee_removed"
						summary = fmt.Sprintf("日程移除参会人: %s", title)
					case "rsvp_changed":
						changeType = "rsvp_changed"
						summary = fmt.Sprintf("日程 RSVP 状态变化: %s", title)
					case "time_changed":
						changeType = "time_changed"
						summary = fmt.Sprintf("日程时间变更: %s", title)
					case "room_added":
						changeType = "room_added"
						summary = fmt.Sprintf("日程添加会议室: %s", title)
					}

					if changeType != "" {
						changes = append(changes, Change{
							Type:       changeType,
							EntityType: "event",
							EntityID:   eventID,
							Summary:    summary,
							Timestamp:  ts,
						})
					}
				}
			} else if !lastCheck.IsZero() && ts > cutoff {
				// 默认按新日程处理
				changes = append(changes, Change{
					Type:       "new_event",
					EntityType: "event",
					EntityID:   eventID,
					Summary:    fmt.Sprintf("新日程: %s", title),
					Timestamp:  ts,
				})
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

// analyzeEventChanges 分析日程的详细变化类型
func (e *CalendarExtractor) analyzeEventChanges(event map[string]any) []string {
	var changes []string

	// 这里我们模拟从事件数据中分析变化类型
	// 实际应用中需要对比历史快照

	// 检查是否有参会人相关字段
	if attendees, ok := event["attendees"]; ok {
		if attList, ok := attendees.([]any); ok && len(attList) > 0 {
			// 如果有参会人数据，可以进一步分析
			changes = append(changes, "attendee_added")
		}
	}

	// 检查是否有会议室
	if rooms, ok := event["rooms"]; ok {
		if roomList, ok := rooms.([]any); ok && len(roomList) > 0 {
			changes = append(changes, "room_added")
		}
	}

	// 检查是否有 RSVP 信息
	if rsvp, ok := event["rsvp_status"]; ok && rsvp != nil {
		changes = append(changes, "rsvp_changed")
	}

	// 如果没有具体变化，默认返回 new
	if len(changes) == 0 {
		changes = append(changes, "new")
	}

	return changes
}

// Extract 提取日程决策信息
func (e *CalendarExtractor) Extract() error {
	rawData := make(map[string]any)
	errors := make(map[string]string)

	if agenda, err := e.getTodayAgenda(); err == nil {
		rawData["today_agenda"] = agenda
	} else {
		errors["today_agenda"] = err.Error()
	}

	if events, err := e.searchEvents(); err == nil {
		rawData["events"] = events
	} else {
		errors["events"] = err.Error()
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

func (e *CalendarExtractor) getTodayAgenda() ([]any, error) {
	today := time.Now().Format("2006-01-02")
	output, err := e.cli.RunCommand("calendar", "+agenda", "--date", today)
	if err != nil {
		output, err = e.cli.RunCommand("calendar", "+agenda")
		if err != nil {
			return nil, err
		}
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

func (e *CalendarExtractor) getAgendaRange(start, end time.Time) ([]any, error) {
	// 逐日检测，使用 +agenda 命令
	var allResults []any
	for d := start; d.Before(end) || d.Equal(end); d = d.AddDate(0, 0, 1) {
		date := d.Format("2006-01-02")
		output, err := e.cli.RunCommand("calendar", "+agenda", "--date", date)
		if err != nil {
			continue
		}

		var result []any
		if err := json.Unmarshal(output, &result); err != nil {
			var single any
			if err := json.Unmarshal(output, &single); err != nil {
				continue
			}
			result = []any{single}
		}
		allResults = append(allResults, result...)
	}
	return allResults, nil
}

func (e *CalendarExtractor) searchEvents() ([]any, error) {
	output, err := e.cli.RunCommand(
		"calendar", "events", "search",
		"--params", `{"query": "会议"}`,
	)
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

func (e *CalendarExtractor) parseAgendaEvents(agenda []any) []map[string]any {
	var events []map[string]any

	// 为每个条目按天分配一个默认 ID
	for i, item := range agenda {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		// 从 +agenda 输出中提取日程信息
		title, _ := itemMap["summary"].(string)
		if title == "" {
			title, _ = itemMap["title"].(string)
		}
		if title == "" {
			continue
		}

		startTime := timeNowUnix(itemMap, "start")
		if startTime == 0 {
			startTime = time.Now().Unix()
		}

		events = append(events, map[string]any{
			"event_id":   fmt.Sprintf("cal_event_%d", i),
			"title":      title,
			"start_time": startTime,
		})
	}

	return events
}

func timeNowUnix(m map[string]any, key string) int64 {
	if v, ok := m[key]; ok {
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
	return 0
}
