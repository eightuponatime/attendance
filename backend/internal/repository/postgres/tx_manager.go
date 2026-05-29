package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type txKeyType struct{}

var txKey = txKeyType{}

type TransactionManager struct {
	db *sqlx.DB
}

func NewTransactionManager(db *sqlx.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// WithTransaction -> wrap the function with transaction
// if @param fn returns error -> rollback
// if @param fn succeed -> commit transaction
func (m *TransactionManager) WithTransaction(
	ctx context.Context,
	fn func(context context.Context) error) error {
	transaction, err := m.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})

	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	contextWithTransaction := context.WithValue(ctx, txKey, transaction)

	err = fn(contextWithTransaction)
	if err != nil {
		rollbackError := transaction.Rollback()
		if rollbackError != nil {
			return fmt.Errorf(
				"rollback failed: %v, original error: %w", rollbackError, err)
		}
		return err
	}

	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction error: %w", err)
	}

	return nil
}

func extractTransaction(ctx context.Context, db *sqlx.DB) sqlx.ExtContext {
	transaction, ok := ctx.Value(txKey).(*sqlx.Tx)
	if ok {
		return transaction
	}
	return db
}
