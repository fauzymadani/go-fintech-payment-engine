package main

import (
	"context"
	"fmt"
	"os"

	"FinTechPorto/internal/database"
	"FinTechPorto/internal/models"

	"log/slog"
)

func main() {
	// Setup simple logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	// Initialize DB (uses env vars or fallback DSN)
	if err := database.Connect(); err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	accounts := []models.Account{
		{UserID: "user_A", Balance: 1000000, Currency: "IDR"},
		{UserID: "user_B", Balance: 500000, Currency: "IDR"},
	}

	for i := range accounts {
		acc := &accounts[i]
		res := database.DB.WithContext(ctx).Create(acc)
		if res.Error != nil {
			slog.Error("failed to create account", "error", res.Error)
			os.Exit(1)
		}
		fmt.Printf("Created account: user_id=%s id=%s balance=%d currency=%s\n", acc.UserID, acc.ID, acc.Balance, acc.Currency)
	}

	fmt.Println("Seeding completed")
}
