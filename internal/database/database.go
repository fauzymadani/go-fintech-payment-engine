package database

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"FinTechPorto/internal/models"
	"log/slog"
)

var DB *gorm.DB

// getDSN reads DSN from environment or returns a sensible default.
func getDSN() string {
	// fallback DSN
	fallback := "host=localhost user=user password=password dbname=payment_db port=5432 sslmode=disable"
	dsn := os.Getenv("DATABASE_DSN")
	if dsn != "" {
		return dsn
	}
	// support individual env vars if DATABASE_DSN is not provided
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "user"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "password"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "payment_db"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	_ = fallback
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s", host, user, password, dbname, port, sslmode)
}

// Connect opens a DB connection and runs AutoMigrate.
func Connect() error {
	// Load .env file (helpful for local development). It's OK if it's missing in production.
	if err := godotenv.Load(); err == nil {
		slog.Info(".env file loaded")
	} else {
		// Log the error but continue — missing .env is acceptable.
		slog.Info(".env file not found; proceeding without it", "error", err)
	}

	// configure GORM logger
	newLogger := logger.New(
		logWriter{},
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      false,
		},
	)

	dsn := getDSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: newLogger})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		return err
	}

	// Store global DB handle
	DB = db

	// Run auto-migration for models
	if err := AutoMigrate(DB); err != nil {
		slog.Error("auto-migrate failed", "error", err)
		return err
	}

	slog.Info("database connected and migrated")
	return nil
}

// AutoMigrate migrates the schema for the provided models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.Account{}, &models.Transaction{})
}

// logWriter implements gorm logger Writer using slog.
type logWriter struct{}

func (logWriter) Printf(format string, args ...interface{}) {
	// GORM's logger calls Printf; convert to slog.Info
	msg := fmt.Sprintf(format, args...)
	slog.Info(msg)
}
