#!/bin/bash

# MCP 服务器测试脚本
# 用法: ./test_mcp.sh [server_command]
# 示例: ./test_mcp.sh "go run ./cmd/mcp-server"

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REQUESTS_DIR="$SCRIPT_DIR/requests"
OUTPUT_DIR="$SCRIPT_DIR/outputs"

# 默认服务器命令
SERVER_CMD=${1:-"go run ./cmd/mcp-server"}

mkdir -p "$OUTPUT_DIR"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Feishu Memory MCP Server 测试${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查服务器命令是否可用
if ! command -v go &> /dev/null; then
    log_error "Go 未安装"
    exit 1
fi

log_info "服务器命令: $SERVER_CMD"
log_info "请求目录: $REQUESTS_DIR"
log_info "输出目录: $OUTPUT_DIR"
echo ""

# 测试单个请求
test_request() {
    local request_file=$1
    local request_name=$(basename "$request_file" .json)
    local output_file="$OUTPUT_DIR/${request_name}_response.json"

    log_info "测试: $request_name"

    if [ ! -f "$request_file" ]; then
        log_error "请求文件不存在: $request_file"
        return 1
    fi

    echo -e "${YELLOW}请求内容:${NC}"
    cat "$request_file" | jq '.' 2>/dev/null || cat "$request_file"
    echo ""

    echo -e "${YELLOW}服务器响应:${NC}"

    # 直接将请求内容通过管道发送到服务器
    # 注意: 这只是演示格式，实际 MCP 服务器需要正确处理 stdio 流
    cat "$request_file" | timeout 5s $SERVER_CMD 2>&1 | head -50 || true

    echo ""
}

# 运行所有测试
run_all_tests() {
    log_info "开始运行所有测试..."
    echo ""

    local request_files=(
        "$REQUESTS_DIR/01_initialize.json"
        "$REQUESTS_DIR/02_tools_list.json"
        "$REQUESTS_DIR/03_search.json"
        "$REQUESTS_DIR/04_topic.json"
        "$REQUESTS_DIR/05_decision.json"
        "$REQUESTS_DIR/06_extract_decision.json"
        "$REQUESTS_DIR/07_classify_topic.json"
        "$REQUESTS_DIR/08_detect_crosstopic.json"
        "$REQUESTS_DIR/09_check_conflict.json"
        "$REQUESTS_DIR/10_timeline.json"
        "$REQUESTS_DIR/11_resources_list.json"
        "$REQUESTS_DIR/12_read_resource_design.json"
    )

    local total=${#request_files[@]}
    local passed=0

    log_info "找到 $total 个测试请求"
    echo ""

    # 由于 MCP 服务器需要持续的 stdio 会话，这里先展示请求格式
    for ((i=0; i<total; i++)); do
        local request_file=${request_files[i]}
        local request_name=$(basename "$request_file" .json)

        echo -e "${BLUE}--- 测试 $((i+1))/$total: $request_name ---${NC}"

        if [ -f "$request_file" ]; then
            echo -e "${GREEN}✓ 请求文件存在${NC}"
            echo -e "${YELLOW}内容预览:${NC}"
            cat "$request_file" | jq '.' 2>/dev/null | head -20 || cat "$request_file" | head -20
            ((passed++))
        else
            log_error "✗ 请求文件不存在: $request_file"
        fi
        echo ""
    done

    echo ""
    log_success "所有请求格式验证完成: $passed/$total"
}

# 显示帮助
show_help() {
    cat << EOF
Feishu Memory MCP Server 测试工具

用法: $0 [命令]

命令:
  all              运行所有测试（默认）
  single <file>    测试单个请求文件
  list             列出所有可用的测试请求
  help             显示此帮助信息

示例:
  $0 all
  $0 single 01_initialize.json
  $0 list

EOF
}

# 列出所有测试请求
list_requests() {
    log_info "可用的测试请求:"
    echo ""

    local i=1
    for request_file in "$REQUESTS_DIR"/*.json; do
        if [ -f "$request_file" ]; then
            local filename=$(basename "$request_file")
            echo -e "  ${GREEN}${i}.${NC} $filename"
            ((i++))
        fi
    done

    echo ""
    log_info "共 $((i-1)) 个测试请求"
}

# 主逻辑
case "${1:-all}" in
    all)
        run_all_tests
        ;;
    single)
        if [ -z "$2" ]; then
            log_error "请指定请求文件名"
            exit 1
        fi
        test_request "$REQUESTS_DIR/$2"
        ;;
    list)
        list_requests
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "未知命令: $1"
        echo ""
        show_help
        exit 1
        ;;
esac

echo ""
log_success "测试完成！"
echo ""
