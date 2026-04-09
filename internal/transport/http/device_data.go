package http

import (
	"IOTProject/internal/domain"
	"IOTProject/internal/service"
	"IOTProject/pkg/error_code"
	"IOTProject/pkg/response"

	"github.com/gin-gonic/gin"
)

type DeviceDataHandle struct {
	svc service.DeviceDataService
}

func NewDeviceDataHandle(svc service.DeviceDataService) *DeviceDataHandle {
	return &DeviceDataHandle{
		svc: svc,
	}
}

// SaveDeviceData 对传感器数据进行入库
func (h *DeviceDataHandle) SaveDeviceData(c *gin.Context) {
	var req domain.DeviceData
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(error_code.InvalidParam)
		return
	}
	err := h.svc.SaveDeviceData(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, "ok")
}

// GetHistory 处理传感器历史数据的查询请求
func (h *DeviceDataHandle) GetHistory(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	var req domain.DataHistoryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		_ = c.Error(error_code.InvalidParam)
		return
	}

	column, ok := domain.AllowedSensorColumns[req.SensorType]
	if !ok {
		_ = c.Error(error_code.DeviceTypeFailError)
		return
	}
	uid, _ := userID.(int64)
	req.UserID = uid
	req.SensorType = column
	data, err := h.svc.GetHistory(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, data)
}
