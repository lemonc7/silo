package main

import (
	"context"
	"fmt"

	"github.com/lemonc7/silo/config"
	"github.com/lemonc7/silo/database"
	"github.com/lemonc7/silo/tmdb"
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

	client, err := tmdb.NewHTTPClient(cfg.TMDB)
	if err != nil {
		panic(err)
	}

	data, err := client.FetchMedia(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println(data)
}
