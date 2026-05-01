#!/bin/bash

# OpenClaw 飞书接入快速配置脚本
set -e

echo "========================================="
echo "  OpenClaw 飞书接入配置脚本"
echo "========================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# 步骤 1: 检查环境
echo ""
echo -e "${YELLOW}[1/6] 检查环境...${NC}"

# 检查 .env 文件
if [ ! -f .env ]; then
    echo -e "${RED}错误: .env 文件不存在！${NC}"
    echo "请先复制 .env.example 为 .env 并配置"
    exit 1
fi
echo -e "${GREEN}✓ .env 文件存在${NC}"

# 检查 lark-cli
if ! command -v lark-cli &> /dev/null; then
    echo -e "${RED}警告: lark-cli 未安装${NC}"
    echo "请先安装 lark-cli: https://github.com/larksuite/lark-cli"
else
    echo -e "${GREEN}✓ lark-cli 已安装${NC}"

    # 检查登录状态
    if lark-cli auth status 2>&1 | grep -q "authenticated"; then
        echo -e "${GREEN}✓ lark-cli 已登录${NC}"
    else
        echo -e "${YELLOW}⚠ lark-cli 未登录，请运行: lark-cli auth login${NC}"
    fi
fi

# 步骤 2: 创建 OpenClaw 配置目录
echo ""
echo -e "${YELLOW}[2/6] 创建 OpenClaw 配置目录...${NC}"
OPENCLAW_CONFIG_DIR="$HOME/.openclaw"
mkdir -p "$OPENCLAW_CONFIG_DIR"
echo -e "${GREEN}✓ 目录已创建: $OPENCLAW_CONFIG_DIR${NC}"

# 步骤 3: 生成 openclaw.json 配置
echo ""
echo -e "${YELLOW}[3/6] 生成 OpenClaw 配置文件...${NC}"

# 从 .env 读取配置
source .env 2>/dev/null || true

OPENCLAW_CONFIG="$OPENCLAW_CONFIG_DIR/openclaw.json"

cat > "$OPENCLAW_CONFIG" <<EOF
{
  "mcpServers": {
    "feishu-memory": {
      "command": "none",
      "env": {},
      "url": "http://localhost:37777"
    }
  },
  "skills": {
    "feishu-memory-interface": {
      "enabled": true,
      "config": {
        "memoryServiceUrl": "http://localhost:37777",
        "autoSearchOnMessage": true,
        "maxSearchResults": 5
      }
    }
  },
  "feishu": {
    "appId": "${LARK_APP_ID:-}",
    "appSecret": "${LARK_APP_SECRET:-}",
    "chatIds": "${LARK_CHAT_IDS:-}",
    "enabled": true
  },
  "project": {
    "name": "feishu-mem",
    "defaultTopic": "general"
  }
}
EOF

echo -e "${GREEN}✓ 配置文件已生成: $OPENCLAW_CONFIG${NC}"

# 步骤 4: 编译并启动服务
echo ""
echo -e "${YELLOW}[4/6] 编译并启动记忆系统服务...${NC}"

if command -v go &> /dev/null; then
    echo "正在编译..."
    go build -o bin/mem-service ./cmd/mem-service/
    echo -e "${GREEN}✓ 编译完成${NC}"
else
    echo -e "${YELLOW}⚠ Go 未安装，跳过编译${NC}"
fi

# 步骤 5: 显示配置信息
echo ""
echo -e "${YELLOW}[5/6] 配置信息汇总...${NC}"
echo ""
echo "========================================="
echo "  配置信息"
echo "========================================="
echo "OpenClaw 配置目录: $OPENCLAW_CONFIG_DIR"
echo "OpenClaw 配置文件: $OPENCLAW_CONFIG"
echo "MCP 服务地址: http://localhost:37777"
echo "飞书 Chat ID: ${LARK_CHAT_IDS:-未配置}"
echo ""

# 步骤 6: 测试 MCP 服务
echo ""
echo -e "${YELLOW}[6/6] 测试 MCP 服务连接...${NC}"

# 检查端口是否被占用
if command -v lsof &> /dev/null; then
    if lsof -i :37777 &> /dev/null; then
        echo -e "${GREEN}✓ 端口 37777 已被占用（服务可能正在运行）${NC}"

        # 测试健康检查
        if curl -s http://localhost:37777/health &> /dev/null; then
            echo -e "${GREEN}✓ MCP 服务响应正常${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ 端口 37777 未被占用，请先启动服务${NC}"
    fi
fi

echo ""
echo "========================================="
echo -e "  ${GREEN}配置完成！${NC}"
echo "========================================="
echo ""
echo "下一步操作："
echo ""
echo "1. 启动记忆系统服务："
echo "   make run"
echo "   或"
echo "   docker-compose up -d"
echo ""
echo "2. 验证 OpenClaw 配置："
echo "   检查 $OPENCLAW_CONFIG 文件"
echo ""
echo "3. 测试飞书消息发送："
echo "   lark-cli im +messages-send \\"
echo "     --chat-id \"${LARK_CHAT_IDS:-oc_28cf78aa701166c04ad4425c53c6c225}\" \\"
echo "     --msg-type text \\"
echo "     --content \"🎉 OpenClaw 配置成功！\""
echo ""
echo "详细文档请参阅: docs/openclaw-config-guide.md"
echo ""
