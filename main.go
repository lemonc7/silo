package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lemonc7/silo/app"
	"github.com/lemonc7/silo/catalog"
	"github.com/lemonc7/silo/config"
	"github.com/lemonc7/silo/database"
	"github.com/lemonc7/silo/release"
)

func main() {
	cfg, err := config.LoadConfig("./config/config.yml")
	if err != nil {
		panic(err)
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
		panic(err)
	}
	defer db.Close()

	rl := release.NewBTClient(cfg.Resource)
	defer rl.Close()

	ctx, stop := context.WithTimeout(context.Background(), 5*time.Minute)
	defer stop()
	if err := rl.EnsureSession(ctx); err != nil {
		panic(err)
	}

	srv := app.NewService(
		db,
		catalog.NewHTTPClient(cfg.TMDB),
		rl,
	)

	if err := srv.InitProfilePriority(ctx, cfg.Resource.Profiles); err != nil {
		panic(err)
	}

	if err := srv.SyncMedia(ctx); err != nil {
		panic(err)
	}
	if err := srv.SyncSeason(ctx); err != nil {
		panic(err)
	}
	if err := srv.SyncEpisode(ctx); err != nil {
		panic(err)
	}
	if err := srv.SyncMoviePage(ctx); err != nil {
		panic(err)
	}
	if err := srv.SyncSeriesPage(ctx); err != nil {
		panic(err)
	}

	if err := srv.SyncMovieMagnets(ctx); err != nil {
		panic(err)
	}
	if err := srv.SyncSeriesMagnets(ctx); err != nil {
		panic(err)
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
