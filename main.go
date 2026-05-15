package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/lemonc7/silo/config"
	"github.com/lemonc7/silo/download"
	"github.com/lemonc7/silo/scheduler"
	"github.com/lemonc7/silo/search"
	"github.com/lemonc7/silo/service"
	"github.com/lemonc7/silo/store"
	"github.com/lemonc7/silo/tmdb"
)

func main() {
	// ── 加载配置 ──────────────────────────────────
	cfg, err := config.LoadConfig("./config.yml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// ── 初始化 Store ──────────────────────────────
	db, err := store.NewSQLiteStore("./silo.db")
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	defer db.Close()

	// ── 初始化 TMDB 客户端 ────────────────────────
	tmdbClient, err := tmdb.NewHTTPClient(cfg.TMDB.BearerToken, cfg.TMDB.AccountID)
	if err != nil {
		log.Fatalf("init tmdb: %v", err)
	}

	// ── 初始化 Rod Page 池 ───────────────────────
	// 注意：需要先启动 headless 浏览器进程
	browserURL := os.Getenv("ROD_BROWSER_URL")
	if browserURL == "" {
		browserURL = "ws://127.0.0.1:9222"
	}
	pool, err := search.NewPool(context.Background(), browserURL, 4)
	if err != nil {
		log.Fatalf("init page pool: %v", err)
	}
	defer pool.Close()

	// ── 初始化 Scraper（当前为空实现） ────────────
	scraper := search.NewNoopScraper(cfg.App.URL)

	// ── 初始化 Downloader（当前为空实现） ─────────
	downloader := download.NewNoopDownloader()

	// ── 组装核心 Service ──────────────────────────
	svc := service.NewSyncService(tmdbClient, pool, scraper, downloader, db)

	// ── 定时调度 ──────────────────────────────────
	sched := scheduler.New()
	sched.AddFunc(cfg.Scheduler.SyncCron, func() {
		if err := svc.SyncAll(context.Background()); err != nil {
			log.Printf("[scheduler] sync error: %v", err)
		}
	})
	sched.Start()
	defer sched.Stop()

	// ── HTTP 接口（手动触发 + 状态查询） ──────────
	mux := http.NewServeMux()

	// 手动触发同步
	mux.HandleFunc("POST /api/sync", func(w http.ResponseWriter, r *http.Request) {
		go func() {
			if err := svc.SyncAll(context.Background()); err != nil {
				log.Printf("[api] sync error: %v", err)
			}
		}()
		w.Write([]byte(`{"status":"started"}`))
	})

	// 健康检查
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Printf("[server] listening on :%d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("[server] %v", err)
		}
	}()

	// ── 优雅退出 ──────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("[main] shutting down...")
	srv.Shutdown(context.Background())
}
