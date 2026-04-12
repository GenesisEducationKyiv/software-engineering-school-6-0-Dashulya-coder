package main

import (
	"log"
	"net/http"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/app"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/config"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/router"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/service"
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
	smtpMailer := mailer.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass)

	subRepo := repository.NewSubscriptionRepository(db)
	repoRepo := repository.NewGitHubRepository(db)

	subService := service.NewSubscriptionService(
		subRepo,
		repoRepo,
		ghClient,
		smtpMailer,
		cfg.BaseURL,
	)

	subHandler := handlers.NewSubscriptionHandler(subService)
	r := router.New(subHandler)

	log.Println("server started on :" + cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
	
	log.Println("server started on :" + cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, http.NewServeMux()); err != nil {
		log.Fatal(err)
	}
}
