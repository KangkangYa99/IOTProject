package http

import (
	"IOTProject/internal/domain"
	"IOTProject/internal/service"
	"IOTProject/pkg/error_code"
	"IOTProject/pkg/response"

	"github.com/gin-gonic/gin"
)

type DeviceHandle struct {
	svc service.DeviceService
}

func NewDeviceHandle(svc service.DeviceService) *DeviceHandle {
	return &DeviceHandle{
		svc: svc,
	}
}

// BindDevice 处理设备绑定接口
func (h *DeviceHandle) BindDevice(c *gin.Context) {
	var req domain.BindDeviceResp
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(error_code.InvalidParam.WithDetails(err.Error()))
		return
	}
	val, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	userID, ok := val.(int64)
	if !ok {
		return
	}
	req.UserID = userID

	if err := h.svc.BindDevice(c.Request.Context(), &req); err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, nil)
}

// UnBindDevice 处理设备解绑接口
func (h *DeviceHandle) UnBindDevice(c *gin.Context) {
	var req domain.UnbindDevice
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(error_code.InvalidParam)
		return
	}
	val, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	userID, ok := val.(int64)
	if !ok {
		return
	}
	req.UserID = userID
	err := h.svc.UnBindDevice(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, nil)
}

// GetDevicesInfo 获取当前登录用户的所有设备列表
func (h *DeviceHandle) GetDevicesInfo(c *gin.Context) {
	val, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	userID, ok := val.(int64)
	if !ok {
		return
	}
	info, err := h.svc.GetDeviceInfo(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, info)
}
