package main

import (
	"context"
	"log"
	"time"

	"github.com/lemonc7/silo/app"
	"github.com/lemonc7/silo/catalog"
	"github.com/lemonc7/silo/config"
	"github.com/lemonc7/silo/database"
	"github.com/lemonc7/silo/release"
	"github.com/lemonc7/silo/repo"
)

func main() {
	cfg, err := config.LoadConfig("./config/config.yml")
	if err != nil {
		panic(err)
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

	rp := repo.New(db)
	for i, p := range cfg.Resource.Profiles {
		if _, err := rp.UpsertProfilePriority(ctx, repo.UpsertProfilePriorityParams{
			Profile:  p,
			Priority: int64(i),
		}); err != nil {
			log.Printf("[db] 插入磁力优先级标签失败: %v", err)
			continue
		}
	}

	srv := app.NewService(
		rp,
		catalog.NewHTTPClient(cfg.TMDB),
		rl,
	)

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
}
