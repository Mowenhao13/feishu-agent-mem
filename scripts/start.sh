#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

LOG_DIR="$PROJECT_ROOT/logs"
DATA_DIR="$PROJECT_ROOT/data"
mkdir -p "$LOG_DIR" "$DATA_DIR/decisions"

PID_FILE="$PROJECT_ROOT/mem-service.pid"

# 检查是否已在运行
if [ -f "$PID_FILE" ]; then
    OLD_PID="$(cat "$PID_FILE")"
    if kill -0 "$OLD_PID" 2>/dev/null; then
        echo "feishu-agent-mem is already running (PID: $OLD_PID)"
        exit 0
    fi
fi

# 加载环境变量
if [ -f "$PROJECT_ROOT/.env" ]; then
    export $(grep -v '^#' "$PROJECT_ROOT/.env" | xargs)
fi

# 检查 lark-cli
if ! command -v lark-cli &>/dev/null; then
    echo "Warning: lark-cli not found in PATH"
fi

# 编译（如果需要）
if [ ! -f "$PROJECT_ROOT/bin/mem-service" ] || [ "$PROJECT_ROOT/cmd/mem-service/main.go" -nt "$PROJECT_ROOT/bin/mem-service" ]; then
    echo "Building feishu-agent-mem..."
    go build -o "$PROJECT_ROOT/bin/mem-service" "$PROJECT_ROOT/cmd/mem-service/"
fi

# 后台启动
echo "Starting feishu-agent-mem..."
nohup "$PROJECT_ROOT/bin/mem-service" >"$LOG_DIR/mem-service.log" 2>&1 &
PID=$!
echo $PID >"$PID_FILE"

echo "feishu-agent-mem started (PID: $PID)"
echo "Logs at: $LOG_DIR/mem-service.log"
