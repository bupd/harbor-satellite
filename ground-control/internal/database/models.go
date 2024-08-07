// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package database

import (
	"time"
)

type Group struct {
	ID        int32
	GroupName string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type GroupImage struct {
	GroupID int32
	ImageID int32
}

type Image struct {
	ID         int32
	Registry   string
	Repository string
	Tag        string
	Digest     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Label struct {
	ID        int32
	LabelName string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LabelImage struct {
	LabelID int32
	ImageID int32
}

type Satellite struct {
	ID        int32
	Name      string
	Token     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SatelliteGroup struct {
	SatelliteID int32
	GroupID     int32
}

type SatelliteLabel struct {
	SatelliteID int32
	LabelID     int32
}
