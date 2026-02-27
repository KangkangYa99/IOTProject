package postgres

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/error_code"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeviceRepository struct {
	db *pgxpool.Pool
}

func NewDeviceRepository(db *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{
		db: db,
	}
}

func (d *DeviceRepository) GetDeviceOwner(ctx context.Context, uid string) (*int64, error) {
	var userID *int64
	query := `SELECT user_id FROM devices WHERE device_uid = $1`
	err := d.db.QueryRow(ctx, query, uid).Scan(&userID)
	if err != nil {
		return nil, err
	}
	return userID, nil
}
func (d *DeviceRepository) BindDevice(ctx context.Context, BindInfo *domain.BindDeviceResp) error {
	ownerID, err := d.GetDeviceOwner(ctx, BindInfo.DeviceUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return error_code.DeviceNotFound
		}
		return fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	if ownerID != nil {
		return error_code.DeviceIsBind
	}

	query := `UPDATE devices SET user_id = $1, device_name = $2 WHERE device_uid = $3 AND user_id IS NULL`

	result, err := d.db.Exec(ctx, query, BindInfo.UserID, BindInfo.DeviceName, BindInfo.DeviceUID)
	if err != nil {
		return fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	if result.RowsAffected() == 0 {
		return error_code.DeviceIsBind
	}
	return nil
}
func (d *DeviceRepository) UnbindDevice(ctx context.Context, DeleteInfo *domain.UnbindDevice) error {
	query := `UPDATE devices SET user_id = NULL, device_name = '' WHERE device_uid = $1 AND user_id = $2`
	result, err := d.db.Exec(ctx, query, DeleteInfo.DeviceUID, DeleteInfo.UserID)
	if err != nil {
		return fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	if result.RowsAffected() == 0 {
		ownerID, err := d.GetDeviceOwner(ctx, DeleteInfo.DeviceUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return error_code.DeviceNotFound
			}
			return fmt.Errorf("%w: %v", error_code.ErrDB, err)
		}
		if ownerID == nil {
			return error_code.DeviceNotBind
		}
		return error_code.NotDeviceOwner
	}
	return nil
}
func (d *DeviceRepository) GetDeviceInfo(ctx context.Context, userID *int64) (*domain.DeviceInfo, error) {
	var info domain.DeviceInfo
	info.Devices = make([]domain.Device, 0)
	query := `
        SELECT device_id, device_name, device_uid, user_id, device_status, last_online, created_at, updated_at 
        FROM devices 
        WHERE user_id = $1 
        ORDER BY device_uid DESC`
	rows, err := d.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	defer rows.Close()
	for rows.Next() {
		var dev domain.Device
		err := rows.Scan(
			&dev.DeviceID, &dev.DeviceName, &dev.DeviceUID,
			&dev.UserID, &dev.DeviceStatus, &dev.LastOnline,
			&dev.CreatedAt, &dev.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		info.Devices = append(info.Devices, dev)
	}
	info.TotalCount = len(info.Devices)
	return &info, nil
}
