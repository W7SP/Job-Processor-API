package paint

import (
	"context"
	"errors"
)

type Paint struct {
	ID       string `json:"id"`
	Color    string `json:"color"`
	Prepared bool   `json:"prepared"`
}

var ErrNotFound = errors.New("not found")

type Store interface {
	Get(ctx context.Context, id string) (*Paint, error)
	GetAll(ctx context.Context) (map[string]*Paint, error)
	Create(ctx context.Context, color string) (*Paint, error)
	DeleteAll(ctx context.Context) error
}

type StatusUpdater interface {
	UpdateStatus(ctx context.Context, id string) error
}
