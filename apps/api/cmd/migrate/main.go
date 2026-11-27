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
	fromJSON := flag.String("from-json", "", "removed: demo data cannot be imported into MySQL")
	flag.Parse()
	if *fromJSON != "" {
		log.Fatal("demo data cannot be imported into MySQL")
	}
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
	log.Print("MySQL migration complete")
}
