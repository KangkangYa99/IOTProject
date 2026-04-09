package postgres

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/error_code"
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

// SaveSensorData 根据 device_uid 保存传感器数据
func (r *DeviceDataRepository) SaveSensorData(ctx context.Context, data *domain.DeviceData) error {
	if data.DeviceID == 0 && data.DeviceUID != "" {
		var deviceID int
		err := r.db.Table("devices").
			Where("device_uid = ?", data.DeviceUID).
			Select("device_id").
			Scan(&deviceID).Error
		if err != nil {
			return err
		}
		if deviceID == 0 {
			return error_code.DeviceNotFound
		}
		data.DeviceID = deviceID
	}
	return r.db.WithContext(ctx).Save(data).Error
}

// GetHistoryByDeviceID 根据设备 UID 获取历史传感器数据列表
func (r *DeviceDataRepository) GetHistoryByDeviceID(ctx context.Context, req *domain.DataHistoryRequest) ([]domain.SensorHistoryItem, error) {
	isBoolType := (req.SensorType == "flame_detected" || req.SensorType == "fan_on" || req.SensorType == "rgb_enable")
	var logs []domain.SensorHistoryItem
	if isBoolType {
		var raw []domain.RawBoolItem
		err := r.db.WithContext(ctx).Table("device_data").
			Select("data_timestamp, "+req.SensorType+" as value").
			Joins("JOIN devices ON devices.device_id = device_data.device_id").
			Where("devices.device_uid = ?", req.DeviceUID).
			Order("device_data.data_timestamp DESC").
			Limit(req.Limit).Scan(&raw).Error
		if err != nil {
			return nil, err
		}
		for _, item := range raw {
			val := 0.0
			if item.Value {
				val = 1.0
			}
			logs = append(logs, domain.SensorHistoryItem{DataTimestamp: item.DataTimestamp, Value: val})
		}
	} else {
		var raw []domain.RawNumItem
		err := r.db.WithContext(ctx).Table("device_data").
			Select("data_timestamp, "+req.SensorType+" as value").
			Joins("JOIN devices ON devices.device_id = device_data.device_id").
			Where("devices.device_uid = ?", req.DeviceUID).
			Order("device_data.data_timestamp DESC").
			Limit(req.Limit).Scan(&raw).Error
		if err != nil {
			return nil, err
		}
		for _, item := range raw {
			logs = append(logs, domain.SensorHistoryItem{DataTimestamp: item.DataTimestamp, Value: item.Value})
		}
	}
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}

	return logs, nil
}
