package main

import "github.com/lemonc7/silo/database"

func main() {
	db, err := database.NewDB("./data/data.db")
	if err != nil {
		panic(err)
	}

	defer db.Close()
}
