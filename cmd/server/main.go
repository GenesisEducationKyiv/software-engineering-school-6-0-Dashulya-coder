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

	subRepo := repository.NewSubscriptionRepository(db)

	err = subRepo.ConfirmByToken(context.Background(), "confirm123")
	if err != nil {
		log.Fatal(err)
	}

	sub, err := subRepo.FindByConfirmToken(context.Background(), "confirm123")
	if err != nil {
		log.Fatal(err)
	}
	if sub == nil {
		log.Fatal("subscription not found")
	}

	log.Printf("after confirm: %+v\n", *sub)

	log.Println("server started on :" + cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, http.NewServeMux()); err != nil {
		log.Fatal(err)
	}
}
