package service

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/error_code"
	"context"
)

type DevicePolicyService struct {
	repo       domain.DevicePolicyRepository
	deviceRepo domain.DeviceRepository
}

func NewDevicePolicyService(repo domain.DevicePolicyRepository, deviceRepo domain.DeviceRepository) *DevicePolicyService {
	return &DevicePolicyService{
		repo:       repo,
		deviceRepo: deviceRepo,
	}
}

// CreatePolicy 创建新的设备自动化策略。
func (s *DevicePolicyService) CreatePolicy(ctx context.Context, userID int64, req *domain.CreatePolicyRequest) (int64, error) {
	device, err := s.deviceRepo.GetDeviceInfoByUID(ctx, req.DeviceUID)
	if err != nil {
		return 0, err
	}
	if device.UserID == nil || *device.UserID != userID {
		return 0, error_code.NotDeviceOwner
	}
	exists, err := s.repo.Exists(ctx, domain.ExistsPolicyQuery{
		DeviceID:       device.DeviceID,
		SensorType:     req.SensorType,
		Operator:       req.Operator,
		ThresholdValue: req.ThresholdValue,
	})
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, error_code.DevicePolicyIsExist
	}
	policy := &domain.DevicePolicy{
		UserID:         userID,
		DeviceID:       device.DeviceID,
		SensorType:     req.SensorType,
		Operator:       req.Operator,
		ThresholdValue: req.ThresholdValue,
		ActionType:     req.ActionType,
		ActionTarget:   req.ActionTarget,
		ActionValue:    req.ActionValue,
		ActionMessage:  req.ActionMessage,
		Enabled:        true,
	}
	if err = s.repo.Create(ctx, policy); err != nil {
		return 0, err
	}
	return policy.PolicyID, nil
}

// GetDevicePolicies 获取设备所有策略。
func (s *DevicePolicyService) GetDevicePolicies(ctx context.Context, deviceID int64) ([]domain.DevicePolicy, error) {
	return s.repo.GetByDeviceID(ctx, deviceID)
}

// GetPoliciesByUID 基于用户身份获取特定设备的策略列表，包含所有权校验。
func (s *DevicePolicyService) GetPoliciesByUID(ctx context.Context, userID int64, deviceUID string) ([]domain.DevicePolicy, error) {
	device, err := s.deviceRepo.GetDeviceInfoByUID(ctx, deviceUID)
	if err != nil {
		return nil, err
	}
	if device.UserID == nil || *device.UserID != userID {
		return nil, error_code.NotDeviceOwner
	}
	return s.repo.GetByDeviceID(ctx, int64(device.DeviceID))
}

// DeletePolicy 删除设备策略，操作前进行权限校验以防止越权。
func (s *DevicePolicyService) DeletePolicy(ctx context.Context, userID int64, policyID int64) error {
	policy, err := s.repo.GetPolicyByID(ctx, policyID)
	if err != nil {
		return err
	}
	if policy.UserID != userID {
		return error_code.NotDeviceOwner
	}
	return s.repo.DeleteByID(ctx, policyID)
}

// GetUnreadAlertCount 获取用户当前未处理的报警数量。
func (s *DevicePolicyService) GetUnreadAlertCount(ctx context.Context, userID int64) (int64, error) {
	return s.repo.GetUnreadCount(ctx, userID)
}

// GetPendingAlerts 获取用户所有待处理报警记录。
func (s *DevicePolicyService) GetPendingAlerts(ctx context.Context, userID int64) ([]domain.AlertLog, error) {
	return s.repo.GetAlertHistory(ctx, userID)
}

// MarkAlertHandled 将报警设为已处理 (1)
func (s *DevicePolicyService) MarkAlertHandled(ctx context.Context, userID int64, logID int64) error {
	// 获取日志，检查是否存在及归属权
	log, err := s.repo.GetLogByID(ctx, logID)
	if err != nil {
		return err
	}
	if log.UserID != userID {
		return error_code.NotDeviceOwner
	}
	if log.Status == 1 {
		return nil
	}
	return s.repo.UpdateStatus(ctx, logID, 1)
}
