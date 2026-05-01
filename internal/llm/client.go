package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

// Config LLM 配置
type Config struct {
	APIKey     string
	BaseURL    string
	Model      string
}

// Client LLM 客户端
type Client struct {
	config *Config
}

// NewClient 创建 LLM 客户端
func NewClient() *Client {
	return &Client{
		config: LoadConfig(),
	}
}

// LoadConfig 从环境变量加载配置
func LoadConfig() *Config {
	// 尝试多个路径加载 .env
	paths := []string{
		".env",
		"../.env",
		"../../.env",
	}

	for _, path := range paths {
		godotenv.Load(path)
	}

	return &Config{
		APIKey:  os.Getenv("ARK_API_KEY"),
		BaseURL: os.Getenv("ARK_BASE_URL"),
		Model:   os.Getenv("ARK_MODEL"),
	}
}

// IsAvailable 检查 LLM 是否可用
func (c *Client) IsAvailable() bool {
	return c.config.APIKey != ""
}

// Call 调用 LLM
func (c *Client) Call(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if c.config.APIKey == "" {
		return "", fmt.Errorf("ARK_API_KEY is not set")
	}

	baseURL := c.config.BaseURL
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}

	modelName := c.config.Model
	if modelName == "" {
		modelName = "doubao-1-5-pro-32k-250115"
	}

	client := arkruntime.NewClientWithApiKey(c.config.APIKey, arkruntime.WithBaseUrl(baseURL))

	req := model.CreateChatCompletionRequest{
		Model: modelName,
		Messages: []*model.ChatCompletionMessage{
			{
				Role: model.ChatMessageRoleSystem,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(systemPrompt),
				},
			},
			{
				Role: model.ChatMessageRoleUser,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(userPrompt),
				},
			},
		},
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("llm call failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == nil {
		return "", fmt.Errorf("no response from llm")
	}

	return *resp.Choices[0].Message.Content.StringValue, nil
}

// ExtractJSON 从 LLM 响应中提取 JSON
func ExtractJSON(content string) string {
	cleaned := content

	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		if end := strings.Index(content[start:], "```"); end != -1 {
			cleaned = strings.TrimSpace(content[start : start+end])
		}
	} else if strings.Contains(content, "```") {
		start := strings.Index(content, "```") + 3
		if end := strings.Index(content[start:], "```"); end != -1 {
			cleaned = strings.TrimSpace(content[start : start+end])
		}
	} else {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start != -1 && end != -1 && end > start {
			cleaned = content[start : end+1]
		}
	}

	return cleaned
}

// ParseExtractionResult 解析决策提取结果
func ParseExtractionResult(content string) (*ExtractionResult, error) {
	jsonStr := ExtractJSON(content)

	var result ExtractionResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("json parse failed: %w", err)
	}

	return &result, nil
}

// ParseClassificationResult 解析议题分类结果
func ParseClassificationResult(content string) (*ClassificationResult, error) {
	jsonStr := ExtractJSON(content)

	var result ClassificationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("json parse failed: %w", err)
	}

	return &result, nil
}

// ParseCrossTopicResult 解析跨议题检测结果
func ParseCrossTopicResult(content string) (*CrossTopicResult, error) {
	jsonStr := ExtractJSON(content)

	var result CrossTopicResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("json parse failed: %w", err)
	}

	return &result, nil
}

// ParseConflictResult 解析冲突评估结果
func ParseConflictResult(content string) (*ConflictResult, error) {
	jsonStr := ExtractJSON(content)

	var result ConflictResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("json parse failed: %w", err)
	}

	return &result, nil
}
