package bike

import (
	"context"
	"errors"
)

type Bike struct {
	ID          string `json:"id"`
	BikeType    string `json:"bike_type"`
	BuildStatus string `json:"build_status"`
}

var ErrNotFound = errors.New("not found")

type Store interface {
	Get(ctx context.Context, id string) (*Bike, error)
	GetAll(ctx context.Context) (map[string]*Bike, error)
	Create(ctx context.Context, bikeType string) (*Bike, error)
	DeleteAll(ctx context.Context) error
}

type StatusUpdater interface {
	UpdateStatus(ctx context.Context, id string, status string) error
}
