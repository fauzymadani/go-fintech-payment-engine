package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"FinTechPorto/internal/broker"
	"FinTechPorto/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Activities holds dependencies for workflow activities.
type Activities struct {
	DB     *gorm.DB
	Broker *broker.KafkaWriter
	Topic  string
}

// TransferParams defines parameters for a transfer.
type TransferParams struct {
	SenderID    string
	RecipientID string
	Amount      int64
	Currency    string
	Memo        *string
}

// DebitAccountActivity subtracts amount from the sender's account.
func (a *Activities) DebitAccountActivity(ctx context.Context, p TransferParams) error {
	return a.DB.Transaction(func(tx *gorm.DB) error {
		var sender models.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND currency = ?", p.SenderID, p.Currency).First(&sender).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("sender not found")
			}
			return err
		}

		if sender.Balance < p.Amount {
			return errors.New("insufficient funds")
		}

		sender.Balance -= p.Amount
		if err := tx.Model(&models.Account{}).Where("id = ?", sender.ID).Update("balance", sender.Balance).Error; err != nil {
			return err
		}
		return nil
	})
}

// CreditAccountActivity adds amount to the recipient's account and creates a transaction record.
func (a *Activities) CreditAccountActivity(ctx context.Context, p TransferParams) (*models.Transaction, error) {
	var tr models.Transaction
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		var recipient models.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND currency = ?", p.RecipientID, p.Currency).First(&recipient).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("recipient not found")
			}
			return err
		}

		recipient.Balance += p.Amount
		if err := tx.Model(&models.Account{}).Where("id = ?", recipient.ID).Update("balance", recipient.Balance).Error; err != nil {
			return err
		}

		tr = models.Transaction{
			SenderID:    p.SenderID,
			RecipientID: p.RecipientID,
			Amount:      p.Amount,
			Currency:    p.Currency,
			Status:      "COMPLETED",
		}
		if p.Memo != nil {
			tr.Memo = *p.Memo
		}

		if err := tx.Create(&tr).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &tr, nil
}

// PublishKafkaEventActivity publishes a JSON event to the configured Kafka topic.
func (a *Activities) PublishKafkaEventActivity(ctx context.Context, event map[string]interface{}) error {
	if a.Broker == nil {
		slog.Info("broker not configured; skipping publish")
		return nil
	}

	b := a.Broker
	bts, err := json.Marshal(event)
	if err != nil {
		slog.Error("failed to marshal event", "error", err)
		return err
	}

	if err := b.PublishTransactionEvent(ctx, event); err != nil {
		slog.Error("failed to publish event", "error", err)
		return err
	}

	slog.Info("published event from activity", "topic", a.Topic, "payload", string(bts))
	return nil
}
