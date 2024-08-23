// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: labels.sql

package database

import (
	"context"
	"time"
)

const createLabel = `-- name: CreateLabel :one
INSERT INTO labels (label_name, created_at, updated_at)
VALUES ($1, $2, $3)
RETURNING id, label_name, created_at, updated_at
`

type CreateLabelParams struct {
	LabelName string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (q *Queries) CreateLabel(ctx context.Context, arg CreateLabelParams) (Label, error) {
	row := q.db.QueryRowContext(ctx, createLabel, arg.LabelName, arg.CreatedAt, arg.UpdatedAt)
	var i Label
	err := row.Scan(
		&i.ID,
		&i.LabelName,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
