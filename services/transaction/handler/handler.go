package handler

import (
	v1 "FinTechPorto/gen/api/transaction/v1"
	transactionv1connect "FinTechPorto/gen/api/transaction/v1/transactionv1connect"
	"context"
	"errors"
	"net/http"

	"log/slog"

	connectgo "github.com/bufbuild/connect-go"
	"github.com/go-chi/chi/v5"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/types/known/timestamppb"

	"FinTechPorto/services/transaction/repository"
)

// transactionHandler implements transactionv1connect.TransactionServiceHandler
type transactionHandler struct {
	repo *repository.Repository
}

// NewHandler creates a new transactionHandler.
func NewHandler(repo *repository.Repository) *transactionHandler {
	return &transactionHandler{repo: repo}
}

func (s *transactionHandler) CreateTransfer(ctx context.Context, req *connectgo.Request[v1.CreateTransferRequest]) (*connectgo.Response[v1.CreateTransferResponse], error) {
	// Log incoming request
	slog.Info("CreateTransfer called",
		"sender_id", req.Msg.SenderId,
		"recipient_id", req.Msg.RecipientId,
		"amount", req.Msg.Amount,
		"currency", req.Msg.Currency,
		"memo", req.Msg.Memo,
	)

	memo := (*string)(nil)
	if req.Msg.Memo != nil {
		m := *req.Msg.Memo
		memo = &m
	}

	tr, err := s.repo.TransferFunds(ctx, req.Msg.SenderId, req.Msg.RecipientId, req.Msg.Amount, req.Msg.Currency, memo)
	if err != nil {
		// Map repository errors to connect errors
		if errors.Is(err, repository.ErrAccountNotFound) {
			return nil, connectgo.NewError(connectgo.CodeNotFound, err)
		}
		if errors.Is(err, repository.ErrInsufficientFunds) {
			return nil, connectgo.NewError(connectgo.CodeInvalidArgument, err)
		}
		return nil, connectgo.NewError(connectgo.CodeInternal, err)
	}

	now := timestamppb.Now()
	respTx := &v1.Transaction{
		TransactionId: tr.ID,
		SenderId:      tr.SenderID,
		RecipientId:   tr.RecipientID,
		Amount:        tr.Amount,
		Currency:      tr.Currency,
		Status:        v1.TransactionStatus_COMPLETED,
		CreatedAt:     now,
	}
	if tr.Memo != "" {
		m := tr.Memo
		respTx.Memo = &m
	}

	resp := &v1.CreateTransferResponse{
		TransactionId: tr.ID,
		Status:        v1.TransactionStatus_COMPLETED,
		Transaction:   respTx,
	}
	return connectgo.NewResponse(resp), nil
}

func (s *transactionHandler) GetTransactionStatus(ctx context.Context, req *connectgo.Request[v1.GetTransactionStatusRequest]) (*connectgo.Response[v1.GetTransactionStatusResponse], error) {
	// Fetch transaction
	tr, err := s.repo.GetTransactionByID(ctx, req.Msg.TransactionId)
	if err != nil {
		if errors.Is(err, repository.ErrAccountNotFound) {
			return nil, connectgo.NewError(connectgo.CodeNotFound, err)
		}
		return nil, connectgo.NewError(connectgo.CodeInternal, err)
	}

	// Map status string to proto enum
	status := v1.TransactionStatus_UNSPECIFIED
	switch tr.Status {
	case "PENDING":
		status = v1.TransactionStatus_PENDING
	case "COMPLETED":
		status = v1.TransactionStatus_COMPLETED
	case "FAILED":
		status = v1.TransactionStatus_FAILED
	case "REVERSED":
		status = v1.TransactionStatus_REVERSED
	}

	resp := &v1.GetTransactionStatusResponse{
		TransactionId: tr.ID,
		Status:        status,
	}
	return connectgo.NewResponse(resp), nil
}

// SetupRouter mounts the handler on a new chi Router and returns the router ready to be used.
func (s *transactionHandler) SetupRouter() http.Handler {
	r := chi.NewRouter()

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	path, handler := transactionv1connect.NewTransactionServiceHandler(s)
	r.Handle(path, handler)

	// Wrap with H2C
	return h2c.NewHandler(r, &http2.Server{})
}
