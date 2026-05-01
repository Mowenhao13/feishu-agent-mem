package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"feishu-mem/internal/config"
	"feishu-mem/internal/core"
	"feishu-mem/internal/mcp"
	"feishu-mem/internal/storage/git"
)

func main() {
	// 1. 加载配置
	_ = godotenv.Load()

	settings := config.DefaultSettings()
	if cfgPath := os.Getenv("CONFIG_PATH"); cfgPath != "" {
		if s, err := config.LoadSettings(cfgPath); err == nil {
			settings = s
		}
	} else if s, err := config.LoadSettings("config/openclaw.yaml"); err == nil {
		settings = s
	}

	// 2. 初始化 Git 存储
	var gitStorage *git.GitStorage
	if settings.Git.WorkDir != "" {
		var err error
		gitStorage, err = git.NewGitStorage(git.Config{
			WorkDir:  settings.Git.WorkDir,
			Remote:   settings.Git.Remote,
			AutoPush: false,
			Branch:   settings.Git.Branch,
		})
		if err != nil {
			log.Printf("Warning: Git storage init failed: %v", err)
		}
	}

	// 3. 初始化内存图
	memoryGraph := core.NewMemoryGraph()
	if gitStorage != nil && settings.Memory.PreloadOnStart {
		if err := memoryGraph.LoadFromGit(gitStorage, settings.Project.Name); err != nil {
			log.Printf("Warning: failed to load from Git: %v", err)
		}
	}

	// 4. 创建 MCP 服务器
	var bitableStore mcp.BitableStoreInterface
	server := mcp.NewMCPServer(memoryGraph, gitStorage, bitableStore)

	// 5. 启动服务器（stdio 模式）
	log.Printf("Feishu Memory MCP Server starting...")
	log.Printf("  Project: %s", settings.Project.Name)
	log.Printf("  Decisions loaded: %d", memoryGraph.Count())

	if err := server.Start(); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
