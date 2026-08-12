package postgres

import (
	"context"
	"fmt"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionManager runs callbacks inside a PostgreSQL transaction.
type TransactionManager struct {
	pool *pgxpool.Pool
}

// NewTransactionManager creates a TransactionManager.
func NewTransactionManager(pool *pgxpool.Pool) *TransactionManager {
	return &TransactionManager{pool: pool}
}

// WithinTransaction executes fn inside a transaction.
func (m *TransactionManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := TxFromContext(ctx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txCtx := ContextWithTx(ctx, tx)
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

var _ ports.TransactionManager = (*TransactionManager)(nil)
