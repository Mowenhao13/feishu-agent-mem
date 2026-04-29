package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/openclaw/internal/adapters"
	"github.com/openclaw/internal/config"
	"github.com/openclaw/internal/core"
	"github.com/openclaw/internal/memory"
	"github.com/openclaw/internal/signal"
)

func main() {
	configPath := flag.String("config", "openclaw.yaml", "配置文件路径")
	flag.Parse()

	log.Println("OpenClaw 记忆系统启动中...")

	// 1. 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("警告: 无法加载配置文件 %s，使用默认配置: %v", *configPath, err)
		cfg = config.DefaultConfig()
	}

	// 2. 初始化 MemoryGraph
	mg := core.NewMemoryGraph()
	log.Printf("MemoryGraph 已初始化")

	// 3. 初始化 GitStorage
	gs, err := adapters.NewGitStorage(cfg.Git.WorkDir, cfg.Git.Remote, cfg.Git.AutoPush)
	if err != nil {
		log.Printf("警告: Git 仓库初始化失败: %v", err)
	} else {
		log.Printf("Git 仓库已就绪: %s", cfg.Git.WorkDir)

		// 从 Git 加载全量决策
		projects, _ := gs.ListProjects()
		for _, project := range projects {
			topics, _ := gs.ListTopics(project)
			for _, topic := range topics {
				decisions, err := gs.ListDecisions(project, topic)
				if err != nil {
					continue
				}
				for _, d := range decisions {
					mg.AddDecision(d)
				}
			}
		}
		log.Printf("从 Git 加载了 %d 条决策到 MemoryGraph", mg.Count())
	}

	// 4. 初始化 BitableStore
	larkCli := adapters.NewLarkCli(cfg.LarkCLI.Bin, cfg.LarkCLI.DefaultIdentity)
	bitable := adapters.NewBitableStore(larkCli, cfg.Bitable.BaseToken, adapters.BitableTableIDs{
		Decision: cfg.Bitable.Tables.Decision,
		Topic:    cfg.Bitable.Tables.Topic,
		Phase:    cfg.Bitable.Tables.Phase,
		Relation: cfg.Bitable.Tables.Relation,
	})
	_ = bitable
	log.Printf("BitableStore 已初始化")

	// 5. 初始化 SignalActivationEngine
	engine := signal.NewSignalActivationEngine(mg)

	// 注册 8 个适配器的信号发射器
	engine.RegisterEmitter(&signal.IMEmitter{})
	engine.RegisterEmitter(&signal.VCEmitter{})
	engine.RegisterEmitter(&signal.DocsEmitter{})
	engine.RegisterEmitter(&signal.CalendarEmitter{})
	engine.RegisterEmitter(&signal.TaskEmitter{})
	engine.RegisterEmitter(&signal.OKREmitter{})
	engine.RegisterEmitter(&signal.ContactEmitter{})
	engine.RegisterEmitter(&signal.WikiEmitter{})

	// 设置决策变更回调 → 写入 Git
	engine.SetMutationHandler(func(mutation *core.DecisionMutation) {
		log.Printf("决策变更: %s %s", mutation.Type, mutation.SDRID)

		if mutation.Decision != nil && mutation.Type == "create" {
			hash, err := gs.WriteDecision(mutation.Decision)
			if err != nil {
				log.Printf("写入 Git 失败: %v", err)
				return
			}
			log.Printf("决策已写入 Git: %s (hash: %s)", mutation.Decision.SDRID, hash)

			// 同步到 Bitable
			if err := bitable.UpsertDecision(context.Background(), mutation.Decision); err != nil {
				log.Printf("同步到 Bitable 失败: %v", err)
			}
		}
	})

	log.Println("SignalActivationEngine 已初始化")

	// 6. 启动 MCP Server
	mcpServer := memory.NewMCPServer(cfg.Service.MCP.Port, mg)
	go func() {
		if err := mcpServer.Start(); err != nil {
			log.Fatalf("MCP Server 启动失败: %v", err)
		}
	}()
	log.Printf("MCP Server 已启动, 端口: %d", cfg.Service.MCP.Port)

	// 7. 初始化 PipelineEngine
	pipeline := core.NewPipelineEngine()
	_ = pipeline
	log.Println("PipelineEngine 已初始化")

	// 8. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Println("OpenClaw 记忆系统就绪")
	<-quit

	log.Println("正在关闭...")
	log.Println("OpenClaw 已停止")
}
