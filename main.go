package main

import (
	"context"

	"github.com/lemonc7/silo/app"
	"github.com/lemonc7/silo/catalog"
	"github.com/lemonc7/silo/config"
	"github.com/lemonc7/silo/database"
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

	srv := app.NewMediaService(
		repo.New(db),
		catalog.NewHTTPClient(cfg.TMDB),
		nil,
	)

	ctx := context.Background()
	if err := srv.SyncMedia(ctx); err != nil {
		panic(err)
	}
	if err := srv.SyncSeason(ctx); err != nil {
		panic(err)
	}
	if err := srv.SyncEpisode(ctx); err != nil {
		panic(err)
	}
}
