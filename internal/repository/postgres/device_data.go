package postgres

import (
	"IOTProject/internal/domain"
	"context"

	"gorm.io/gorm"
)

type DeviceDataRepository struct {
	db *gorm.DB
}

func NewDeviceDataRepository(db *gorm.DB) *DeviceDataRepository {
	return &DeviceDataRepository{
		db: db,
	}
}

func (r *DeviceDataRepository) SaveSensorData(ctx context.Context, data *domain.DeviceData) error {
	return r.db.WithContext(ctx).Create(data).Error
}

// GetHistoryByDeviceID 根据设备 UID 获取历史传感器数据列表
func (r *DeviceDataRepository) GetHistoryByDeviceID(ctx context.Context, req *domain.DataHistoryRequest) ([]domain.DeviceData, error) {
	var logs []domain.DeviceData
	err := r.db.WithContext(ctx).
		Joins("JOIN devices ON devices.device_id = device_data.device_id").
		Where("devices.device_uid = ?", req.DeviceUID).
		Order("device_data.data_timestamp DESC").
		Limit(req.Limit).
		Find(&logs).Error
	return logs, err
}
