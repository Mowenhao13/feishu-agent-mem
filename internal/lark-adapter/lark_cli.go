package larkadapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// LarkCLI lark-cli 调用封装
type LarkCLI struct{}

// NewLarkCLI 创建 lark-cli 封装
func NewLarkCLI() *LarkCLI {
	return &LarkCLI{}
}

// RunCommand 运行 lark-cli 命令
func (l *LarkCLI) RunCommand(args ...string) ([]byte, error) {
	cmd := exec.Command("lark-cli", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("lark-cli %s failed: %w, stderr: %s", strings.Join(args, " "), err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// RunCommandJSON 运行命令并解析 JSON 结果
func (l *LarkCLI) RunCommandJSON(result any, args ...string) error {
	output, err := l.RunCommand(args...)
	if err != nil {
		return err
	}
	return json.Unmarshal(output, result)
}

// TryCommand 尝试运行命令，失败时返回模拟数据并标记为模拟
func (l *LarkCLI) TryCommand(args ...string) (output []byte, isMock bool, err error) {
	output, err = l.RunCommand(args...)
	if err != nil {
		return nil, true, err
	}
	return output, false, nil
}
