package domain

import "context"

type EventListener interface {
	SaveData(ctx context.Context, data *DeviceData) error
	GetDeviceOwner(ctx context.Context, uid string) (*int64, error)
	UpdateDeviceStatus(ctx context.Context, uid string, status string) error
}
