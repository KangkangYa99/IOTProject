package domain

import (
	"context"
	"time"
)

// DeviceData 描述传感器数据的实体模型
type DeviceData struct {
	DataID              uint      `json:"-" gorm:"primaryKey;autoIncrement;column:data_id"`
	DeviceUID           string    `json:"device_uid" gorm:"-"`
	DeviceID            int       `json:"-" gorm:"column:device_id"`
	Temperature         float64   `json:"temperature" gorm:"column:temperature"`
	Humidity            float64   `json:"humidity" gorm:"column:humidity"`
	LightIntensity      float64   `json:"light_intensity" gorm:"column:light_intensity"`
	NoiseLevel          float64   `json:"noise_level" gorm:"column:noise_level"`
	FlameDetected       bool      `json:"flame_detected" gorm:"column:flame_detected"`
	CarbonMonoxideLevel float64   `json:"carbon_monoxide_level" gorm:"column:carbon_monoxide_level"`
	FanOn               bool      `json:"fan_on" gorm:"column:fan_on"`
	LightOn             bool      `json:"light_on" gorm:"column:rgb_enable"`
	RGBRed              int       `json:"rgb_red" gorm:"column:rgb_red"`
	RGBGreen            int       `json:"rgb_green" gorm:"column:rgb_green"`
	RGBBlue             int       `json:"rgb_blue" gorm:"column:rgb_blue"`
	DataTimestamp       time.Time `json:"data_timestamp" gorm:"column:data_timestamp;default:now()"`
}

// AllowedSensorColumns 定义了前端请求的传感器类型字符串与数据库列名的合法映射关系，
var AllowedSensorColumns = map[string]string{
	"temp":  "temperature",           //温度
	"humi":  "humidity",              //湿度
	"light": "light_intensity",       //光照
	"noise": "noise_level",           //噪音
	"fire":  "flame_detected",        //火焰
	"Co":    "carbon_monoxide_level", //一氧化碳
}

// DataHistoryRequest 设备历史数据查询请求模型
type DataHistoryRequest struct {
	UserID     int64  `form:"-"`
	DeviceUID  string `form:"device_uid" binding:"required"`
	SensorType string `form:"sensor_type" binding:"required"`
	Limit      int    `form:"limit" binding:"min=1,max=100"`
}

// RawNumItem 用于从数据库扫描数值型传感器历史数据的临时结构体。
type RawNumItem struct {
	DataTimestamp time.Time
	Value         float64
}

// RawBoolItem 用于从数据库扫描布尔型传感器历史数据的临时结构体。
type RawBoolItem struct {
	DataTimestamp time.Time
	Value         bool
}

// SensorHistoryItem 标准化历史数据响应对象。
type SensorHistoryItem struct {
	DataTimestamp time.Time   `json:"data_timestamp"`
	Value         interface{} `json:"value"`
}
type MessagePusher interface {
	SendToUser(userID int64, message []byte)
	SendToDevice(deviceUID string, message []byte) error
}
type DeviceDataRepository interface {
	SaveSensorData(ctx context.Context, Data *DeviceData) error
	GetHistoryByDeviceID(ctx context.Context, req *DataHistoryRequest) ([]SensorHistoryItem, error)
}
