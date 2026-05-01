package search

import (
	"feishu-mem/internal/decision"
	"sort"
	"time"
)

// SearchResult 搜索结果
type SearchResult struct {
	SDRID         string        `json:"sdr_id"`
	Title         string        `json:"title"`
	Topic         string        `json:"topic"`
	ImpactLevel   string        `json:"impact_level"`
	Status        string        `json:"status"`
	Phase         string        `json:"phase"`
	DecidedAt     *time.Time    `json:"decided_at"`
	RelevanceScore float64      `json:"relevance_score"`
	DecisionSnippet string        `json:"decision_snippet"` // 决策内容摘要
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query         string        `json:"query"`
	Topic         string        `json:"topic,omitempty"`
	ImpactLevel   string        `json:"impact_level,omitempty"`
	Status        string        `json:"status,omitempty"`
	Limit         int           `json:"limit"` // 默认 10
	IncludeSnippet bool          `json:"include_snippet,omitempty"`
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Results     []SearchResult `json:"results"`
	Total       int            `json:"total"`
	TokenCost   int            `json:"token_cost"` // 估算的 token 消耗
}

// SearchOptions 搜索选项（内部使用）
type SearchOptions struct {
	Topic         string
	ImpactLevel   string
	Status        string
	Limit         int
	MaxTokens     int // 最大输出 token 数（~ 20000）
	Dedup         bool
	Project       string
}

// 搜索接口
type Searcher interface {
	Search(req SearchRequest) (*SearchResponse, error)
}

// TokenEstimator token 估算器
type TokenEstimator interface {
	Estimate(results []SearchResult) int
}

// SearchRanker 搜索结果排名器
type SearchRanker interface {
	Rank(results []SearchResult, query string) []SearchResult
}

// SimpleSearchRanker 简单的搜索结果排名器
type SimpleSearchRanker struct{}

func (sr *SimpleSearchRanker) Rank(results []SearchResult, query string) []SearchResult {
	// 简单的相关性评分
	for i, r := range results {
		score := 0.0
		titleLower := stringsToLower(r.Title)
		queryLower := stringsToLower(query)

		// 标题完全匹配
		if titleLower == queryLower {
			score += 10.0
		} else if containsIgnoreCase(r.Title, query) {
			score += 5.0
		}

		// 决策内容匹配
		if containsIgnoreCase(r.DecisionSnippet, query) {
			score += 3.0
		}

		// 影响级别加权
		switch r.ImpactLevel {
		case string(decision.ImpactCritical):
			score += 2.0
		case string(decision.ImpactMajor):
			score += 1.0
		}

		results[i].RelevanceScore = score
	}

	// 按相关性降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].RelevanceScore > results[j].RelevanceScore
	})

	return results
}

// 简单的 token 估算器
type SimpleTokenEstimator struct{}

func (te *SimpleTokenEstimator) Estimate(results []SearchResult) int {
	// 简单估算：每个结果 ~ 50 tokens
	return len(results) * 50
}

// 辅助函数（简化版）
func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	ls := stringsToLower(s)
	lsub := stringsToLower(substr)
	return stringsContains(ls, lsub)
}

func stringsToLower(s string) string {
	var res []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		res = append(res, c)
	}
	return string(res)
}

func stringsContains(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func snippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

