package memory

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/openclaw/internal/core"
)

// MCPServer MCP Server — 向 OpenClaw 暴露渐进式披露工具（memory-openclaw-integration.md §4.1）
type MCPServer struct {
	port  int
	tools *MemoryTools
	index *IndexBuilder
	graph *core.MemoryGraph

	// 信号缓冲
	signalBuffer []SignalRecord
}

// SignalRecord 信号记录
type SignalRecord struct {
	SignalID   string    `json:"signal_id"`
	Adapter    string    `json:"adapter"`
	Strength   string    `json:"strength"`
	Summary    string    `json:"summary"`
	Timestamp  time.Time `json:"timestamp"`
	DecisionID string    `json:"decision_id,omitempty"`
}

// NewMCPServer 创建 MCP Server
func NewMCPServer(port int, mg *core.MemoryGraph) *MCPServer {
	return &MCPServer{
		port:         port,
		tools:        NewMemoryTools(mg),
		index:        NewIndexBuilder(mg),
		graph:        mg,
		signalBuffer: make([]SignalRecord, 0),
	}
}

// Start 启动 MCP Server
func (s *MCPServer) Start() error {
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("/health", s.handleHealth)

	// MCP 工具端点
	mux.HandleFunc("/mcp/search", s.handleSearch)
	mux.HandleFunc("/mcp/topic", s.handleTopic)
	mux.HandleFunc("/mcp/decision", s.handleDecision)
	mux.HandleFunc("/mcp/timeline", s.handleTimeline)
	mux.HandleFunc("/mcp/conflict", s.handleConflict)
	mux.HandleFunc("/mcp/signal", s.handleSignal)
	mux.HandleFunc("/mcp/index", s.handleIndex)

	// Webhook 接收
	mux.HandleFunc("/hooks/post-commit", s.handlePostCommit)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("MCP Server 启动于 %s", addr)
	return http.ListenAndServe(addr, mux)
}

// AddSignal 添加信号记录
func (s *MCPServer) AddSignal(record SignalRecord) {
	s.signalBuffer = append(s.signalBuffer, record)
	// 只保留最近 100 条
	if len(s.signalBuffer) > 100 {
		s.signalBuffer = s.signalBuffer[len(s.signalBuffer)-100:]
	}
}

// ---- 处理函数 ----

func (s *MCPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{
		"status":           "ok",
		"decisions_loaded": s.graph.Count(),
		"last_sync":        time.Now().Format(time.RFC3339),
	})
}

func (s *MCPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	idx := s.index.BuildIndex()
	jsonResponse(w, map[string]interface{}{
		"index":      idx.FormatIndex(),
		"token_cost": 800,
	})
}

func (s *MCPServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	var input SearchInput
	if err := parseBody(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := s.tools.Search(input)
	jsonResponse(w, result)
}

func (s *MCPServer) handleTopic(w http.ResponseWriter, r *http.Request) {
	var input TopicInput
	if err := parseBody(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := s.tools.Topic(input)
	jsonResponse(w, result)
}

func (s *MCPServer) handleDecision(w http.ResponseWriter, r *http.Request) {
	var input DecisionInput
	if err := parseBody(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := s.tools.Decision(input)
	if result == nil {
		jsonResponse(w, map[string]string{"error": "decision not found"})
		return
	}
	jsonResponse(w, result)
}

func (s *MCPServer) handleTimeline(w http.ResponseWriter, r *http.Request) {
	var input TimelineInput
	if err := parseBody(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := s.tools.Timeline(input)
	jsonResponse(w, result)
}

func (s *MCPServer) handleConflict(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]string{"message": "not implemented"})
}

func (s *MCPServer) handleSignal(w http.ResponseWriter, r *http.Request) {
	limit := 5
	if r.URL.Query().Get("limit") != "" {
		fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)
	}

	records := s.signalBuffer
	if len(records) > limit {
		records = records[len(records)-limit:]
	}

	var items []SignalItem
	for _, sig := range records {
		items = append(items, SignalItem{
			SignalID:   sig.SignalID,
			Adapter:    sig.Adapter,
			Strength:   sig.Strength,
			Summary:    sig.Summary,
			Timestamp:  sig.Timestamp.Format(time.RFC3339),
			DecisionID: sig.DecisionID,
		})
	}

	jsonResponse(w, SignalResult{
		Signals:   items,
		TokenCost: len(items) * 60,
	})
}

func (s *MCPServer) handlePostCommit(w http.ResponseWriter, r *http.Request) {
	// 接收 Git post-commit hook 的回调
	log.Println("收到 post-commit 回调")
	jsonResponse(w, map[string]string{"status": "ok"})
}

// ---- 辅助函数 ----

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func parseBody(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
