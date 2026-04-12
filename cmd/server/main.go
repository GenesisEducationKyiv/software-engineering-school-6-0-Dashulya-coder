package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/app"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/config"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
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

	repoRepo := repository.NewGitHubRepository(db)

	err = repoRepo.UpdateLastSeenTag(
		context.Background(),
		1,
		"go1.24.1",
		"https://github.com/golang/go/releases/tag/go1.24.1",
	)
	if err != nil {
		log.Fatal(err)
	}

	updatedRepo, err := repoRepo.GetByID(context.Background(), 1)
	if err != nil {
		log.Fatal(err)
	}
	if updatedRepo == nil {
		log.Fatal("repository not found by id")
	}

	log.Printf("updated repository: %+v\n", *updatedRepo)
	
	log.Println("server started on :" + cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, http.NewServeMux()); err != nil {
		log.Fatal(err)
	}
}
