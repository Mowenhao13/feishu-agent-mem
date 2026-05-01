package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	gosignal "os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"feishu-mem/internal/config"
	"feishu-mem/internal/core"
	larkadapter "feishu-mem/internal/lark-adapter"
	"feishu-mem/internal/mcp"
	"feishu-mem/internal/signal"
	"feishu-mem/internal/storage/bitable"
	"feishu-mem/internal/storage/git"
)

func main() {
	// 0. 加载 .env
	_ = godotenv.Load()

	log.Println("Starting feishu-agent-mem service...")

	// 1. 加载配置
	settings := config.DefaultSettings()
	if cfgPath := os.Getenv("CONFIG_PATH"); cfgPath != "" {
		if s, err := config.LoadSettings(cfgPath); err == nil {
			settings = s
		}
	} else if s, err := config.LoadSettings("config/openclaw.yaml"); err == nil {
		settings = s
	}
	larkCfg := larkadapter.LoadConfig()

	log.Printf("  Project: %s", settings.Project.Name)
	log.Printf("  ChatIDs: %v", larkCfg.ChatIDs)
	log.Printf("  MCP Port: %d", settings.MCP.Port)

	// 2. 初始化 Git 存储
	gitStorage, err := git.NewGitStorage(git.Config{
		WorkDir:  settings.Git.WorkDir,
		Remote:   settings.Git.Remote,
		AutoPush: settings.Git.AutoPush,
		Branch:   settings.Git.Branch,
	})
	if err != nil {
		log.Fatalf("Failed to initialize Git storage: %v", err)
	}

	// 3. 初始化内存图
	memoryGraph := core.NewMemoryGraph()
	if settings.Memory.PreloadOnStart {
		log.Println("Loading decisions from Git...")
		if err := memoryGraph.LoadFromGit(gitStorage, settings.Project.Name); err != nil {
			log.Printf("Warning: failed to load from Git: %v", err)
		} else {
			log.Printf("Loaded %d decisions into memory", memoryGraph.Count())
		}
	}

	// 4. 初始化 Bitable 存储
	larkCLI := larkadapter.NewLarkCLI()
	bitableStore := bitable.NewBitableStore(bitable.Config{
		BaseToken: settings.Bitable.BaseToken,
		Tables: bitable.TablesConfig{
			Decision: settings.Bitable.Tables.Decision,
			Topic:    settings.Bitable.Tables.Topic,
		},
	}, larkCLI)

	// 5. 初始化 Pipeline
	pipeline := core.NewPipelineEngine(gitStorage, bitableStore, memoryGraph)

	// 6. 初始化信号引擎
	signalEngine := signal.NewSignalActivationEngine(pipeline, memoryGraph)

	// 7. 创建所有 Detector
	detectors := map[signal.AdapterType]larkadapter.Detector{
		signal.AdapterIM: larkadapter.NewIMExtractor(larkCfg),
	}

	// 8. 状态管理器（用于跟踪 lastCheck）
	stateMgr := larkadapter.NewStateManager(
		filepath.Join(larkadapter.StateDir(), "detect_state.json"),
	)

	// 9. 启动 MCP Server (stdio mode)
	mcpServer := mcp.NewMCPServer(memoryGraph, gitStorage, bitableStore)

	// 检查是否以 stdio mode 启动（通过环境变量或参数）
	// 如果是 MCP_SERVER_MODE=stdio，则直接运行 MCP 协议
	if os.Getenv("MCP_SERVER_MODE") == "stdio" {
		log.Println("Starting MCP server in stdio mode...")
		if err := mcpServer.Start(); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
		return
	}

	// 否则作为正常服务运行，但同时也可以通过子进程方式调用
	log.Println("Running in service mode (MCP stdio server available via subprocess)")

	log.Println("feishu-agent-mem service is ready!")
	log.Printf("  Decisions loaded: %d", memoryGraph.Count())
	log.Printf("  Topics: %d", memoryGraph.TopicCount(settings.Project.Name))
	log.Printf("  MCP port: %d", settings.MCP.Port)

	// 首次立即执行一次检测
	log.Println("Running initial detection cycle...")
	runDetectionCycle(detectors, signalEngine, stateMgr)

	// 10. 主循环
	sigChan := make(chan os.Signal, 1)
	gosignal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(settings.Polling.Interval)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 检测循环 — 核心修复
	go func() {
		for {
			select {
			case <-ticker.C:
				runDetectionCycle(detectors, signalEngine, stateMgr)
			case <-ctx.Done():
				return
			}
		}
	}()

	// 等待退出信号
	sig := <-sigChan
	log.Printf("Received signal: %v, shutting down...", sig)

	if err := mcpServer.Stop(); err != nil {
		log.Printf("Error stopping MCP server: %v", err)
	}

	log.Println("feishu-agent-mem service stopped successfully")
}

// runDetectionCycle 执行一轮检测 → 信号 → 处理
func runDetectionCycle(
	detectors map[signal.AdapterType]larkadapter.Detector,
	engine *signal.SignalActivationEngine,
	stateMgr *larkadapter.StateManager,
) {
	log.Printf("runDetectionCycle: %d detectors", len(detectors))

	for adapter, detector := range detectors {
		lastCheck := stateMgr.GetLastCheck(detector.Name())
		log.Printf("[%s] Starting Detect, lastCheck=%v", detector.Name(), lastCheck)

		result, err := larkadapter.ExtractDetect(detector)
		if err != nil {
			log.Printf("[%s] Detection failed: %v", detector.Name(), err)
			continue
		}

		_ = stateMgr.UpdateLastCheck(detector.Name(), time.Now())

		if !result.HasChanges {
			log.Printf("[%s] No changes detected", detector.Name())
			continue
		}

		log.Printf("[%s] Detected %d changes since %v", detector.Name(), len(result.Changes), lastCheck)

		for _, ch := range result.Changes {
			log.Printf("  → %s [%s] %s", ch.Type, ch.EntityType, ch.Summary)
		}

		// 通过信号引擎处理
		report, err := engine.OnDetectResult(adapter, result)
		if err != nil {
			log.Printf("[%s] Signal processing failed: %v", detector.Name(), err)
			continue
		}
		if report != nil {
			log.Printf("[%s] → %d mutations, %d conflicts", detector.Name(), len(report.Mutations), len(report.Conflicts))
		}
	}

	log.Println("runDetectionCycle: done")
}
