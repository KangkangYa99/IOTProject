package service

import (
	"IOTProject/internal/domain"
	"context"
)

type DeviceService struct {
	repo domain.DeviceInterface
}

func NewDeviceService(repo domain.DeviceInterface) *DeviceService {
	return &DeviceService{
		repo: repo,
	}
}
func (s *DeviceService) BindDevice(ctx context.Context, DeviceInfo *domain.BindDeviceResp) error {
	err := s.repo.BindDevice(ctx, DeviceInfo)
	if err != nil {
		return err
	}
	return nil
}

func (s *DeviceService) UnBindDevice(ctx context.Context, DeviceInfo *domain.UnbindDevice) error {
	err := s.repo.UnbindDevice(ctx, DeviceInfo)
	if err != nil {
		return err
	}
	return nil
}
func (s *DeviceService) GetDeviceInfo(ctx context.Context, userID *int64) (*domain.DeviceInfo, error) {
	return s.repo.GetDeviceInfo(ctx, userID)
}
