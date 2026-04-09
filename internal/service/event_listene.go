package service

import (
	"IOTProject/internal/domain"
	"context"
)

type DeviceDataSvc interface {
	SaveDeviceData(ctx context.Context, data *domain.DeviceData) error
}

type DeviceEventHandler struct {
	// 1. 确保这里引用的是 Service 层，用于处理入库+推送逻辑
	dataSvc    DeviceDataSvc
	deviceRepo domain.DeviceRepository
}

func NewDeviceEventHandler(
	ds DeviceDataSvc, // 传入实现了 SaveDeviceData 的 Service 实例
	dr domain.DeviceRepository,
) *DeviceEventHandler {
	return &DeviceEventHandler{
		dataSvc:    ds,
		deviceRepo: dr,
	}
}

// SaveData 对应 EventListener 接口
func (h *DeviceEventHandler) SaveData(ctx context.Context, data *domain.DeviceData) error {
	return h.dataSvc.SaveDeviceData(ctx, data)
}

// GetDeviceOwner 对应 EventListener 接口
func (h *DeviceEventHandler) GetDeviceOwner(ctx context.Context, uid string) (*int64, error) {
	return h.deviceRepo.GetDeviceOwner(ctx, uid)
}

func (h *DeviceEventHandler) UpdateDeviceStatus(ctx context.Context, uid string, status string) error {
	return h.deviceRepo.UpdateDeviceStatus(ctx, uid, status)
}
