package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"gaobeidian-shangkeyou-miniapp/apps/api/internal/app"
)

func main() {
	dataDir := os.Getenv("API_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mode := os.Getenv("APP_ENV")
	if mode == "" {
		mode = "local-demo"
	}
	var service *app.Service
	if mode == "local-demo" {
		repo, err := app.NewJSONRepository(dataDir)
		if err != nil {
			log.Fatalf("init JSON repository: %v", err)
		}
		service = app.NewService(repo)
	} else if mode == "integration" || mode == "production" {
		repo, err := app.OpenMySQL(os.Getenv("MYSQL_DSN"))
		if err != nil {
			log.Fatalf("init MySQL repository: %v", err)
		}
		defer repo.DB.Close()
		if err := app.CheckMySQLSchema(context.Background(), repo.DB); err != nil {
			log.Fatalf("MySQL schema: %v (run cmd/migrate first)", err)
		}
		auth, err := app.NewLiveAuth(repo, os.Getenv("WECHAT_APP_ID"), os.Getenv("WECHAT_APP_SECRET"))
		if err != nil {
			log.Fatal(err)
		}
		service = app.NewService(repo, app.WithLiveAuth(auth), app.WithMode(mode))
	} else {
		log.Fatalf("unsupported APP_ENV %q", mode)
	}

	log.Printf("api environment=%s listening on %s", mode, addr)
	if err := http.ListenAndServe(addr, app.NewRouter(service)); err != nil {
		log.Fatal(err)
	}
}
