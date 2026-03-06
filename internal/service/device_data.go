package service

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/error_code"
	"context"
)

type DeviceDataService struct {
	repo       domain.DeviceDataRepository
	deviceRepo domain.DeviceRepository
}

func NewDeviceDataService(
	repo domain.DeviceDataRepository,
	deviceRepo domain.DeviceRepository,
) *DeviceDataService {
	return &DeviceDataService{
		repo:       repo,
		deviceRepo: deviceRepo,
	}
}

// GetHistory 根据设备 UID 获取历史传感器数据
// 1. 权限校验：通过 deviceRepo 确认设备是否存在且归属于当前请求用户 (UserID)。
// 2. 参数清洗：对请求参数进行有效性检查，并对分页限制 (Limit) 设定默认值。
func (s *DeviceDataService) GetHistory(ctx context.Context, req *domain.DataHistoryRequest) ([]domain.DeviceData, error) {
	ownerID, err := s.deviceRepo.GetDeviceOwner(ctx, req.DeviceUID)
	if err != nil {
		return nil, error_code.DeviceNotFound
	}
	if ownerID == nil || *ownerID != req.UserID {
		return nil, error_code.NotDeviceOwner
	}
	if req.DeviceUID == "" {
		return nil, error_code.InvalidParam
	}
	if req.Limit <= 0 {
		req.Limit = 100
	}
	return s.repo.GetHistoryByDeviceID(ctx, req)
}
