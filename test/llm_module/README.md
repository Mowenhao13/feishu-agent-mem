# LLM 模块测试

参照 docs/llm-module-design.md、docs/prompts.md 和 docs/agent 集成设计价值观.md 实现的完整测试套件。

## 完整测试覆盖

### 1. 提示词管理（Prompts）- 100% 覆盖

| 测试 | 覆盖内容 |
|------|---------|
| TestAllPrompts | 完整的 4 个提示词模板测试 |
| TestPromptManager | 静态段 + 动态段构建测试 |
| TestPromptTemplates | 静态段缓存复用测试 |

#### 4 个完整提示词模板

1. **16.1 决策提取提示词（extraction）** - 从飞书内容提取结构化决策
   - 完整的决策识别信号（中文/英文/Pin消息/待办/审批）
   - 决策状态判定（in_discussion/decided/pending）
   - 精确的输出格式要求（JSON）
   - 严格的规则约束

2. **16.2 议题归属提示词（classification）** - 判定决策所属议题
   - 候选议题列表动态注入
   - 输出格式带置信度和备选议题
   - 优先匹配技术模块名的规则

3. **16.3 跨议题检测提示词（crosstopic）** - 判定决策是否影响其他议题
   - 完整的判定维度表格（决策类型/模块名提及/技术依赖）
   - 输出格式带跨议题引用和理由
   - 默认 Single，只 2 个以上维度指向 Cross 才标记

4. **16.4 冲突评估提示词（conflict）** - 判定两个决策是否语义矛盾
   - 完整的 5 种矛盾类型（直接/参数/时序/范围/无矛盾）
   - 带建议的输出格式
   - 与 decision-tree.md §5.3 对齐的规则

### 2. Token 预算控制（Budget）

| 测试 | 覆盖内容 |
|------|---------|
| TestBudgetTracker | Token 使用追踪和预算检查 |
| TestBudgetLimits | 默认预算配置验证 |

#### 预算配置（对齐 agent 集成设计价值观）

- SearchTool: 20000 tokens（搜索结果 ≤ 2w字符）
- FileRead: 10000 tokens（文件读取）
- TotalPerAgent: 8000 tokens（单个 Agent）
- MaxTotal: 50000 tokens（全局上限）

### 3. 降级策略（Fallback）

| 测试 | 覆盖内容 |
|------|---------|
| TestFallbackExtraction | 关键词匹配提取决策 |
| TestFallbackClassification | 关键词匹配分类议题 |

#### 降级关键词

- 中文: "决定"、"确认"、"结论"、"通过"、"定下来"、"最终方案"
- 英文: "decided"、"LGTM"、"lgtm"、"approve"

### 4. 错误处理（Error Handling）

| 测试 | 覆盖内容 |
|------|---------|
| TestCheckpointManager | 检查点机制（保存/加载/回滚） |
| TestGuardrails | 最大循环次数和 Token 预算限制 |

#### 检查点机制（对齐 agent 集成设计价值观）

```
Step1 (保存) → Step2 (保存) → Step3 (检查点) → Step4 (保存) → Step5 (失败)
```

#### 护栏检查

- 最大循环次数: 10 次（避免死循环）
- 最大 Token 预算: 50000 tokens（避免上下文溢出）

### 5. 搜索工具（Search Tool）

| 测试 | 覆盖内容 |
|------|---------|
| TestSearchTool | 专用搜索工具 |
| TestSearchToolCache | 缓存去重机制 |
| TestSearchToolBudget | 搜索工具预算限制（≤ 2w字符） |

#### 专用工具价值（对齐 agent 集成设计价值观）

- 精确控制输出格式
- 文件缓存去重
- Token 预算检查

### 6. 完整集成测试

| 测试 | 覆盖内容 |
|------|---------|
| TestFullIntegration | 4 个场景完整流程测试 |

## 运行测试

```bash
go test -v ./test/llm_module
```

## 测试输出示例

```
=== RUN   TestAllPrompts
    === RUN   TestAllPrompts/extraction
    === RUN   TestAllPrompts/classification
    === RUN   TestAllPrompts/crosstopic
    === RUN   TestAllPrompts/conflict
--- PASS: TestAllPrompts
=== RUN   TestPromptManager
--- PASS: TestPromptManager
=== RUN   TestPromptTemplates
--- PASS: TestPromptTemplates
=== RUN   TestBudgetTracker
--- PASS: TestBudgetTracker
=== RUN   TestBudgetLimits
--- PASS: TestBudgetLimits
=== RUN   TestFallbackExtraction
--- PASS: TestFallbackExtraction
=== RUN   TestFallbackClassification
--- PASS: TestFallbackClassification
=== RUN   TestCheckpointManager
--- PASS: TestCheckpointManager
=== RUN   TestGuardrails
--- PASS: TestGuardrails
=== RUN   TestSearchTool
--- PASS: TestSearchTool
=== RUN   TestSearchToolCache
--- PASS: TestSearchToolCache
=== RUN   TestSearchToolBudget
--- PASS: TestSearchToolBudget
=== RUN   TestFullIntegration
=== RUN   TestFullIntegration/extraction
=== RUN   TestFullIntegration/classification
=== RUN   TestFullIntegration/crosstopic
=== RUN   TestFullIntegration/conflict
--- PASS: TestFullIntegration
PASS
ok   feishu-mem/test/llm_module 0.007s
```

## 模块设计要点

### 1. 专用工具原则（对齐 agent 集成设计价值观）

每个场景一个专用工具，在通用工具描述里明确写出：哪些场景该用其他工具，不靠模型判断。

### 2. 两级工具加载（对齐 agent 集成设计价值观）

- 模型需要工具 → 调用 Tool Search 传入关键词 → 匹配 searchHint+工具名+说明 → 返回完整定义含参数格式 → 模型正常调用

### 3. 工具输出预算控制（对齐 agent 集成设计价值观）

- 搜索工具: ≤ 20000 tokens（~2w字符）
- 文件读取: 特殊处理

### 4. 提示词设计（对齐 prompts.md）

- 静态段 + 动态段分离
- 指令极其具体：告诉 LLM 何时该输出、何时该跳过、何时该请求更多信息
- 预算控制：每次调用限定在 ~2K-6K tokens

### 5. 错误处理与自我纠正（对齐 agent 集成设计价值观）

- 检查点机制，避免从头再来
- 自我纠正：执行 N 步 → 暂停反思 → 对齐原始目标 → 继续执行 → 回滚 + 重新规划
- 护栏检查：最大循环次数、最大 Token 预算、超时熔断

### 6. 降级策略（对齐 agent 集成设计价值观）

- LLM 不可用时使用纯规则实现
- 关键词匹配的快速路径
- 提供安全可靠的兜底方案

## 4 个场景完整流程

1. **决策提取** - 从飞书内容提取决策（决策提取器）
2. **议题归属** - 判定决策所属议题（议题分类器）
3. **跨议题检测** - 判定决策是否影响其他议题（跨议题影响检测器）
4. **冲突评估** - 判定两个决策是否语义矛盾（决策冲突评估器）

