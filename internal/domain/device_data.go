package domain

import (
	"context"
	"time"
)

// DeviceData 描述传感器数据的实体模型
type DeviceData struct {
	DataID              uint      `gorm:"primaryKey;column:data_id"`
	DeviceUID           string    `json:"device_uid" gorm:"-"`
	DeviceID            int       `gorm:"column:device_id"`
	Temperature         float64   `gorm:"column:temperature"`
	Humidity            float64   `gorm:"column:humidity"`
	LightIntensity      float64   `gorm:"column:light_intensity"`
	NoiseLevel          float64   `gorm:"column:noise_level"`
	FlameDetected       bool      `gorm:"column:flame_detected"`
	CarbonMonoxideLevel float64   `gorm:"column:carbon_monoxide_level"`
	LightOn             bool      `gorm:"column:light_on"`
	FanOn               bool      `gorm:"column:fan_on"`
	RGBRed              int       `gorm:"column:rgb_red"`
	RGBGreen            int       `gorm:"column:rgb_green"`
	RGBBlue             int       `gorm:"column:rgb_blue"`
	DataTimestamp       time.Time `gorm:"column:data_timestamp;default:now()"`
}

func (DeviceData) TableName() string {
	return "device_data"
}

// DataHistoryRequest 设备历史数据查询请求模型
type DataHistoryRequest struct {
	UserID    int64  `form:"-"`
	DeviceUID string `form:"device_uid" binding:"required"`
	Limit     int    `form:"limit" binding:"min=1,max=100"`
}

type DeviceDataRepository interface {
	SaveSensorData(ctx context.Context, Data *DeviceData) error
	GetHistoryByDeviceID(ctx context.Context, req *DataHistoryRequest) ([]DeviceData, error)
}
