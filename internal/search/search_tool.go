package search

import (
	"fmt"
	"sync"

	"feishu-mem/internal/decision"
	"feishu-mem/internal/core"
)

// SearchTool 专用搜索工具
// 设计原则：
//  1. 精确控制输出格式
//  2. 文件缓存去重
//  3. Token 预算检查（搜索工具 ≤ 20000 tokens）
type SearchTool struct {
	// 输出预算（token 预算，~20000 字符）
	OutputBudget int

	// 是否去重
	Dedup bool

	// 缓存
	cache map[string]cacheEntry
	cacheMu sync.RWMutex

	// 内存图（数据源）
	memoryGraph *core.MemoryGraph

	// 排名器
	ranker SearchRanker

	// token 估算器
	tokenEstimator TokenEstimator

	// 默认项目
	defaultProject string
}

type cacheEntry struct {
	results []SearchResult
	timestamp int64
}

// NewSearchTool 创建新的搜索工具
func NewSearchTool(memoryGraph *core.MemoryGraph, defaultProject string) *SearchTool {
	return &SearchTool{
		OutputBudget: 20000,
		Dedup: true,
		cache: make(map[string]cacheEntry),
		memoryGraph: memoryGraph,
		ranker: &SimpleSearchRanker{},
		tokenEstimator: &SimpleTokenEstimator{},
		defaultProject: defaultProject,
	}
}

// Search 执行搜索
func (st *SearchTool) Search(req SearchRequest) (*SearchResponse, error) {
	// 1. 检查缓存
	cacheKey := st.buildCacheKey(req)
	if cached, ok := st.getCache(cacheKey); ok {
		return &SearchResponse{
			Results:   cached,
			Total:     len(cached),
			TokenCost: st.tokenEstimator.Estimate(cached),
		}, nil
	}

	// 2. 搜索
	options := st.reqToOptions(req)
	results, err := st.searchInternal(options)
	if err != nil {
		return nil, err
	}

	// 3. 去重
	if st.Dedup {
		results = st.deduplicate(results)
	}

	// 4. 排名
	results = st.ranker.Rank(results, req.Query)

	// 5. 限制数量
	limit := req.Limit
	if limit == 0 {
		limit = 10
	}
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	// 6. 截断到预算
	results = st.truncateToBudget(results, st.OutputBudget)

	// 7. 缓存结果
	st.setCache(cacheKey, results)

	// 8. 返回响应
	return &SearchResponse{
		Results:   results,
		Total:     len(results),
		TokenCost: st.tokenEstimator.Estimate(results),
	}, nil
}

// 内部搜索实现
func (st *SearchTool) searchInternal(opts SearchOptions) ([]SearchResult, error) {
	var results []SearchResult

	if st.memoryGraph == nil {
		return results, nil
	}

	// 从内存图搜索
	var decisions []*decision.DecisionNode
	if opts.Topic != "" {
		decisions = st.memoryGraph.QueryByTopic(opts.Project, opts.Topic)
	} else {
		decisions = st.memoryGraph.GetAllDecisions()
	}

	// 过滤
	for _, d := range decisions {
		// 按议题过滤
		if opts.Topic != "" && d.Topic != opts.Topic {
			continue
		}

		// 按状态过滤
		if opts.Status != "" && string(d.Status) != opts.Status {
			continue
		}

		// 按影响级别过滤
		if opts.ImpactLevel != "" && string(d.ImpactLevel) != opts.ImpactLevel {
			continue
		}

		// 转换为搜索结果
		sr := SearchResult{
			SDRID:         d.SDRID,
			Title:         d.Title,
			Topic:         d.Topic,
			ImpactLevel:   string(d.ImpactLevel),
			Status:        string(d.Status),
			Phase:         d.Phase,
			DecidedAt:     d.DecidedAt,
			RelevanceScore: 0.0,
			DecisionSnippet: snippet(d.Decision, 200),
		}
		results = append(results, sr)
	}

	return results, nil
}

// 去重
func (st *SearchTool) deduplicate(results []SearchResult) []SearchResult {
	seen := make(map[string]bool)
	unique := make([]SearchResult, 0, len(results))
	for _, r := range results {
		if !seen[r.SDRID] {
			seen[r.SDRID] = true
			unique = append(unique, r)
		}
	}
	return unique
}

// 截断到预算（简单估算，按长度）
func (st *SearchTool) truncateToBudget(results []SearchResult, maxChars int) []SearchResult {
	if maxChars <= 0 {
		return results
	}

	var truncated []SearchResult
	currentLen := 0
	for _, r := range results {
		// 估算当前结果的长度
		resultLen := len(r.Title) + len(r.Topic) + len(r.ImpactLevel) + len(r.Status) + len(r.DecisionSnippet)

		if currentLen+resultLen <= maxChars {
			truncated = append(truncated, r)
			currentLen += resultLen
		} else {
			break
		}
	}
	return truncated
}

// 构建缓存键
func (st *SearchTool) buildCacheKey(req SearchRequest) string {
	return fmt.Sprintf("%s|%s|%s|%s|%d", req.Query, req.Topic, req.ImpactLevel, req.Status, req.Limit)
}

// 获取缓存
func (st *SearchTool) getCache(key string) ([]SearchResult, bool) {
	st.cacheMu.RLock()
	defer st.cacheMu.RUnlock()

	entry, ok := st.cache[key]
	if !ok {
		return nil, false
	}
	return entry.results, true
}

// 设置缓存
func (st *SearchTool) setCache(key string, results []SearchResult) {
	st.cacheMu.Lock()
	defer st.cacheMu.Unlock()

	st.cache[key] = cacheEntry{
		results: results,
	}
}

// 清空缓存
func (st *SearchTool) ClearCache() {
	st.cacheMu.Lock()
	defer st.cacheMu.Unlock()

	st.cache = make(map[string]cacheEntry)
}

// 将请求转换为内部选项
func (st *SearchTool) reqToOptions(req SearchRequest) SearchOptions {
	opts := SearchOptions{
		Topic:         req.Topic,
		ImpactLevel:   req.ImpactLevel,
		Status:        req.Status,
		Limit:         req.Limit,
		MaxTokens:     st.OutputBudget,
		Dedup:         st.Dedup,
		Project:       st.defaultProject,
	}
	if opts.Limit == 0 {
		opts.Limit = 10
	}
	return opts
}

// ListTopics 列出所有可用议题
func (st *SearchTool) ListTopics(project string) []string {
	if st.memoryGraph == nil {
		return []string{}
	}
	return st.memoryGraph.ListAllTopics(project)
}

// GetDecision 获取单个决策
func (st *SearchTool) GetDecision(sdrID string) (*decision.DecisionNode, bool) {
	if st.memoryGraph == nil {
		return nil, false
	}
	return st.memoryGraph.GetDecision(sdrID)
}

// SetRanker 设置自定义排名器
func (st *SearchTool) SetRanker(r SearchRanker) {
	if r != nil {
		st.ranker = r
	}
}

// SetTokenEstimator 设置自定义 token 估算器
func (st *SearchTool) SetTokenEstimator(te TokenEstimator) {
	if te != nil {
		st.tokenEstimator = te
	}
}

