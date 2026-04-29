package adapters

import (
	"context"
	"fmt"
	"time"
)

// LarkCli lark-cli 命令封装层 — 所有飞书操作通过此层转发（openclaw-architecture.md §2.2）
type LarkCli struct {
	binPath  string
	identity string // user | bot
	executor *CliExecutor
}

// NewLarkCli 创建 LarkCli 适配器
func NewLarkCli(binPath, identity string) *LarkCli {
	return &LarkCli{
		binPath:  binPath,
		identity: identity,
		executor: NewCliExecutor(binPath, identity),
	}
}

// ---- 群聊提取（decision-extraction.md §4.1）----

// Chat 飞书群聊
type Chat struct {
	ChatID string `json:"chat_id"`
	Name   string `json:"name"`
}

// Message 群聊消息
type Message struct {
	MessageID string    `json:"message_id"`
	ChatID    string    `json:"chat_id"`
	Sender    Sender    `json:"sender"`
	Body      string    `json:"body"`
	SendTime  time.Time `json:"send_time"`
	ThreadID  string    `json:"thread_id,omitempty"`
}

// Sender 消息发送者
type Sender struct {
	OpenID string `json:"open_id"`
	Name   string `json:"name,omitempty"`
}

// SearchChats 搜索群聊
func (c *LarkCli) SearchChats(ctx context.Context, query string) ([]Chat, error) {
	output, err := c.executor.Run(ctx, "im", "+chat-search", "--query", query)
	if err != nil {
		return nil, fmt.Errorf("search chats: %w", err)
	}
	var chats []Chat
	if err := parseJSON(output, &chats); err != nil {
		return nil, err
	}
	return chats, nil
}

// ListMessages 按时间范围列出群消息
func (c *LarkCli) ListMessages(ctx context.Context, chatID string, start, end int64) ([]Message, error) {
	output, err := c.executor.Run(ctx,
		"im", "+chat-messages-list",
		"--chat-id", chatID,
		"--start-time", fmt.Sprintf("%d", start),
		"--end-time", fmt.Sprintf("%d", end),
		"--page-all",
	)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	var messages []Message
	if err := parseJSON(output, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// SearchMessages 关键词搜索消息（跨群搜索，仅 user 身份）
func (c *LarkCli) SearchMessages(ctx context.Context, query string, chatIDs []string, start, end int64) ([]Message, error) {
	args := []string{"im", "+messages-search", "--query", query}
	for _, id := range chatIDs {
		args = append(args, "--chat-ids", id)
	}
	args = append(args,
		"--start-time", fmt.Sprintf("%d", start),
		"--end-time", fmt.Sprintf("%d", end),
		"--page-all",
	)
	output, err := c.executor.Run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("search messages: %w", err)
	}
	var messages []Message
	if err := parseJSON(output, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// GetPinMessages 获取 Pin 消息
func (c *LarkCli) GetPinMessages(ctx context.Context, chatID string) ([]Message, error) {
	output, err := c.executor.Run(ctx, "im", "pins", "list", "--params", fmt.Sprintf(`{"chat_id": "%s"}`, chatID))
	if err != nil {
		return nil, fmt.Errorf("get pin messages: %w", err)
	}
	var messages []Message
	if err := parseJSON(output, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// ---- 会议提取（decision-extraction.md §4.5）----

// Meeting 会议
type Meeting struct {
	MeetingID    string    `json:"meeting_id"`
	Topic        string    `json:"topic"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Participants []string  `json:"participants"`
}

// MeetingNotes 会议纪要产物
type MeetingNotes struct {
	MeetingID     string   `json:"meeting_id"`
	NoteDocToken  string   `json:"note_doc_token"`
	VerbatimToken string   `json:"verbatim_doc_token"`
	Summary       string   `json:"summary"`
	ActionItems   []string `json:"action_items"`
}

// SearchMeetings 搜索已结束会议
func (c *LarkCli) SearchMeetings(ctx context.Context, start, end int64, query string) ([]Meeting, error) {
	output, err := c.executor.Run(ctx,
		"vc", "+search",
		"--start-time", fmt.Sprintf("%d", start),
		"--end-time", fmt.Sprintf("%d", end),
		"--query", query,
		"--page-all",
	)
	if err != nil {
		return nil, fmt.Errorf("search meetings: %w", err)
	}
	var meetings []Meeting
	if err := parseJSON(output, &meetings); err != nil {
		return nil, err
	}
	return meetings, nil
}

// GetMeetingNotes 获取会议纪要产物
func (c *LarkCli) GetMeetingNotes(ctx context.Context, meetingID string) (*MeetingNotes, error) {
	output, err := c.executor.Run(ctx, "vc", "+notes", "--meeting-ids", meetingID)
	if err != nil {
		return nil, fmt.Errorf("get meeting notes: %w", err)
	}
	var notes MeetingNotes
	if err := parseJSON(output, &notes); err != nil {
		return nil, err
	}
	return &notes, nil
}

// ---- 文档提取（decision-extraction.md §4.3）----

// DocRef 文档引用
type DocRef struct {
	DocToken string `json:"doc_token"`
	Title    string `json:"title"`
	DocType  string `json:"doc_type"`
	URL      string `json:"url"`
}

// Comment 文档评论
type Comment struct {
	CommentID string `json:"comment_id"`
	Content   string `json:"content"`
	User      string `json:"user"`
	IsSolved  bool   `json:"is_solved"`
}

// SearchDocs 搜索云空间文档
func (c *LarkCli) SearchDocs(ctx context.Context, query string) ([]DocRef, error) {
	output, err := c.executor.Run(ctx, "docs", "+search", "--query", query, "--page-all")
	if err != nil {
		return nil, fmt.Errorf("search docs: %w", err)
	}
	var docs []DocRef
	if err := parseJSON(output, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

// FetchDoc 读取文档内容
func (c *LarkCli) FetchDoc(ctx context.Context, docToken, apiVersion, format string) (string, error) {
	output, err := c.executor.Run(ctx,
		"docs", "+fetch",
		"--api-version", apiVersion,
		"--doc", docToken,
		"--doc-format", format,
	)
	if err != nil {
		return "", fmt.Errorf("fetch doc: %w", err)
	}
	return string(output), nil
}

// GetDocComments 获取文档评论
func (c *LarkCli) GetDocComments(ctx context.Context, docToken, docType string) ([]Comment, error) {
	output, err := c.executor.Run(ctx,
		"drive", "file.comments", "list",
		"--params", fmt.Sprintf(`{"file_token": "%s", "file_type": "%s"}`, docToken, docType),
	)
	if err != nil {
		return nil, fmt.Errorf("get doc comments: %w", err)
	}
	var comments []Comment
	if err := parseJSON(output, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// ---- 日程提取（decision-extraction.md §4.2）----

// CalendarEvent 日程
type CalendarEvent struct {
	EventID   string   `json:"event_id"`
	Title     string   `json:"title"`
	StartTime int64    `json:"start_time"`
	EndTime   int64    `json:"end_time"`
	Attendees []string `json:"attendees"`
}

// GetAgenda 查看日程
func (c *LarkCli) GetAgenda(ctx context.Context, date string) ([]CalendarEvent, error) {
	output, err := c.executor.Run(ctx, "calendar", "+agenda", "--date", date)
	if err != nil {
		return nil, fmt.Errorf("get agenda: %w", err)
	}
	var events []CalendarEvent
	if err := parseJSON(output, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// SearchEvents 搜索日程
func (c *LarkCli) SearchEvents(ctx context.Context, query string, start, end int64) ([]CalendarEvent, error) {
	output, err := c.executor.Run(ctx,
		"calendar", "events", "search",
		"--params", fmt.Sprintf(`{"query": "%s", "start_time": "%d", "end_time": "%d"}`, query, start, end),
	)
	if err != nil {
		return nil, fmt.Errorf("search events: %w", err)
	}
	var events []CalendarEvent
	if err := parseJSON(output, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// ---- 任务提取（decision-extraction.md §4.6）----

// Task 任务
type Task struct {
	GUID     string `json:"guid"`
	Title    string `json:"title"`
	Status   string `json:"status"` // completed | pending
	Assignee string `json:"assignee"`
}

// GetMyTasks 获取分配给我的任务
func (c *LarkCli) GetMyTasks(ctx context.Context) ([]Task, error) {
	output, err := c.executor.Run(ctx, "task", "+get-my-tasks", "--page-size", "50")
	if err != nil {
		return nil, fmt.Errorf("get my tasks: %w", err)
	}
	var tasks []Task
	if err := parseJSON(output, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// SearchTasks 搜索任务
func (c *LarkCli) SearchTasks(ctx context.Context, query string) ([]Task, error) {
	output, err := c.executor.Run(ctx, "task", "+search", "--query", query, "--page-size", "50")
	if err != nil {
		return nil, fmt.Errorf("search tasks: %w", err)
	}
	var tasks []Task
	if err := parseJSON(output, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// ---- OKR 提取（decision-extraction.md §4.7）----

// OkrCycle OKR 周期
type OkrCycle struct {
	CycleID string `json:"cycle_id"`
	Period  string `json:"period"`
}

// OkrDetail OKR 详情
type OkrDetail struct {
	CycleID    string         `json:"cycle_id"`
	Objectives []OkrObjective `json:"objectives"`
}

// OkrObjective OKR 目标
type OkrObjective struct {
	ObjectiveID string      `json:"objective_id"`
	Title       string      `json:"title"`
	KeyResults  []KeyResult `json:"key_results"`
}

// KeyResult 关键结果
type KeyResult struct {
	KRID     string  `json:"kr_id"`
	Title    string  `json:"title"`
	Progress float64 `json:"progress"`
}

// GetOkrCycle 获取 OKR 周期列表
func (c *LarkCli) GetOkrCycle(ctx context.Context, userID string) ([]OkrCycle, error) {
	output, err := c.executor.Run(ctx, "okr", "+cycle-list", "--user-id", userID)
	if err != nil {
		return nil, fmt.Errorf("get okr cycle: %w", err)
	}
	var cycles []OkrCycle
	if err := parseJSON(output, &cycles); err != nil {
		return nil, err
	}
	return cycles, nil
}

// GetOkrDetail 获取周期内 OKR 详情
func (c *LarkCli) GetOkrDetail(ctx context.Context, cycleID string) (*OkrDetail, error) {
	output, err := c.executor.Run(ctx, "okr", "+cycle-detail", "--cycle-id", cycleID)
	if err != nil {
		return nil, fmt.Errorf("get okr detail: %w", err)
	}
	var detail OkrDetail
	if err := parseJSON(output, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

// ---- 通讯录（decision-extraction.md §4.8）----

// User 用户信息
type User struct {
	OpenID     string `json:"open_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Department string `json:"department"`
}

// SearchUser 搜索用户
func (c *LarkCli) SearchUser(ctx context.Context, query string) (*User, error) {
	output, err := c.executor.Run(ctx, "contact", "+search-user", "--query", query)
	if err != nil {
		return nil, fmt.Errorf("search user: %w", err)
	}
	var user User
	if err := parseJSON(output, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUser 获取用户详情
func (c *LarkCli) GetUser(ctx context.Context, userID string) (*User, error) {
	output, err := c.executor.Run(ctx, "contact", "+get-user", "--user-id", userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	var user User
	if err := parseJSON(output, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ---- 辅助方法 ----

// parseJSON 解析 JSON 输出（占位，实际使用 encoding/json）
func parseJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	// 实际实现使用 encoding/json.Unmarshal
	return nil
}
