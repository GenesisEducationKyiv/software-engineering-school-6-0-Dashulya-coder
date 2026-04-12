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

	//err = repoRepo.Create(context.Background(), &model.GitHubRepository{
	//	FullName: "golang/go",
	//	Owner:    "golang",
	//	Name:     "go",
	//})
	//if err != nil {
	//	log.Fatal(err)
	//}

	foundRepo, err := repoRepo.FindByFullName(context.Background(), "golang/go")
	if err != nil {
		log.Fatal(err)
	}
	if foundRepo == nil {
		log.Fatal("repository not found")
	}

	log.Printf("found repository: %+v\n", *foundRepo)

	log.Println("server started on :" + cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, http.NewServeMux()); err != nil {
		log.Fatal(err)
	}
}
