package adapters

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CliExecutor 实际的 CLI 命令执行（openclaw-architecture.md §2.2.1）
type CliExecutor struct {
	bin      string
	identity string // user | bot
}

// NewCliExecutor 创建 CLI 执行器
func NewCliExecutor(bin, identity string) *CliExecutor {
	return &CliExecutor{
		bin:      bin,
		identity: identity,
	}
}

// CommandOutput 命令执行输出
type CommandOutput struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Run 执行 lark-cli 命令
func (ce *CliExecutor) Run(ctx context.Context, args ...string) ([]byte, error) {
	// 构建完整参数
	cmdArgs := make([]string, 0, len(args)+2)
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, "--as", ce.identity)

	cmd := exec.CommandContext(ctx, ce.bin, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("lark-cli exec failed: %w\nstderr: %s", err, stderr.String())
	}

	if stderr.Len() > 0 {
		// 非致命错误日志
		_ = stderr.String()
	}

	return stdout.Bytes(), nil
}

// RunWithSchema 带 schema 检查的调用（openclaw-architecture.md §2.2.1）
func (ce *CliExecutor) RunWithSchema(ctx context.Context, resource, method string) ([]byte, error) {
	// 先查看 schema
	schemaArgs := []string{"schema", fmt.Sprintf("%s.%s", resource, method)}
	schemaOutput, err := ce.Run(ctx, schemaArgs...)
	if err != nil {
		return nil, fmt.Errorf("schema check failed: %w", err)
	}

	// 执行命令（JSON 格式输出）
	execArgs := []string{resource, method, "--format", "json"}
	return ce.Run(ctx, execArgs...)
}

// ResolveUser 解析 open_id 为联系人信息
func (ce *CliExecutor) ResolveUser(ctx context.Context, openID string) ([]byte, error) {
	return ce.Run(ctx, "contact", "+get-user", "--user-id", openID)
}

// GetCommand returns the full command string for logging
func (ce *CliExecutor) GetCommand(args ...string) string {
	all := append([]string{ce.bin}, args...)
	all = append(all, "--as", ce.identity)
	return strings.Join(all, " ")
}
