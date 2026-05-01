#!/bin/bash

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && cd .. && pwd)"
PID_FILE="$PROJECT_ROOT/mem-service.pid"

if [ ! -f "$PID_FILE" ]; then
    echo "feishu-agent-mem is not running (PID file not found)"
    exit 0
fi

PID="$(cat "$PID_FILE")"

if kill -0 "$PID" 2>/dev/null; then
    echo "Stopping feishu-agent-mem (PID: $PID)..."
    kill "$PID"

    # 等待进程结束
    for i in {1..10}; do
        if ! kill -0 "$PID" 2>/dev/null; then
            break
        fi
        sleep 1
    done

    # 如果还在运行，强制 kill
    if kill -0 "$PID" 2>/dev/null; then
        echo "Force stopping feishu-agent-mem..."
        kill -9 "$PID"
    fi
fi

rm -f "$PID_FILE"
echo "feishu-agent-mem stopped"
