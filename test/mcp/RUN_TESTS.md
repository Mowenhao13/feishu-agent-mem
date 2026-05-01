# MCP 测试运行指南

## 前置条件

1. 确保已设置 `.env` 文件
```bash
cp .env.example .env
# 编辑 .env，设置所需配置
```

2. 确保 Go 版本 >= 1.22

## 环境变量说明

| 环境变量 | 说明 | 测试 |
|---------|------|-----|
| `ARK_API_KEY` | 火山引擎 ARK API Key | Test 08-12 |
| `ARK_BASE_URL` | ARK API 基础地址（可选） | Test 08-12 |
| `ARK_MODEL` | 使用的模型名称（可选） | Test 08-12 |
| `CLAW_APP_ID` | 飞书应用 AppID（用于推送消息） | Test 13 |
| `CLAW_APP_SECRET` | 飞书应用 AppSecret（用于推送消息） | Test 13 |
| `CLAW_CHAT_IDS` | 飞书 Chat IDs（逗号分隔） | Test 13 |
| `LARK_APP_ID` | 飞书应用 AppID（用于接收消息） | Test 14 |
| `LARK_APP_SECRET` | 飞书应用 AppSecret（用于接收消息） | Test 14 |
| `LARK_CHAT_IDS` | 飞书 Chat IDs（逗号分隔） | Test 14 |

## 运行测试

### 方式一：运行所有测试（推荐）

```bash
cd /path/to/feishu-agent-mem

# 运行所有 MCP 测试
go test -v ./test/mcp/...
```

### 方式二：运行特定测试

```bash
# 只运行协议格式测试（不需要 API key）
go test -v ./test/mcp -run "Test_0[1-7]_"

# 只运行真实 LLM 测试（需要 API key）
go test -v ./test/mcp -run "Test_0[8-9]_"
go test -v ./test/mcp -run "Test_1[0-2]_"
```

### 方式三：设置环境变量后运行

```bash
# 显式设置 API key
export ARK_API_KEY="your_api_key_here"
export ARK_BASE_URL="https://ark.cn-beijing.volces.com/api/v3"
export ARK_MODEL="doubao-1-5-pro-32k-250115"

# 然后运行测试
go test -v ./test/mcp
```

## 测试说明

### 测试分类

| 测试编号 | 测试名称 | 需要 API key | 说明 |
|---------|---------|-------------|------|
| 01 | InitializeRequest | ❌ | 初始化请求格式 |
| 02 | ToolsListRequest | ❌ | 工具列表请求格式 |
| 03 | ToolCallRequest | ❌ | 工具调用请求格式 |
| 04 | ResourcesListRequest | ❌ | 资源列表请求格式 |
| 05 | ResourceReadRequest | ❌ | 资源读取请求格式 |
| 06 | ResponseValidation | ❌ | 响应格式验证 |
| 07 | FullProtocolFlow | ❌ | 完整流程模拟 |
| 08 | RealLLM_ExtractDecision | ✅ | 真实 LLM 决策提取 |
| 09 | RealLLM_ClassifyTopic | ✅ | 真实 LLM 议题分类 |
| 10 | RealLLM_DetectCrossTopic | ✅ | 真实 LLM 跨议题检测 |
| 11 | RealLLM_CheckConflict | ✅ | 真实 LLM 冲突评估 |
| 12 | RealLLM_EndToEnd | ✅ | 真实 LLM 端到端流程 |
| 13 | MCPToLarkPush | ⚠️ | MCP-server 推送消息到飞书（需要 CLAW_* 配置） |
| 14 | LarkToMCPReceive | ⚠️ | 从飞书接收消息到 MCP-server（需要 LARK_* 配置） |

### 运行示例

#### 场景1：只运行格式测试（开发时快速验证）

```bash
go test -v ./test/mcp -run "Test_0[1-7]_"
```

输出示例：
```
=== RUN   Test_01_InitializeRequest
=== RUN   Test_02_ToolsListRequest
...
--- PASS: Test_07_FullProtocolFlow (0.00s)
PASS
```

#### 场景2：运行真实 LLM 测试（验证 LLM 集成）

```bash
# 先确认 API key 已设置
echo $ARK_API_KEY

# 运行所有真实 LLM 测试
go test -v ./test/mcp -run "RealLLM"
```

输出示例：
```
=== RUN   Test_08_RealLLM_ExtractDecision
=== RUN   Test_09_RealLLM_ClassifyTopic
=== RUN   Test_10_RealLLM_DetectCrossTopic
=== RUN   Test_11_RealLLM_CheckConflict
=== RUN   Test_12_RealLLM_EndToEnd
...
--- PASS: Test_12_RealLLM_EndToEnd (5.6s)
PASS
```

## 测试脚本

```bash
cd test/mcp

# 列出所有可用的测试请求
./test_mcp.sh list

# 验证所有请求格式
./test_mcp.sh all
```

## 故障排除

### 问题1：找不到包

```
错误: cannot find package "feishu-mem/test/mcp"
```

解决：从项目根目录运行
```bash
cd /path/to/feishu-agent-mem
go test -v ./test/mcp
```

### 问题2：跳过 LLM 测试

```
提示: ARK_API_KEY 未设置，跳过真实 LLM 测试
```

解决：设置 API key
```bash
export ARK_API_KEY="your_api_key"
go test -v ./test/mcp
```

### 问题3：测试超时

```
错误: test timed out after 30s
```

解决：增加超时时间
```bash
go test -v ./test/mcp -timeout 2m
```

## 完整测试流程示例

```bash
# 1. 进入项目根目录
cd /path/to/feishu-agent-mem

# 2. 加载环境变量
source .env

# 3. 先运行格式测试（快速验证）
echo "=== 运行格式测试 ==="
go test -v ./test/mcp -run "Test_0[1-7]_"

# 4. 如果格式测试通过，再运行真实 LLM 测试
echo ""
echo "=== 运行真实 LLM 测试 ==="
go test -v ./test/mcp -run "RealLLM" -timeout 2m
```
