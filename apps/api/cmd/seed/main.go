package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"gaobeidian-shangkeyou-miniapp/apps/api/internal/app"
)

func main() {
	if mode := os.Getenv("APP_ENV"); mode != "" && mode != "local-demo" {
		log.Fatal("seed is available only in local-demo")
	}
	reset := flag.Bool("reset", false, "reset local demo data")
	flag.Parse()

	dataDir := os.Getenv("API_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}

	repo, err := app.NewJSONRepository(dataDir)
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}
	service := app.NewService(repo)

	if *reset {
		state, err := service.ResetDemo(context.Background(), "stf_boss", "RESET")
		if err != nil {
			log.Fatalf("reset demo: %v", err)
		}
		fmt.Printf("reset demo data: members=%d orders=%d coupons=%d\n", state.MemberCount, state.OrderCount, state.CouponCount)
		return
	}

	state, err := service.GetDemoState(context.Background())
	if err != nil {
		log.Fatalf("get demo state: %v", err)
	}
	fmt.Printf("demo data ready: members=%d orders=%d coupons=%d\n", state.MemberCount, state.OrderCount, state.CouponCount)
}
