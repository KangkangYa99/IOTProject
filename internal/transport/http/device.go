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
func (h *DeviceHandle) BindDevice(c *gin.Context) {
	var req domain.BindDeviceResp
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(error_code.InvalidParam)
		return
	}
	userID, exists := c.Get("userID")

	if exists {
		req.UserID = userID.(int64)
	}

	err := h.svc.BindDevice(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, nil)
}
func (h *DeviceHandle) UnBindDevice(c *gin.Context) {
	var req domain.UnbindDevice
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(error_code.InvalidParam)
		return
	}
	userID, exists := c.Get("userID")
	if exists {
		req.UserID = userID.(int64)
	}
	err := h.svc.UnBindDevice(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, nil)
}
func (h *DeviceHandle) GetDevicesInfo(c *gin.Context) {
	val, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	userID := val.(int64)
	info, err := h.svc.GetDeviceInfo(c.Request.Context(), &userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, info)
}
