package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"FinTechPorto/services/transaction/handler"
	"FinTechPorto/services/transaction/repository"

	"FinTechPorto/internal/broker"
	"FinTechPorto/internal/database"
	"strings"
)

func main() {
	// Configure slog default logger
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	// Initialize database (auto-migrate models)
	if err := database.Connect(); err != nil {
		slog.Error("database initialization failed", "error", err)
		os.Exit(1)
	}

	// Initialize Kafka writer from env
	var kafkaWriter *broker.KafkaWriter
	brokersEnv := os.Getenv("KAFKA_BROKERS")
	topic := os.Getenv("KAFKA_TOPIC")
	if brokersEnv != "" && topic != "" {
		brokers := strings.Split(brokersEnv, ",")
		kafkaWriter = broker.NewKafkaWriter(brokers, topic)
		slog.Info("kafka writer initialized", "brokers", brokers, "topic", topic)
	} else {
		slog.Info("kafka not configured; proceeding without broker")
	}

	// Initialize repository and handler
	repo := repository.New(database.DB, kafkaWriter)
	h := handler.NewHandler(repo)

	// Use handler's router which includes health and the ConnectRPC service
	h2cHandler := h.SetupRouter()

	// Read application port from env, fallback to 8081 to avoid conflict with Temporal dashboard on 8080
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8081"
	}
	addr := ":" + appPort

	srv := &http.Server{
		Addr:         addr,
		Handler:      h2cHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Ensure kafka writer is closed on exit if initialized
	defer func() {
		if kafkaWriter != nil {
			_ = kafkaWriter.Close()
		}
	}()

	// Log which port the server will listen on
	slog.Info("starting transaction service", "addr", srv.Addr)
	slog.Info("server listening", "port", appPort)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
