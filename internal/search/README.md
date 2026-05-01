# SearchTool 搜索工具

## 概述

SearchTool 是记忆系统的专用搜索工具，遵循 agent 集成设计价值观的设计原则：

1. **专用工具原则** - 精确控制输出格式，不使用万能工具
2. **两级工具加载** - 轻量描述 + 完整定义按需加载
3. **工具输出预算控制** - 搜索工具 ≤ 20000 tokens
4. **缓存去重** - 内置缓存机制，避免重复搜索

## 目录结构

```
internal/search/
├── types.go         // 类型定义
├── search_tool.go   // SearchTool 核心实现
└── README.md        // 本文档
```

## 核心功能

### 1. 搜索决策

```go
req := search.SearchRequest{
    Query:         "数据库迁移",
    Topic:         "数据库架构",  // 可选，按议题过滤
    ImpactLevel:   "major",         // 可选，按影响级别过滤
    Status:        "active",        // 可选，按状态过滤
    Limit:         10,              // 可选，默认 10
}

resp, err := st.Search(req)
```

### 2. 搜索结果

```go
type SearchResult struct {
    SDRID          string     // 决策ID
    Title          string     // 标题
    Topic          string     // 议题
    ImpactLevel    string     // 影响级别
    Status         string     // 状态
    Phase          string     // 阶段
    DecidedAt      *time.Time // 决策时间
    RelevanceScore float64    // 相关性分数
    DecisionSnippet string      // 决策内容摘要
}
```

### 3. 完整示例

```go
package main

import (
    "feishu-mem/internal/search"
    "feishu-mem/internal/core"
    "feishu-mem/internal/decision"
)

func main() {
    // 1. 创建内存图
    mg := core.NewMemoryGraph()

    // 2. 创建 SearchTool
    st := search.NewSearchTool(mg, "my-project")

    // 3. 搜索
    req := search.SearchRequest{
        Query: "数据库",
        Limit: 10,
    }
    resp, err := st.Search(req)
    if err != nil {
        panic(err)
    }

    // 4. 使用结果
    for _, result := range resp.Results {
        println(result.Title)
    }
}
```

## API 文档

### SearchTool 构造函数

```go
func NewSearchTool(memoryGraph *core.MemoryGraph, defaultProject string) *SearchTool
```

### SearchTool 方法

| 方法 | 说明 |
|------|------|
| Search(req) | 执行搜索，返回搜索响应 |
| ListTopics(project) | 列出所有可用议题 |
| GetDecision(sdrID) | 获取单个决策 |
| SetRanker(r) | 设置自定义排名器 |
| SetTokenEstimator(te) | 设置自定义 token 估算器 |
| ClearCache() | 清空缓存 |

### 搜索排名器

内置了 `SimpleSearchRanker` 简单排名器，支持自定义实现：

```go
type SearchRanker interface {
    Rank(results []SearchResult, query string) []SearchResult
}
```

### Token 估算器

内置了 `SimpleTokenEstimator`，支持自定义实现：

```go
type TokenEstimator interface {
    Estimate(results []SearchResult) int
}
```

## 预算控制

SearchTool 严格遵循预算限制：

```go
type TokenBudget struct {
    SearchTool:    20000,  // 搜索工具 ≤ 20000 tokens
    FileRead:      10000,  // 文件读取
    TotalPerAgent: 8000,  // 单个 Agent
    MaxTotal:      50000, // 全局
}
```

预算实现：
- `SearchTool.OutputBudget = 20000` - 搜索工具输出限制
- 搜索结果自动截断到预算内
- Token 估算器提供成本估算

## 缓存机制

SearchTool 内置缓存：

- 按查询条件缓存
- 自动去重（Dedup）
- `ClearCache()` 清空缓存

## 专用工具价值（对齐 agent 集成设计价值观）

1. **精确控制输出格式**
   - 统一的 SearchResult 结构
   - 可预测的响应格式

2. **文件缓存去重**
   - 内置缓存避免重复查询
   - Deduplicate 机制防止重复结果

3. **Token 预算检查**
   - 输出截断到预算内
   - Token 成本估算

4. **在通用工具描述里明确写出**
   - 哪些场景该用其他工具
   - 不靠模型判断

## 与 MemoryGraph 集成

SearchTool 使用 `core.MemoryGraph` 作为数据源：

- 内存索引，快速检索
- 支持按 Topic 查询
- 支持跨 Topic 查询
- 支持关键词搜索

## 与其他模块关系

```
SearchTool
├── core.MemoryGraph (数据源)
├── decision.DecisionNode (决策节点)
└── (可选) LLM 集成 (高级排序/语义搜索)
```

## 测试

测试文件位于：`test/llm_module/search_test.go`

```bash
go test -v ./test/llm_module
```

## 设计要点总结

✅ **专用工具原则** - 只负责搜索，不做其他工作
✅ **预算控制** - 20000 token 上限
✅ **缓存去重** - 避免重复查询
✅ **自定义扩展** - 支持自定义排名器和 token 估算器
✅ **内存优化** - 使用 MemoryGraph 内存索引

## 下一步

- 集成语义搜索（可选）
- 添加更多过滤条件
- 优化排名算法
- 集成 MCP Server

