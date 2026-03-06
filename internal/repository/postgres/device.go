package postgres

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/error_code"
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type DeviceRepository struct {
	gorm *gorm.DB
}

func NewDeviceRepository(db2 *gorm.DB) *DeviceRepository {
	return &DeviceRepository{
		gorm: db2,
	}
}

// BindDevice 执行原子的设备更新操作
func (d *DeviceRepository) BindDevice(ctx context.Context, BindInfo *domain.BindDeviceResp) error {
	result := d.gorm.WithContext(ctx).
		Model(&domain.Device{}).
		Where("device_uid = ? AND user_id IS NULL", BindInfo.DeviceUID).
		Updates(map[string]interface{}{
			"user_id":     BindInfo.UserID,
			"device_name": BindInfo.DeviceName,
		})

	if result.Error != nil {
		return fmt.Errorf("%w: %v", error_code.ErrDB, result.Error)
	}
	if result.RowsAffected == 0 {
		return error_code.DeviceIsBind
	}
	return nil
}

// UnbindDevice 执行数据库层面的解绑动作
// 根据设备 UID 和用户 ID 进行匹配更新，返回受影响的行数。
func (d *DeviceRepository) UnbindDevice(ctx context.Context, req *domain.UnbindDevice) (int64, error) {
	result := d.gorm.WithContext(ctx).
		Model(&domain.Device{}).
		// 使用结构体中的字段进行匹配
		Where("device_uid = ? AND user_id = ?", req.DeviceUID, req.UserID).
		Select("UserID", "DeviceName").
		Updates(map[string]interface{}{
			"user_id":     nil,
			"device_name": "",
		})

	return result.RowsAffected, result.Error
}

// GetDeviceInfo 根据用户 ID 获取设备列表
// 自动处理结果集映射与总数统计，按照设备 UID 降序排列。
func (d *DeviceRepository) GetDeviceInfo(ctx context.Context, userID int64) (*domain.DeviceInfo, error) {
	var devices []domain.Device
	err := d.gorm.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("device_uid DESC").
		Find(&devices).Error

	if err != nil {
		return nil, fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	return &domain.DeviceInfo{
		TotalCount: len(devices),
		Devices:    devices,
	}, nil
}

// GetDeviceOwner 根据设备唯一标识查询所属用户 ID
func (d *DeviceRepository) GetDeviceOwner(ctx context.Context, uid string) (*int64, error) {
	var device domain.Device
	err := d.gorm.WithContext(ctx).
		Select("user_id").
		Where("device_uid = ?", uid).
		First(&device).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, error_code.DeviceNotFound
		}
		return nil, fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	return device.UserID, nil
}
