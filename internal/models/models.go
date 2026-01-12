package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Account represents a wallet account with a UUID primary key.
type Account struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	UserID    string `gorm:"index;not null"`
	Balance   int64  `gorm:"not null"`
	Currency  string `gorm:"size:3;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BeforeCreate hook to set a UUID when creating an Account.
func (a *Account) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// Transaction represents a transfer between two accounts.
type Transaction struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	SenderID    string    `gorm:"index;not null"`
	RecipientID string    `gorm:"index;not null"`
	Amount      int64     `gorm:"not null"`
	Currency    string    `gorm:"size:3;not null"`
	Status      string    `gorm:"size:32;not null"`
	Memo        string    `gorm:"size:1024"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

// BeforeCreate hook to set a UUID when creating a Transaction.
func (t *Transaction) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}
