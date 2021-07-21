package storeutil

import (
	"context"
	"database/sql"
	"errors"
)

type ctxKeyType string

var ctxKeyTx = ctxKeyType("ctx-key-transaction")

// TxOptions allows the caller of CtxTxWraper.CtxWithTx to control various options to the transaction.
type TxOptions func(o *sql.TxOptions) *sql.TxOptions

// TxWithIsolation tells the DB driver the isolation level of the transaction.
func TxWithIsolation(level sql.IsolationLevel) func(o *sql.TxOptions) *sql.TxOptions {
	return func(o *sql.TxOptions) *sql.TxOptions {
		o.Isolation = level
		return o
	}
}

// TxReadonly signals the DB driver that the transaction is read-only.
func TxReadonly() func(o *sql.TxOptions) *sql.TxOptions {
	return func(o *sql.TxOptions) *sql.TxOptions {
		o.ReadOnly = true
		return o
	}
}

// CtxWithTx attach a database transaction to the context. It returns the
// context unchanged if there's error starting the transaction.
func CtxWithTx(ctx context.Context, db *sql.DB, opts ...TxOptions) (context.Context, error) {
	o := &sql.TxOptions{Isolation: sql.LevelSerializable}
	for _, opt := range opts {
		o = opt(o)
	}
	txn, err := db.BeginTx(ctx, o)
	if err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, ctxKeyTx, txn), nil
}

// FinishTxForCtx commits or rolls back the transaction attatched to the
// context depending on the error passed in and returns the result. It errors
// if the context doesn't have a transaction attached.
func FinishTxForCtx(ctx context.Context, err error) error {
	tx := ctx.Value(ctxKeyTx)
	txn, ok := tx.(*sql.Tx)
	if !ok {
		return errors.New("the context has no transcation attached")
	}
	if err != nil {
		// intentionlly ignore the error to avoid shadowing the real
		// error causing the rollback.
		_ = txn.Rollback()
	}
	return txn.Commit()
}

// WithTx runs the provided closure in a transaction. Then transaction is
// committed if the closure returns no error, and rolled back otherwise. When a
// transaction is already attached to the context, it is used instead and no
// automatical commit or rollback happens.
//nolint:unparam
func WithTx(ctx context.Context, db *sql.DB, f func(*sql.Tx) error, opts ...TxOptions) (err error) {
	// use the existing transaction if present
	tx := ctx.Value(ctxKeyTx)
	if txn, ok := tx.(*sql.Tx); ok {
		return f(txn)
	}

	o := &sql.TxOptions{Isolation: sql.LevelSerializable}
	for _, opt := range opts {
		o = opt(o)
	}
	var txn *sql.Tx
	txn, err = db.BeginTx(ctx, o)
	if err != nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			_ = txn.Rollback()
			panic(r)
		}
		if err != nil {
			// intentionlly ignore the error to avoid shadowing the
			// real error causing the rollback.
			_ = txn.Rollback()
		} else {
			err = txn.Commit()
		}
	}()
	err = f(txn)
	return
}

// UseTxFromCtx calls withTx if a transaction is attached to the context, or calls noTx.
func UseTxFromCtx(ctx context.Context, withTx func(*sql.Tx) error, noTx func() error) (err error) {
	tx := ctx.Value(ctxKeyTx)
	if txn, ok := tx.(*sql.Tx); ok {
		return withTx(txn)
	}
	return noTx()
}

// MustUseTxFromCtx calls withTx if a transaction is attached to the context.
func MustUseTxFromCtx(ctx context.Context, withTx func(*sql.Tx) error) (err error) {
	tx := ctx.Value(ctxKeyTx)
	if txn, ok := tx.(*sql.Tx); ok {
		return withTx(txn)
	}
	return errors.New("ctx doesn't contain a txn")
}
