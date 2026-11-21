package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"gaobeidian-shangkeyou-miniapp/apps/api/internal/app"
)

func main() {
	username := flag.String("username", "", "staff login name")
	name := flag.String("name", "", "staff display name")
	role := flag.String("role", "", "boss, manager, frontdesk, or finance")
	flag.Parse()
	password := os.Getenv("STAFF_BOOTSTRAP_PASSWORD")
	if *username == "" || *name == "" || *role == "" || password == "" {
		log.Fatal("username, name, role and STAFF_BOOTSTRAP_PASSWORD are required")
	}
	if strings.TrimSpace(os.Getenv("MYSQL_DSN")) == "" {
		log.Fatal("MYSQL_DSN is required")
	}
	repo, err := app.OpenMySQL(os.Getenv("MYSQL_DSN"))
	if err != nil {
		log.Fatal(err)
	}
	defer repo.DB.Close()
	if err := app.ProvisionStaff(repo, *username, *name, app.Role(*role), password); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("staff %s provisioned\n", *username)
}
