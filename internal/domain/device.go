package domain

import (
	"context"
	"time"
)

type Device struct {
	DeviceID     int64     `json:"device_id"`
	DeviceName   *string   `json:"device_name"`
	DeviceUID    string    `json:"device_uid"`
	UserID       int64     `json:"user_id"`
	DeviceStatus string    `json:"device_status"`
	LastOnline   time.Time `json:"last_online"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type BindDeviceResp struct {
	DeviceUID  string `json:"device_uid" binding:"required"`
	DeviceName string `json:"device_name" binding:"required"`
	UserID     int64  `json:"user_id"`
}
type UnbindDevice struct {
	DeviceUID string `json:"device_uid" binding:"required"`
	UserID    int64  `json:"-"`
}
type DeviceInfo struct {
	TotalCount int      `json:"total_count"`
	Devices    []Device `json:"devices"`
	Message    string   `json:"-"`
}
type DeviceInterface interface {
	BindDevice(ctx context.Context, BindInfo *BindDeviceResp) error
	UnbindDevice(ctx context.Context, DeleteInfo *UnbindDevice) error
	GetDeviceInfo(ctx context.Context, userID *int64) (*DeviceInfo, error)
}
