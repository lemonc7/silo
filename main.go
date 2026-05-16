package main

import (
	"context"

	"github.com/lemonc7/silo/config"
	"github.com/lemonc7/silo/database"
	"github.com/lemonc7/silo/media"
	"github.com/lemonc7/silo/repo"
	"github.com/lemonc7/silo/service"
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

	client, err := media.NewHTTPClient(cfg.TMDB)
	if err != nil {
		panic(err)
	}

	queries := repo.New(db)
	svc := service.NewMediaService(queries, client)

	ctx := context.Background()
	if err := svc.SyncMedia(ctx); err != nil {
		panic(err)
	}
	if err := svc.SyncTV(ctx); err != nil {
		panic(err)
	}
}
