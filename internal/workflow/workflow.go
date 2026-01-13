package workflow

import (
	"time"

	"FinTechPorto/internal/models"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TransferWorkflow orchestrates debit, credit, and publish activities.
func TransferWorkflow(ctx workflow.Context, params TransferParams) error {
	// Configure activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Execute Debit
	if err := workflow.ExecuteActivity(ctx, "DebitAccountActivity", params).Get(ctx, nil); err != nil {
		return err
	}

	// Execute Credit and retrieve transaction
	var tr models.Transaction
	if err := workflow.ExecuteActivity(ctx, "CreditAccountActivity", params).Get(ctx, &tr); err != nil {
		return err
	}

	// Prepare event
	event := map[string]interface{}{
		"transaction_id": tr.ID,
		"sender_id":      tr.SenderID,
		"recipient_id":   tr.RecipientID,
		"amount":         tr.Amount,
		"status":         tr.Status,
	}

	// Publish
	if err := workflow.ExecuteActivity(ctx, "PublishKafkaEventActivity", event).Get(ctx, nil); err != nil {
		return err
	}

	return nil
}
