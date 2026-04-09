package postgres

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/error_code"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type DeviceRepository struct {
	gorm  *gorm.DB
	redis *redis.Client
}

func NewDeviceRepository(gorm *gorm.DB, redis *redis.Client) *DeviceRepository {
	return &DeviceRepository{
		gorm:  gorm,
		redis: redis,
	}
}

// BindDevice 执行原子的设备更新操作
func (d *DeviceRepository) BindDevice(ctx context.Context, BindInfo *domain.BindDeviceResp) error {
	var exists int64
	err := d.gorm.WithContext(ctx).Model(&domain.Device{}).
		Where("device_uid = ?", BindInfo.DeviceUID).
		Count(&exists).Error
	if err != nil {
		return fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	if exists == 0 {
		return error_code.DeviceNotFound
	}
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
	cacheKey := fmt.Sprintf("device:user:%s", BindInfo.DeviceUID)
	d.redis.Del(ctx, cacheKey)
	return nil
}

// UnbindDevice 执行数据库层面的解绑动作
// 根据设备 UID 和用户 ID 进行匹配更新，返回受影响的行数。
func (d *DeviceRepository) UnbindDevice(ctx context.Context, UnBindInfo *domain.UnbindDevice) (int64, error) {
	var rowsAffected int64
	err := d.gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var device domain.Device
		if err := tx.Where("device_uid = ? AND user_id = ?", UnBindInfo.DeviceUID, UnBindInfo.UserID).
			First(&device).Error; err != nil {
			return err
		}
		result := tx.Model(&domain.Device{}).
			Where("device_id = ?", device.DeviceID).
			Updates(map[string]interface{}{
				"user_id":     nil,
				"device_name": "",
			})
		if result.Error != nil {
			return result.Error
		}
		rowsAffected = result.RowsAffected
		if err := tx.Where("device_id = ? AND user_id = ?", device.DeviceID, UnBindInfo.UserID).
			Delete(&domain.DevicePolicy{}).Error; err != nil {
			return err
		}

		return nil
	})
	if err == nil && rowsAffected > 0 {
		cacheKey := fmt.Sprintf("device:user:%s", UnBindInfo.DeviceUID)
		d.redis.Del(ctx, cacheKey)
	}
	return rowsAffected, err
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
	cacheKey := fmt.Sprintf("device:user:%s", uid)
	val, err := d.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		userID, _ := strconv.ParseInt(val, 10, 64)
		if userID == -1 {
			return nil, error_code.DeviceNotBind
		}
		return &userID, nil
	}
	var device domain.Device
	err = d.gorm.WithContext(ctx).
		Select("user_id").
		Where("device_uid = ?", uid).
		First(&device).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			d.redis.Set(ctx, cacheKey, -1, 5*time.Minute)
			return nil, error_code.DeviceNotFound
		}
		return nil, fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	if device.UserID == nil {
		d.redis.Set(ctx, cacheKey, -1, 10*time.Minute)
		return nil, error_code.DeviceNotBind
	}
	d.redis.Set(ctx, cacheKey, *device.UserID, 24*time.Hour)
	return device.UserID, nil
}

// GetDeviceInfoByUID 根据设备唯一标识符 (UID) 查询完整的设备记录。
func (d *DeviceRepository) GetDeviceInfoByUID(ctx context.Context, uid string) (*domain.Device, error) {
	var device domain.Device
	err := d.gorm.WithContext(ctx).Where("device_uid = ?", uid).First(&device).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, error_code.DeviceNotFound
		}
		return nil, fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	return &device, nil
}

// UpdateDeviceName 更新指定设备的名称。
func (d *DeviceRepository) UpdateDeviceName(ctx context.Context, uid string, name string) error {
	result := d.gorm.WithContext(ctx).
		Model(&domain.Device{}).
		Where("device_uid = ?", uid).
		Update("device_name", name)
	if result.Error != nil {
		return fmt.Errorf("%w: %v", error_code.ErrDB, result.Error)
	}
	cacheKey := fmt.Sprintf("device:user:%s", uid)
	d.redis.Del(ctx, cacheKey)
	return nil
}

// GetUIDByID 根据设备自增 ID 获取其物理 UID
func (d *DeviceRepository) GetUIDByID(ctx context.Context, deviceID int64) (string, error) {
	var device domain.Device

	// 我们只想要 device_uid 这一列的数据
	err := d.gorm.WithContext(ctx).
		Model(&domain.Device{}).
		Select("device_uid").
		Where("device_id = ?", deviceID).
		First(&device).Error

	if err != nil {
		return "", err
	}
	return device.DeviceUID, nil
}

// UpdateDeviceStatus 根据设备情况记录上下线
func (d *DeviceRepository) UpdateDeviceStatus(ctx context.Context, uid string, status string) error {
	updates := map[string]interface{}{
		"device_status": status,
	}
	// 如果是上线，则更新最后在线时间
	if status == "online" {
		updates["last_online"] = time.Now()
	}

	// 使用 Table("devices") 直接指定表名，防止模型解析错误
	return d.gorm.WithContext(ctx).
		Table("devices").
		Where("device_uid = ?", uid).
		Updates(updates).Error
}
