package main

import (
	"log/slog"
	"os"

	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		slog.Error("notifier failed", "error", err)
		os.Exit(1)
	}
}
