package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txKey struct{}

// ContextWithTx stores a pgx transaction in the context.
func ContextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFromContext returns the transaction from context when present.
func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}
