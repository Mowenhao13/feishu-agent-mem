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

// Detect 检测日程变化（新增/更新的日程事件）
func (e *CalendarExtractor) Detect(lastCheck time.Time) (*DetectResult, error) {
	changes := []Change{}

	// 检测新日程：从今天起往后3天的日程
	today := time.Now()
	endDate := today.AddDate(0, 0, 3)

	agenda, err := e.getAgendaRange(today, endDate)
	if err == nil {
		events := e.parseAgendaEvents(agenda)
		cutoff := lastCheck.Unix()

		for _, ev := range events {
			ts, _ := ev["start_time"].(int64)
			if !lastCheck.IsZero() && ts <= cutoff {
				continue
			}
			changes = append(changes, Change{
				Type:       "new",
				EntityType: "event",
				EntityID:   ev["event_id"].(string),
				Summary:    fmt.Sprintf("新日程: %s", ev["title"].(string)),
				Timestamp:  ts,
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
