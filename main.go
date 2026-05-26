package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/lemonc7/silo/catalog"
	"github.com/lemonc7/silo/config"
	"github.com/lemonc7/silo/database"
	"github.com/lemonc7/silo/download"
	"github.com/lemonc7/silo/release"
	"github.com/lemonc7/silo/server"
	"github.com/lemonc7/silo/service"
)

func main() {
	cfg, err := config.LoadConfig("./config/config.yml")
	if err != nil {
		slog.Error("加载配置文件", slog.String("err", err.Error()))
		os.Exit(1)
	}

	logger := newLogger(cfg.Log)
	slog.SetDefault(logger)
	if loc, err := time.LoadLocation(cfg.Log.TZ); err == nil {
		time.Local = loc
	} else {
		logger.Warn("加载时区失败, 使用默认值", "tz", cfg.Log.TZ, "err", err)
	}

	db, err := database.NewDB("./data/data.db", cfg.Database)
	if err != nil {
		slog.Error("打开数据库", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	rl := release.NewBTClient(cfg.Resource)
	defer rl.Close()

	qb := download.NewQBClient(cfg.Downloader)
	srv := service.NewService(
		db,
		catalog.NewHTTPClient(cfg.TMDB),
		rl,
		qb,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go runSyncScheduler(ctx, srv, cfg.Resource.Profiles, cfg.Worker)

	app := server.InitApp(*cfg, srv)
	app.Run(ctx)
}

func runSyncScheduler(ctx context.Context, srv *service.Service, profiles []string, cfg config.WorkerConfig) {
	if cfg.TMDBSpec == "" {
		cfg.TMDBSpec = "0 2 * * *"
	}
	if cfg.ReleaseSpec == "" {
		cfg.ReleaseSpec = "@every 6h"
	}
	if cfg.DownloadSpec == "" {
		cfg.DownloadSpec = "@every 30m"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Minute
	}

	var running atomic.Bool
	run := func(reason, phase string, fn func(context.Context) error) {
		if !running.CompareAndSwap(false, true) {
			slog.Info("跳过同步任务，上一轮仍在运行", "component", "worker", "phase", phase, "reason", reason)
			return
		}
		defer running.Store(false)

		runCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()

		start := time.Now()
		slog.Info("开始同步任务", "component", "worker", "phase", phase, "reason", reason)
		if err := fn(runCtx); err != nil {
			slog.Error("同步任务失败", "component", "worker", "phase", phase, "reason", reason, "elapsed", time.Since(start), "err", err)
			return
		}
		slog.Info("同步任务完成", "component", "worker", "phase", phase, "reason", reason, "elapsed", time.Since(start))
	}

	c := cron.New(cron.WithLocation(time.Local))
	jobs := []struct {
		phase string
		spec  string
		fn    func(context.Context) error
	}{
		{phase: "tmdb", spec: cfg.TMDBSpec, fn: srv.SyncCatalog},
		{phase: "release", spec: cfg.ReleaseSpec, fn: func(ctx context.Context) error { return srv.SyncReleases(ctx, profiles) }},
		{phase: "download", spec: cfg.DownloadSpec, fn: func(ctx context.Context) error { return srv.SyncDownloads(ctx, profiles) }},
	}

	for _, job := range jobs {
		if _, err := c.AddFunc(job.spec, func() {
			run("cron", job.phase, job.fn)
		}); err != nil {
			slog.Error("注册同步定时任务失败", "component", "worker", "phase", job.phase, "spec", job.spec, "err", err)
			return
		}
	}

	slog.Info("同步定时任务已启动",
		"component", "worker",
		"tmdb_spec", cfg.TMDBSpec,
		"release_spec", cfg.ReleaseSpec,
		"download_spec", cfg.DownloadSpec,
		"timeout", cfg.Timeout,
	)
	c.Start()
	if cfg.RunOnStart {
		go func() {
			for _, job := range jobs {
				run("startup", job.phase, job.fn)
			}
		}()
	}

	<-ctx.Done()
	stopCtx := c.Stop()
	select {
	case <-stopCtx.Done():
	case <-time.After(10 * time.Second):
		slog.Warn("等待同步定时任务停止超时", "component", "worker")
	}
}

func newLogger(cfg config.LogConfig) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}
	if strings.EqualFold(cfg.Format, "json") {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
