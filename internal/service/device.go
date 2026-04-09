package service

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/error_code"
	"context"
	"errors"

	"gorm.io/gorm"
)

type DeviceService struct {
	repo domain.DeviceRepository
}

func NewDeviceService(repo domain.DeviceRepository) *DeviceService {
	return &DeviceService{
		repo: repo,
	}
}
func (s *DeviceService) BindDevice(ctx context.Context, req *domain.BindDeviceResp) error {
	//查看设备拥有者
	//Info, err := s.repo.GetDeviceInfoByUID(ctx, req.DeviceUID)
	//if err != nil {
	//	if errors.Is(err, gorm.ErrRecordNotFound) {
	//		return error_code.DeviceNotFound
	//	}
	//	return err
	//}
	//设备被其他人绑定
	//if Info.UserID != nil {
	//	log.Printf("[DEBUG] 设备 %s 已被绑定, 拥有者 ID: %d", req.DeviceUID, Info.UserID)
	//	return error_code.DeviceIsBind
	//}
	//return s.repo.BindDevice(ctx, req)

	err := s.repo.BindDevice(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

// UnBindDevice 执行设备解绑逻辑
// 采用“先校验、后执行”的策略。
func (s *DeviceService) UnBindDevice(ctx context.Context, req *domain.UnbindDevice) error {
	//判断设备归属
	ownerID, err := s.repo.GetDeviceOwner(ctx, req.DeviceUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return error_code.DeviceNotFound
		}
		return err
	}
	if ownerID == nil {
		return error_code.DeviceNotBind
	}
	if *ownerID != req.UserID {
		return error_code.NotDeviceOwner
	}
	rows, err := s.repo.UnbindDevice(ctx, req)
	if err != nil {
		return err
	}
	if rows == 0 {
		return error_code.NotDeviceOwner
	}
	return nil
}

// GetDeviceInfo 获取指定用户的设备资产列表
func (s *DeviceService) GetDeviceInfo(ctx context.Context, userID int64) (*domain.DeviceInfo, error) {
	return s.repo.GetDeviceInfo(ctx, userID)
}

// UpdateDeviceName 更新指定设备的名称。
func (s *DeviceService) UpdateDeviceName(ctx context.Context, req *domain.UpdateDeviceNameRequest) error {
	device, err := s.repo.GetDeviceInfoByUID(ctx, req.DeviceUID)
	if err != nil {
		return err
	}
	if device.UserID == nil || *device.UserID != req.UserID {
		return error_code.NotDeviceOwner
	}
	return s.repo.UpdateDeviceName(ctx, req.DeviceUID, req.DeviceName)
}
