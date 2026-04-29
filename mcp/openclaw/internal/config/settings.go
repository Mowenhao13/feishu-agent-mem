package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

// Config 系统配置（openclaw-architecture.md §5.1）
type Config struct {
	Project ProjectConfig `yaml:"project"`
	LarkCLI LarkCLIConfig `yaml:"lark_cli"`
	Git     GitConfig     `yaml:"git"`
	Bitable BitableConfig `yaml:"bitable"`
	Events  EventsConfig  `yaml:"events"`
	Polling PollingConfig `yaml:"polling"`
	Memory  MemoryConfig  `yaml:"memory"`
	Signal  SignalConfig  `yaml:"signal"`
	Service ServiceConfig `yaml:"service"`
}

// ProjectConfig 项目配置
type ProjectConfig struct {
	Name   string   `yaml:"name"`
	Topics []string `yaml:"topics"`
	Phases []string `yaml:"phases"`
}

// LarkCLIConfig lark-cli 配置
type LarkCLIConfig struct {
	Bin             string `yaml:"bin"`
	DefaultIdentity string `yaml:"default_identity"` // user | bot
}

// GitConfig Git 配置
type GitConfig struct {
	WorkDir  string `yaml:"work_dir"`
	Remote   string `yaml:"remote"`
	AutoPush bool   `yaml:"auto_push"`
	Branch   string `yaml:"branch"`
}

// BitableConfig Bitable 配置
type BitableConfig struct {
	BaseToken string        `yaml:"base_token"`
	Tables    BitableTables `yaml:"tables"`
}

// BitableTables 表名配置
type BitableTables struct {
	Decision string `yaml:"decision"`
	Topic    string `yaml:"topic"`
	Phase    string `yaml:"phase"`
	Relation string `yaml:"relation"`
}

// EventSubscription 事件订阅配置
type EventSubscription struct {
	Events       []string `yaml:"events"`
	Filter       string   `yaml:"filter,omitempty"`
	Handler      string   `yaml:"handler"` // direct_pipeline | buffered | log_only
	MaxBatch     int      `yaml:"max_batch,omitempty"`
	IntervalSecs int      `yaml:"interval_secs,omitempty"`
}

// EventsConfig 事件管理配置
type EventsConfig struct {
	Subscriptions []EventSubscription `yaml:"subscriptions"`
}

// PollTask 轮询任务配置
type PollTask struct {
	Time  string   `yaml:"time"`
	Tasks []string `yaml:"tasks"`
}

// ConsistencyCheckConfig 一致性校验配置
type ConsistencyCheckConfig struct {
	IntervalMinutes int  `yaml:"interval_minutes"`
	AutoFix         bool `yaml:"auto_fix"`
}

// PollingConfig 轮询配置
type PollingConfig struct {
	Daily            []PollTask             `yaml:"daily"`
	Weekly           []WeeklyPollTask       `yaml:"weekly"`
	ConsistencyCheck ConsistencyCheckConfig `yaml:"consistency_check"`
}

// WeeklyPollTask 每周轮询任务
type WeeklyPollTask struct {
	Day   string   `yaml:"day"`
	Time  string   `yaml:"time"`
	Tasks []string `yaml:"tasks"`
}

// MemoryConfig 记忆配置
type MemoryConfig struct {
	PreloadOnStart         bool `yaml:"preload_on_start"`
	MaxCacheSize           int  `yaml:"max_cache_size"`
	DirtyFlushIntervalSecs int  `yaml:"dirty_flush_interval_secs"`
}

// SignalConfig 信号引擎配置
type SignalConfig struct {
	StrongBudget int `yaml:"strong_budget"`
	MediumBudget int `yaml:"medium_budget"`
	WeakBudget   int `yaml:"weak_budget"`
}

// MCPConfig MCP Server 配置
type MCPConfig struct {
	Port int `yaml:"port"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Name     string    `yaml:"name"`
	Port     int       `yaml:"port"`
	LogLevel string    `yaml:"log_level"`
	MCP      MCPConfig `yaml:"mcp"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Project: ProjectConfig{
			Name:   "default-project",
			Topics: []string{},
			Phases: []string{"需求评审", "技术选型", "开发", "测试", "上线"},
		},
		LarkCLI: LarkCLIConfig{
			Bin:             "/usr/local/bin/lark-cli",
			DefaultIdentity: "user",
		},
		Git: GitConfig{
			WorkDir:  "./openclaw-data",
			Remote:   "",
			AutoPush: false,
			Branch:   "main",
		},
		Bitable: BitableConfig{
			BaseToken: "",
			Tables: BitableTables{
				Decision: "tbl_decision",
				Topic:    "tbl_topic",
				Phase:    "tbl_phase",
				Relation: "tbl_relation",
			},
		},
		Events: EventsConfig{
			Subscriptions: []EventSubscription{
				{
					Events:  []string{"im.message.receive_v1", "im.message.pin"},
					Handler: "direct_pipeline",
				},
				{
					Events:  []string{"vc.meeting.meeting_ended"},
					Handler: "direct_pipeline",
				},
			},
		},
		Memory: MemoryConfig{
			PreloadOnStart:         true,
			MaxCacheSize:           10000,
			DirtyFlushIntervalSecs: 10,
		},
		Signal: SignalConfig{
			StrongBudget: 6000,
			MediumBudget: 2000,
			WeakBudget:   500,
		},
		Service: ServiceConfig{
			Name:     "openclaw-memory",
			Port:     37777,
			LogLevel: "info",
			MCP: MCPConfig{
				Port: 37777,
			},
		},
	}
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
