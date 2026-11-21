package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"gaobeidian-shangkeyou-miniapp/apps/api/internal/app"
)

func main() {
	fromJSON := flag.String("from-json", "", "import an existing JSON demo directory into an empty MySQL database")
	flag.Parse()
	repo, err := app.OpenMySQL(os.Getenv("MYSQL_DSN"))
	if err != nil {
		log.Fatal(err)
	}
	defer repo.DB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := app.MigrateMySQL(ctx, repo.DB); err != nil {
		log.Fatal(err)
	}
	if *fromJSON != "" {
		if err := app.ImportJSONToMySQL(*fromJSON, repo); err != nil {
			log.Fatal(err)
		}
	}
	log.Print("MySQL migration complete")
}
