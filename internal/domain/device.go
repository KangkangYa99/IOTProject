package domain

import (
	"context"
	"time"
)

// Device 数据库对应的设备实体模型
// 包含硬件唯一标识、归属用户、状态以及最后在线时间。
type Device struct {
	DeviceID     int64     `gorm:"primaryKey;column:device_id" json:"device_id"`
	DeviceName   *string   `gorm:"column:device_name" json:"device_name"`
	DeviceUID    string    `gorm:"column:device_uid;uniqueIndex" json:"device_uid"`
	UserID       *int64    `gorm:"column:user_id" json:"user_id"`
	DeviceStatus string    `gorm:"column:device_status" json:"device_status"`
	LastOnline   time.Time `gorm:"column:last_online" json:"last_online"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// BindDeviceResp 设备绑定请求/响应 DTO
// 用于绑定操作时的数据载体。虽然叫 Resp，但在你的 Service 中承担了 Request 的角色。
type BindDeviceResp struct {
	DeviceUID  string `json:"device_uid" binding:"required"`
	DeviceName string `json:"device_name" binding:"required"`
	UserID     int64  `json:"user_id"`
}

// UnbindDevice 设备解绑请求 DTO
// 仅需提供设备 UID，UserID 通过 Context 安全注入。
type UnbindDevice struct {
	DeviceUID string `json:"device_uid" binding:"required"`
	UserID    int64  `json:"-"`
}

// DeviceInfo 设备列表响应封装
// 包含总数统计与具体的设备对象列表。
type DeviceInfo struct {
	TotalCount int      `json:"total_count"`
	Devices    []Device `json:"devices"`
	Message    string   `json:"-"` // 内部消息，不序列化到 JSON
}
type DeviceRepository interface {
	BindDevice(ctx context.Context, BindInfo *BindDeviceResp) error
	UnbindDevice(ctx context.Context, req *UnbindDevice) (int64, error)
	GetDeviceInfo(ctx context.Context, userID int64) (*DeviceInfo, error)
	GetDeviceOwner(ctx context.Context, uid string) (*int64, error)
}
