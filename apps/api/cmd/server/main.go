package main

import (
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

	repo, err := app.NewJSONRepository(dataDir)
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}
	service := app.NewService(repo)

	log.Printf("api listening on %s", addr)
	if err := http.ListenAndServe(addr, app.NewRouter(service)); err != nil {
		log.Fatal(err)
	}
}
