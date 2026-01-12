package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"FinTechPorto/services/transaction/handler"
	"FinTechPorto/services/transaction/repository"

	"FinTechPorto/internal/database"
)

func main() {
	// Configure slog default logger
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	// Initialize database (auto-migrate models)
	if err := database.Connect(); err != nil {
		slog.Error("database initialization failed", "error", err)
		os.Exit(1)
	}

	// Initialize repository and handler
	repo := repository.New(database.DB)
	h := handler.NewHandler(repo)

	// Use handler's router which includes health and the ConnectRPC service
	h2cHandler := h.SetupRouter()

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      h2cHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	slog.Info("starting transaction service", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
