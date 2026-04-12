package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/app"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/config"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
)

func main() {
	cfg := config.Load()

	if err := app.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatal(err)
	}

	db, err := app.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("database connected successfully")

	ghClient := github.NewClient(cfg.GithubToken)

	exists, err := ghClient.RepositoryExists(context.Background(), "golang", "go")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("repo exists: %v\n", exists)

	tag, url, err := ghClient.GetLatestRelease(context.Background(), "golang", "go")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("latest release: tag=%s url=%s\n", tag, url)
	
	log.Println("server started on :" + cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, http.NewServeMux()); err != nil {
		log.Fatal(err)
	}
}
