package handler

import (
	v1 "FinTechPorto/gen/api/transaction/v1"
	transactionv1connect "FinTechPorto/gen/api/transaction/v1/transactionv1connect"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"log/slog"

	connectgo "github.com/bufbuild/connect-go"
	"github.com/go-chi/chi/v5"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	_ "google.golang.org/protobuf/types/known/timestamppb"

	"FinTechPorto/services/transaction/repository"

	"go.temporal.io/sdk/client"

	"FinTechPorto/internal/workflow"
)

// transactionHandler implements transactionv1connect.TransactionServiceHandler
type transactionHandler struct {
	repo    *repository.Repository
	tclient client.Client
}

// NewHandler creates a new transactionHandler.
func NewHandler(repo *repository.Repository, tc client.Client) *transactionHandler {
	return &transactionHandler{repo: repo, tclient: tc}
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

	// Build workflow params
	params := workflow.TransferParams{
		SenderID:    req.Msg.SenderId,
		RecipientID: req.Msg.RecipientId,
		Amount:      req.Msg.Amount,
		Currency:    req.Msg.Currency,
		Memo:        memo,
	}

	// Start workflow asynchronously
	workflowID := "transfer-" + time.Now().Format("20060102-150405-000000")
	run, err := s.tclient.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "transaction-task-queue",
	}, workflow.TransferWorkflow, params)
	if err != nil {
		slog.Error("failed to start workflow", "error", err)
		return nil, connectgo.NewError(connectgo.CodeInternal, err)
	}

	// Return immediate response with workflow/run id and PENDING status
	resp := &v1.CreateTransferResponse{
		TransactionId: workflowID,
		Status:        v1.TransactionStatus_PENDING,
	}
	_ = run
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
	// register multiple path variants to ensure correct routing
	r.Handle(path, handler)
	// without trailing slash
	trimmed := strings.TrimRight(path, "/")
	r.Handle(trimmed, handler)
	// wildcard forms
	r.Handle(path+"*", handler)
	r.Handle(trimmed+"/*", handler)

	// Wrap with H2C
	return h2c.NewHandler(r, &http2.Server{})
}
