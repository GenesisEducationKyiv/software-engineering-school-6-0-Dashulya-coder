package main

import (
	"log"
	"net/http"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/app"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/config"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
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

	m := mailer.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass)

	err = m.SendConfirmation(
		"test@example.com",
		"http://localhost:8080/api/confirm/test-token",
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("confirmation email sent")

	log.Println("server started on :" + cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, http.NewServeMux()); err != nil {
		log.Fatal(err)
	}
}
