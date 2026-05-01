#!/usr/bin/env node

const { Server } = require('@modelcontextprotocol/sdk/server/index.js');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio.js');
const http = require('http');

const MEMORY_SERVICE_URL = process.env.MEMORY_SERVICE_URL || 'http://localhost:37777';

const server = new Server(
    {
        name: 'feishu-memory',
        version: '1.0.0',
    },
    {
        capabilities: {
            tools: {},
        },
    }
);

// 工具定义
const TOOLS = [
    {
        name: 'memory.search',
        description: '搜索记忆系统中的决策记录',
        inputSchema: {
            type: 'object',
            properties: {
                query: { type: 'string', description: '搜索关键词' },
                topic: { type: 'string', description: '限定议题（可选）' },
                limit: { type: 'number', description: '返回数量，默认 10' },
            },
            required: ['query'],
        },
    },
    {
        name: 'memory.topic',
        description: '获取指定议题下的所有决策',
        inputSchema: {
            type: 'object',
            properties: {
                topic: { type: 'string', description: '议题名称' },
            },
            required: ['topic'],
        },
    },
    {
        name: 'memory.decision',
        description: '获取单个决策的完整信息',
        inputSchema: {
            type: 'object',
            properties: {
                sdr_id: { type: 'string', description: '决策 ID' },
            },
            required: ['sdr_id'],
        },
    },
    {
        name: 'memory.timeline',
        description: '获取决策时间线',
        inputSchema: {
            type: 'object',
            properties: {},
        },
    },
    {
        name: 'memory.conflict',
        description: '获取冲突决策列表',
        inputSchema: {
            type: 'object',
            properties: {},
        },
    },
    {
        name: 'memory.signal',
        description: '获取最近的信号列表',
        inputSchema: {
            type: 'object',
            properties: {},
        },
    },
];

// 注册工具
server.setRequestHandler('tools/list', async () => {
    return { tools: TOOLS };
});

// 工具调用处理
server.setRequestHandler('tools/call', async (request) => {
    const { name, arguments: args } = request.params;
    const toolName = name.replace('memory.', '');

    try {
        const result = await makeRequest(`/tools/${toolName}`, args);
        return {
            content: [
                {
                    type: 'text',
                    text: JSON.stringify(result, null, 2),
                },
            ],
        };
    } catch (err) {
        return {
            content: [
                {
                    type: 'text',
                    text: `Error: ${err.message}`,
                },
            ],
            isError: true,
        };
    }
});

// HTTP 请求辅助函数
function makeRequest(path, data) {
    return new Promise((resolve, reject) => {
        const url = new URL(path, MEMORY_SERVICE_URL);
        const postData = JSON.stringify(data);

        const options = {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Content-Length': Buffer.byteLength(postData),
            },
        };

        const req = http.request(url, options, (res) => {
            let body = '';
            res.on('data', (chunk) => {
                body += chunk;
            });
            res.on('end', () => {
                try {
                    const result = JSON.parse(body);
                    resolve(result);
                } catch (err) {
                    reject(new Error('Failed to parse response'));
                }
            });
        });

        req.on('error', (err) => {
            reject(err);
        });

        req.write(postData);
        req.end();
    });
}

// 启动服务
async function main() {
    const transport = new StdioServerTransport();
    await server.connect(transport);
    console.error('Feishu Memory MCP server running');
}

main().catch((err) => {
    console.error('Error starting server:', err);
    process.exit(1);
});
