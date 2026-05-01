package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const chatID = "oc_28cf78aa701166c04ad4425c53c6c225"

func main() {
	// 模拟飞书群聊中产生的一条决策性消息
	decisionMsg := `决定了，就用Go实现记忆系统。Confirm采用Go 1.22，数据库用PostgreSQL，缓存用Redis。大家没意见就approve这个方案。`

	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║   lark-sender — 模拟决策消息推送测试          ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("▸ 模拟决策内容:")
	fmt.Printf("  「%s」\n\n", decisionMsg)

	// 1. 发送模拟决策消息到飞书群聊
	fmt.Println("▸ [1/4] 发送模拟消息到群聊...")
	cmd := exec.Command("lark-cli", "im", "+messages-send",
		"--chat-id", chatID,
		"--text", fmt.Sprintf("决定了，就用Go实现记忆系统。采用Go 1.22，数据库用PostgreSQL，缓存用Redis。大家确认一下。"),
		"--as", "user",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  ❌ 发送失败: %v\n%s\n", err, string(out))
	} else {
		fmt.Println("  ✅ 已发送到飞书群聊")
		fmt.Printf("  返回: %s\n", string(out))
	}

	// 2. 检查 Git Storage
	fmt.Println()
	fmt.Println("▸ [2/4] 检查 Git Storage 决策记录...")
	time.Sleep(2 * time.Second)

	gitCheck := exec.Command("docker", "exec", "openclaw-zh",
		"bash", "-c",
		"cd /root/openclaw-workspace/feishu-agent-mem && "+
			"ls -la data/decisions/ 2>/dev/null && "+
			"git -C data log --oneline -5 2>/dev/null || echo '(no git commits yet)'")
	gitOut, gitErr := gitCheck.CombinedOutput()
	if gitErr != nil {
		fmt.Printf("  ⚠️  检查 Git 失败: %v\n", gitErr)
	} else {
		fmt.Printf("  Git 仓库状态:\n%s\n", string(gitOut))
	}

	// 3. 检查 Bitable 多维表格
	fmt.Println("▸ [3/4] 检查 Bitable 多维表格新记录...")
	time.Sleep(1 * time.Second)

	bitableCheck := exec.Command("lark-cli", "base", "+record-list",
		"--base-token", "NnnMb5mWJaBJkXsHf69cIpfMn8b",
		"--table-id", "tblEBXkSxaqxnY6l",
	)
	bitableOut, bitableErr := bitableCheck.CombinedOutput()
	if bitableErr != nil {
		fmt.Printf("  ⚠️  检查 Bitable 失败: %v\n", bitableErr)
	} else {
		var result map[string]any
		if err := json.Unmarshal(bitableOut, &result); err == nil {
			if data, ok := result["data"].(map[string]any); ok {
				if records, ok := data["data"].([]any); ok {
					fmt.Printf("  📊 Bitable 决策表记录数: %d 条\n", len(records))
					for i, r := range records {
						fmt.Printf("     [%d] %v\n", i+1, r)
					}
				}
			}
		}
	}

	// 4. 检查 OpenClaw LLM 调用
	fmt.Println()
	fmt.Println("▸ [4/4] 检查 OpenClaw LLM 调用记录...")
	time.Sleep(1 * time.Second)

	llmCheck := exec.Command("docker", "exec", "openclaw-zh",
		"bash", "-c",
		"ls -la /root/.openclaw/agents/main/sessions/*.jsonl 2>/dev/null && "+
			"tail -20 /root/.openclaw/agents/main/sessions/*.jsonl 2>/dev/null || echo '(no session logs)'")
	llmOut, llmErr := llmCheck.CombinedOutput()
	if llmErr != nil {
		fmt.Printf("  ⚠️  检查 LLM 调用记录失败: %v\n", llmErr)
	} else {
		fmt.Printf("  OpenClaw 会话日志:\n%s\n", string(llmOut))
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║   测试完成                                    ║")
	fmt.Println("║   Bitable: https://jcneyh7qlo8i.feishu.cn    ║")
	fmt.Println("║            /base/NnnMb5mWJaBJkXsHf69cIpfMn8b ║")
	fmt.Println("║   Dashboard: http://127.0.0.1:18789/         ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
}
