// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: group_images.sql

package database

import (
	"context"
	"time"
)

const assignImageToGroup = `-- name: AssignImageToGroup :exec
INSERT INTO group_images (group_id, image_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`

type AssignImageToGroupParams struct {
	GroupID int32
	ImageID int32
}

func (q *Queries) AssignImageToGroup(ctx context.Context, arg AssignImageToGroupParams) error {
	_, err := q.db.ExecContext(ctx, assignImageToGroup, arg.GroupID, arg.ImageID)
	return err
}

const getImagesForGroup = `-- name: GetImagesForGroup :many
SELECT id, registry, repository, tag, digest, created_at, updated_at, group_id, image_id
FROM images
JOIN group_images ON images.id = group_images.image_id
WHERE group_images.group_id = $1
`

type GetImagesForGroupRow struct {
	ID         int32
	Registry   string
	Repository string
	Tag        string
	Digest     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	GroupID    int32
	ImageID    int32
}

func (q *Queries) GetImagesForGroup(ctx context.Context, groupID int32) ([]GetImagesForGroupRow, error) {
	rows, err := q.db.QueryContext(ctx, getImagesForGroup, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetImagesForGroupRow
	for rows.Next() {
		var i GetImagesForGroupRow
		if err := rows.Scan(
			&i.ID,
			&i.Registry,
			&i.Repository,
			&i.Tag,
			&i.Digest,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.GroupID,
			&i.ImageID,
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
