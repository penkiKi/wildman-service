package database

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
)

type DB struct {
	*sql.DB
}

func (database *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return database.DB.ExecContext(ctx, rebind(query), args...)
}

func (database *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return database.DB.QueryContext(ctx, rebind(query), args...)
}

func (database *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return database.DB.QueryRowContext(ctx, rebind(query), args...)
}

func (database *DB) BeginTx(ctx context.Context, options *sql.TxOptions) (*Tx, error) {
	transaction, err := database.DB.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: transaction}, nil
}

type Tx struct {
	*sql.Tx
}

func (transaction *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return transaction.Tx.ExecContext(ctx, rebind(query), args...)
}

func (transaction *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return transaction.Tx.QueryContext(ctx, rebind(query), args...)
}

func (transaction *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return transaction.Tx.QueryRowContext(ctx, rebind(query), args...)
}

func rebind(query string) string {
	if !strings.Contains(query, "?") {
		return query
	}
	var builder strings.Builder
	parameter := 1
	for _, character := range query {
		if character == '?' {
			builder.WriteByte('$')
			builder.WriteString(strconv.Itoa(parameter))
			parameter++
		} else {
			builder.WriteRune(character)
		}
	}
	return builder.String()
}
