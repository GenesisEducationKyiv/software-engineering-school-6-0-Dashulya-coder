package main

import (
	"log/slog"
	"os"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/notifier/app"
)

func main() {
	if err := app.Run(); err != nil {
		slog.Error("notifier failed", "error", err)
		os.Exit(1)
	}
}
