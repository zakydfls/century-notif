package repository

import (
	"context"

	"gorm.io/gorm"
)

// TxManager opens a single SQL transaction and hands the *gorm.DB (tx) to the callback.
// Inside the callback, services rebind each repo via repo.WithTrx(tx) so every operation
// runs in the same transaction. Commit on nil return, rollback on error.
type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

func (m *TxManager) Do(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}
