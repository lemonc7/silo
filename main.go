package main

import (
	"github.com/lemonc7/silo/config"
	"github.com/lemonc7/silo/database"
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
}
