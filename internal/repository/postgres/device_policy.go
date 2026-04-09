package postgres

import (
	"IOTProject/internal/domain"
	"context"

	"gorm.io/gorm"
)

type DevicePolicyRepository struct {
	db *gorm.DB
}

func NewDevicePolicyRepository(db *gorm.DB) domain.DevicePolicyRepository {
	return &DevicePolicyRepository{
		db: db,
	}
}

// Create 将一条新的设备自动化策略写入数据库。
func (r *DevicePolicyRepository) Create(ctx context.Context, policy *domain.DevicePolicy) error {
	return r.db.WithContext(ctx).Create(policy).Error
}

// GetByDeviceID 获取指定设备 ID 下所有处于启用状态的策略列表。
func (r *DevicePolicyRepository) GetByDeviceID(ctx context.Context, deviceID int64) ([]domain.DevicePolicy, error) {
	var policies []domain.DevicePolicy
	err := r.db.
		WithContext(ctx).
		Where("device_id = ? AND enabled = true", deviceID).
		Find(&policies).Error
	return policies, err
}

// Exists 查询新增设备策略是否存在
func (r *DevicePolicyRepository) Exists(ctx context.Context, query domain.ExistsPolicyQuery) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.DevicePolicy{}).
		Where("device_id = ? AND sensor_type = ? AND operator = ? AND threshold_value = ?",
			query.DeviceID, query.SensorType, query.Operator, query.ThresholdValue).
		Count(&count).Error
	return count > 0, err
}

// GetPolicyByID 根据策略 ID 查询单条策略详情。
func (r *DevicePolicyRepository) GetPolicyByID(ctx context.Context, policyID int64) (*domain.DevicePolicy, error) {
	var policy domain.DevicePolicy
	err := r.db.WithContext(ctx).First(&policy, policyID).Error
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

// DeleteByID 根据策略 ID 删除对应的自动化记录。
func (r *DevicePolicyRepository) DeleteByID(ctx context.Context, policyID int64) error {
	return r.db.WithContext(ctx).Delete(&domain.DevicePolicy{}, policyID).Error
}

// Save 将触发的报警事件记录到 alert_logs 表中。
func (r *DevicePolicyRepository) Save(ctx context.Context, log *domain.AlertLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetUnreadCount 查询未读报警总数
func (r *DevicePolicyRepository) GetUnreadCount(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.AlertLog{}).
		Where("user_id = ? AND status = 0", userID).
		Count(&count).Error
	return count, err
}

// GetAlertHistory 获取该用户最近的未处理报警列表，按创建时间倒序。
func (r *DevicePolicyRepository) GetAlertHistory(ctx context.Context, userID int64) ([]domain.AlertLog, error) {
	var logs []domain.AlertLog
	err := r.db.WithContext(ctx).
		Model(&domain.AlertLog{}).
		Where("user_id = ? AND status = 0", userID).
		Order("created_at DESC").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// UpdateStatus 修改报警日志的状态（如将“未读”更新为“已读”）。
func (r *DevicePolicyRepository) UpdateStatus(ctx context.Context, logID int64, status int8) error {
	return r.db.WithContext(ctx).
		Model(&domain.AlertLog{}).
		Where("log_id = ?", logID).
		Update("status", status).Error
}

// GetLogByID 根据日志主键获取报警详情。
func (r *DevicePolicyRepository) GetLogByID(ctx context.Context, logID int64) (*domain.AlertLog, error) {
	var log domain.AlertLog
	// 使用 First 查询，若未找到会返回 gorm.ErrRecordNotFound
	err := r.db.WithContext(ctx).
		Model(&domain.AlertLog{}).
		Where("log_id = ?", logID).
		First(&log).Error

	if err != nil {
		return nil, err
	}
	return &log, nil
}
