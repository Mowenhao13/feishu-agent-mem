package git

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"

	"feishu-mem/internal/decision"
	"gopkg.in/yaml.v3"
)

// RenderDecisionFile 渲染决策文件为 YAML frontmatter + Markdown
func RenderDecisionFile(node *decision.DecisionNode) string {
	var buf bytes.Buffer

	// YAML frontmatter
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	_ = enc.Encode(node)
	buf.WriteString("---\n\n")

	// Markdown 正文
	buf.WriteString(fmt.Sprintf("# %s\n\n", node.Title))
	buf.WriteString("## 决策\n\n")
	buf.WriteString(node.Decision + "\n\n")
	if node.Rationale != "" {
		buf.WriteString("## 依据\n\n")
		buf.WriteString(node.Rationale + "\n\n")
	}

	// 元数据
	buf.WriteString("## 元数据\n\n")
	buf.WriteString(fmt.Sprintf("- **议题**: %s\n", node.Topic))
	buf.WriteString(fmt.Sprintf("- **状态**: %s\n", node.Status))
	buf.WriteString(fmt.Sprintf("- **影响级别**: %s\n", node.ImpactLevel))
	if node.Proposer != "" {
		buf.WriteString(fmt.Sprintf("- **提出人**: %s\n", node.Proposer))
	}
	if node.Executor != "" {
		buf.WriteString(fmt.Sprintf("- **执行人**: %s\n", node.Executor))
	}

	return buf.String()
}

// ParseDecisionFile 解析决策文件
func ParseDecisionFile(data []byte) (*decision.DecisionNode, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	// 读取 YAML frontmatter
	var frontmatter []byte
	inFrontmatter := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
			} else {
				break
			}
		} else if inFrontmatter {
			frontmatter = append(frontmatter, line...)
			frontmatter = append(frontmatter, '\n')
		}
	}

	if len(frontmatter) == 0 {
		return nil, fmt.Errorf("no frontmatter found")
	}

	var node decision.DecisionNode
	if err := yaml.Unmarshal(frontmatter, &node); err != nil {
		return nil, err
	}

	return &node, nil
}

// FormatDecisionNode 格式化决策节点为简洁视图
func FormatDecisionNode(node *decision.DecisionNode) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("[%s] %s\n", node.SDRID, node.Title))
	buf.WriteString(fmt.Sprintf("  Topic: %s | Status: %s | Impact: %s\n",
		node.Topic, node.Status, node.ImpactLevel))
	if node.DecidedAt != nil {
		buf.WriteString(fmt.Sprintf("  Decided: %s\n", node.DecidedAt.Format(time.RFC3339)))
	}
	buf.WriteString(fmt.Sprintf("  Commit: %s\n", truncate(node.GitCommitHash, 8)))

	return buf.String()
}
