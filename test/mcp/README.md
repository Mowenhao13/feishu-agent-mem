# MCP 协议测试

本目录包含 MCP 服务器的完整测试套件，包括手动构建的 JSON-RPC 2.0 请求消息。

## 目录结构

```
test/mcp/
├── README.md                   # 本文档
├── mcp_protocol_test.go        # Go 单元测试
├── test_mcp.sh                # 测试脚本
├── requests/                  # 请求示例
│   ├── 01_initialize.json
│   ├── 02_tools_list.json
│   ├── 03_search.json
│   ├── 04_topic.json
│   ├── 05_decision.json
│   ├── 06_extract_decision.json
│   ├── 07_classify_topic.json
│   ├── 08_detect_crosstopic.json
│   ├── 09_check_conflict.json
│   ├── 10_timeline.json
│   ├── 11_resources_list.json
│   └── 12_read_resource_design.json
└── outputs/                   # 输出目录（运行测试时生成）
```

## 使用方式

### 方式一：Go 单元测试

```bash
cd /path/to/feishu-agent-mem
go test -v ./test/mcp
```

### 方式二：测试脚本

```bash
cd /path/to/feishu-agent-mem/test/mcp

# 查看帮助
./test_mcp.sh help

# 列出所有测试请求
./test_mcp.sh list

# 运行所有测试（验证请求格式）
./test_mcp.sh all
```

### 方式三：手动测试请求

你可以直接使用这些 JSON 请求来测试 MCP 服务器：

```bash
# 1. 启动 MCP 服务器（在另一个终端）
cd /path/to/feishu-agent-mem
make build-mcp
./bin/mcp-server

# 2. 在另一个终端测试（手动方式）
# 注意：实际使用时需要完整的会话管理
cat test/mcp/requests/01_initialize.json
cat test/mcp/requests/02_tools_list.json
```

## 请求说明

### 01_initialize.json - 初始化连接

建立与 MCP 服务器的连接，交换能力信息。

### 02_tools_list.json - 列出可用工具

获取服务器提供的所有工具列表。

### 03_search.json - 搜索决策

搜索记忆系统中的决策记录。

### 04_topic.json - 查询议题

获取指定议题下的所有决策。

### 05_decision.json - 决策详情

获取单个决策的详细信息。

### 06_extract_decision.json - 提取决策（LLM）

从文本内容中智能提取决策信息。

### 07_classify_topic.json - 分类议题（LLM）

将决策智能分类到正确的议题。

### 08_detect_crosstopic.json - 跨议题检测（LLM）

检测决策是否影响多个议题。

### 09_check_conflict.json - 冲突评估（LLM）

评估两个决策之间是否存在冲突。

### 10_timeline.json - 时间线

获取决策历史时间线。

### 11_resources_list.json - 列出资源

获取服务器提供的所有资源。

### 12_read_resource_design.json - 读取资源

读取设计文档资源。

## JSON-RPC 2.0 协议说明

### 请求格式

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "method_name",
  "params": {
    "key": "value"
  }
}
```

### 响应格式

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "key": "value"
  }
}
```

### 错误响应

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32000,
    "message": "错误描述"
  }
}
```

## MCP 协议标准方法

### 标准方法列表

| 方法 | 描述 |
|------|------|
| `initialize` | 初始化连接 |
| `tools/list` | 列出可用工具 |
| `tools/call` | 调用工具 |
| `resources/list` | 列出可用资源 |
| `resources/read` | 读取资源 |
| `prompts/list` | 列出提示词 |
| `prompts/get` | 获取提示词 |

## 完整测试会话示例

下面是一个完整的 MCP 会话流程：

```
1. 客户端发送 initialize 请求
2. 服务器返回 initialize 响应（包含 capabilities）
3. 客户端发送 tools/list 请求
4. 服务器返回可用工具列表
5. 客户端发送 resources/list 请求
6. 服务器返回可用资源列表
7. 客户端调用工具（如 search）
8. 服务器返回工具执行结果
9. （可选）继续调用更多工具...
```

## 扩展说明

### 添加新的测试请求

要添加新的测试请求，只需在 `requests/` 目录下创建新的 JSON 文件，
遵循相同的命名规范和 JSON-RPC 2.0 格式。

### 测试响应验证

实际的响应验证可以通过以下方式：
- 运行完整的 Go 单元测试 (`mcp_protocol_test.go`)
- 手动检查日志输出
- 使用 MCP 测试工具

## 相关文档

- `docs/mcp-server-usage.md` - MCP 服务器使用指南
- `internal/mcp/server.go` - MCP 服务器实现
- `docs/mcp-server-example.md` - MCP 协议说明
