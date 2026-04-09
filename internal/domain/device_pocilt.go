package domain

import (
	"context"
	"time"
)

// DevicePolicy 数据库对应的设备策略实体模型
type DevicePolicy struct {
	PolicyID       int64     `gorm:"column:policy_id;primaryKey;autoIncrement"`
	UserID         int64     `gorm:"column:user_id"`
	DeviceID       int64     `gorm:"column:device_id"`
	SensorType     string    `gorm:"column:sensor_type"`
	Operator       string    `gorm:"column:operator"`
	ThresholdValue float64   `gorm:"column:threshold_value"`
	ActionType     string    `gorm:"column:action_type"`
	ActionTarget   string    `gorm:"column:action_target"`
	ActionValue    string    `gorm:"column:action_value"`
	ActionMessage  string    `gorm:"column:action_message"`
	Enabled        bool      `gorm:"column:enabled"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

// 策略行为类型常量定义
const (
	ActionAlert   = "alert"   // 仅报警
	ActionControl = "control" // 仅执行传感器
	ActionBoth    = "both"    // 报警并执行传感器
)

// CreatePolicyRequest 创建策略请求
type CreatePolicyRequest struct {
	DeviceUID      string  `json:"device_uid" binding:"required"`
	SensorType     string  `json:"sensor_type" binding:"required"`
	Operator       string  `json:"operator" binding:"required"`
	ThresholdValue float64 `json:"threshold_value"`
	ActionType     string  `json:"action_type" binding:"required"`
	ActionTarget   string  `json:"action_target"`
	ActionValue    string  `json:"action_value"`
	ActionMessage  string  `json:"action_message"`
}

// AlertLog 历史策略模型
type AlertLog struct {
	LogID      int64     `gorm:"column:log_id;primaryKey;autoIncrement"`
	UserID     int64     `gorm:"column:user_id"`
	DeviceID   int64     `gorm:"column:device_id"`
	PolicyID   int64     `gorm:"column:policy_id"`
	SensorType string    `gorm:"column:sensor_type"`
	CurrentVal float64   `gorm:"column:current_val"`
	Threshold  float64   `gorm:"column:threshold"`
	Message    string    `gorm:"column:message"`
	Status     int8      `gorm:"column:status;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// ExistsPolicyQuery 用于 Exists 检查
type ExistsPolicyQuery struct {
	DeviceID       int64
	SensorType     string
	Operator       string
	ThresholdValue float64
}

// AlertHistoryQuery 用于历史记录查询

// TableName 指定数据表名。
func (AlertLog) TableName() string {
	return "alert_logs"
}
func (DevicePolicy) TableName() string {
	return "device_policies"
}

type DevicePolicyRepository interface {
	Create(ctx context.Context, policy *DevicePolicy) error
	GetByDeviceID(ctx context.Context, deviceID int64) ([]DevicePolicy, error)
	Exists(ctx context.Context, query ExistsPolicyQuery) (bool, error)
	DeleteByID(ctx context.Context, policyID int64) error
	GetPolicyByID(ctx context.Context, policyID int64) (*DevicePolicy, error)
	Save(ctx context.Context, log *AlertLog) error
	GetUnreadCount(ctx context.Context, userID int64) (int64, error)
	GetAlertHistory(ctx context.Context, userID int64) ([]AlertLog, error)
	UpdateStatus(ctx context.Context, logID int64, status int8) error
	GetLogByID(ctx context.Context, logID int64) (*AlertLog, error)
}
