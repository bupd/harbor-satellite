// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: satellites.sql

package database

import (
	"context"
	"time"
)

const createSatellite = `-- name: CreateSatellite :one
INSERT INTO satellites (name, token, created_at, updated_at)
VALUES ($1, $2, $3, $4)
RETURNING id, name, token, created_at, updated_at
`

type CreateSatelliteParams struct {
	Name      string
	Token     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (q *Queries) CreateSatellite(ctx context.Context, arg CreateSatelliteParams) (Satellite, error) {
	row := q.db.QueryRowContext(ctx, createSatellite,
		arg.Name,
		arg.Token,
		arg.CreatedAt,
		arg.UpdatedAt,
	)
	var i Satellite
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Token,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteSatellite = `-- name: DeleteSatellite :exec
DELETE FROM satellites
WHERE id = $1
`

func (q *Queries) DeleteSatellite(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, deleteSatellite, id)
	return err
}

const getSatelliteByName = `-- name: GetSatelliteByName :one
SELECT id, name, token, created_at, updated_at FROM satellites
WHERE name = $1 LIMIT 1
`

func (q *Queries) GetSatelliteByName(ctx context.Context, name string) (Satellite, error) {
	row := q.db.QueryRowContext(ctx, getSatelliteByName, name)
	var i Satellite
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Token,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getSatelliteByToken = `-- name: GetSatelliteByToken :one
SELECT id, name, token, created_at, updated_at FROM satellites
WHERE token = $1 LIMIT 1
`

func (q *Queries) GetSatelliteByToken(ctx context.Context, token string) (Satellite, error) {
	row := q.db.QueryRowContext(ctx, getSatelliteByToken, token)
	var i Satellite
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Token,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getSatelliteID = `-- name: GetSatelliteID :one
SELECT id FROM satellites
WHERE name = $1 LIMIT 1
`

func (q *Queries) GetSatelliteID(ctx context.Context, name string) (int32, error) {
	row := q.db.QueryRowContext(ctx, getSatelliteID, name)
	var id int32
	err := row.Scan(&id)
	return id, err
}

const listSatellites = `-- name: ListSatellites :many
SELECT id, name, token, created_at, updated_at FROM satellites
`

func (q *Queries) ListSatellites(ctx context.Context) ([]Satellite, error) {
	rows, err := q.db.QueryContext(ctx, listSatellites)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Satellite
	for rows.Next() {
		var i Satellite
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Token,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
