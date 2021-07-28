// Code generated by sqlc. DO NOT EDIT.
// source: operation.sql

package db

import (
	"context"
)

const createOperation = `-- name: CreateOperation :exec
INSERT INTO operations(id) VALUES ($1)
`

func (q *Queries) CreateOperation(ctx context.Context, id string) error {
	_, err := q.exec(ctx, q.createOperationStmt, createOperation, id)
	return err
}
