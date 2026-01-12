package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	v1 "FinTechPorto/gen/api/transaction/v1"
	transactionv1connect "FinTechPorto/gen/api/transaction/v1/transactionv1connect"

	"log/slog"

	connectgo "github.com/bufbuild/connect-go"
	"github.com/go-chi/chi/v5"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// transactionService implements transactionv1connect.TransactionServiceHandler
type transactionService struct {
}

func (s *transactionService) CreateTransfer(ctx context.Context, req *connectgo.Request[v1.CreateTransferRequest]) (*connectgo.Response[v1.CreateTransferResponse], error) {
	// Avoid unused parameter warning
	_ = ctx

	// Log incoming request
	slog.Info("CreateTransfer called",
		"sender_id", req.Msg.SenderId,
		"recipient_id", req.Msg.RecipientId,
		"amount", req.Msg.Amount,
		"currency", req.Msg.Currency,
		"memo", req.Msg.Memo,
	)

	// Build mock transaction
	now := timestamppb.Now()
	tx := &v1.Transaction{
		TransactionId: "tx_123",
		SenderId:      req.Msg.SenderId,
		RecipientId:   req.Msg.RecipientId,
		Amount:        req.Msg.Amount,
		Currency:      req.Msg.Currency,
		Status:        v1.TransactionStatus_PENDING,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if req.Msg.Memo != nil {
		m := *req.Msg.Memo
		tx.Memo = &m
	}

	resp := &v1.CreateTransferResponse{
		TransactionId: "tx_123",
		Status:        v1.TransactionStatus_PENDING,
		Transaction:   tx,
	}
	return connectgo.NewResponse(resp), nil
}

func (s *transactionService) GetTransactionStatus(ctx context.Context, req *connectgo.Request[v1.GetTransactionStatusRequest]) (*connectgo.Response[v1.GetTransactionStatusResponse], error) {
	// Avoid unused parameter warning
	_ = ctx

	// Return a mock COMPLETED status
	resp := &v1.GetTransactionStatusResponse{
		TransactionId: req.Msg.TransactionId,
		Status:        v1.TransactionStatus_COMPLETED,
	}
	return connectgo.NewResponse(resp), nil
}

func main() {
	// Configure slog default logger
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	r := chi.NewRouter()

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Register ConnectRPC handler
	svc := &transactionService{}
	path, handler := transactionv1connect.NewTransactionServiceHandler(svc)
	r.Handle(path, handler)

	// Wrap with H2C to allow HTTP/2 without TLS (local development)
	h := h2c.NewHandler(r, &http2.Server{})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      h,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	slog.Info("starting transaction service", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
