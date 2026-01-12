package repository

import (
	"context"
	"errors"
	"fmt"

	"FinTechPorto/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// ErrAccountNotFound is returned when an account cannot be found.
	ErrAccountNotFound = errors.New("account not found")
	// ErrInsufficientFunds is returned when sender has insufficient balance.
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// Repository wraps DB operations for transactions and accounts.
type Repository struct {
	db *gorm.DB
}

// New creates a new Repository.
func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// TransferFunds transfers amount from senderID to recipientID within a DB transaction.
// memo is optional and may be nil.
func (r *Repository) TransferFunds(ctx context.Context, senderID, recipientID string, amount int64, currency string, memo *string) (*models.Transaction, error) {
	var createdTx models.Transaction

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Load sender with row-level lock
		var sender models.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND currency = ?", senderID, currency).First(&sender).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAccountNotFound
			}
			return fmt.Errorf("failed to query sender: %w", err)
		}

		// validate balance
		if sender.Balance < amount {
			return ErrInsufficientFunds
		}

		// Load recipient and lock
		var recipient models.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND currency = ?", recipientID, currency).First(&recipient).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAccountNotFound
			}
			return fmt.Errorf("failed to query recipient: %w", err)
		}

		// update balances
		sender.Balance -= amount
		recipient.Balance += amount

		if err := tx.Model(&models.Account{}).Where("id = ?", sender.ID).Update("balance", sender.Balance).Error; err != nil {
			return fmt.Errorf("failed to update sender balance: %w", err)
		}
		if err := tx.Model(&models.Account{}).Where("id = ?", recipient.ID).Update("balance", recipient.Balance).Error; err != nil {
			return fmt.Errorf("failed to update recipient balance: %w", err)
		}

		// create transaction record
		tr := models.Transaction{
			SenderID:    sender.ID,
			RecipientID: recipient.ID,
			Amount:      amount,
			Currency:    currency,
			Status:      "COMPLETED",
		}
		if memo != nil {
			tr.Memo = *memo
		}

		if err := tx.Create(&tr).Error; err != nil {
			return fmt.Errorf("failed to create transaction record: %w", err)
		}

		createdTx = tr
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &createdTx, nil
}

// GetTransactionByID retrieves a transaction by its ID.
func (r *Repository) GetTransactionByID(ctx context.Context, id string) (*models.Transaction, error) {
	var tr models.Transaction
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&tr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &tr, nil
}
