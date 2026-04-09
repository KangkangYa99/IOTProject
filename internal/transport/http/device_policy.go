package http

import (
	"IOTProject/internal/domain"
	"IOTProject/internal/service"
	"IOTProject/internal/websocket"
	"IOTProject/pkg/error_code"
	"IOTProject/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DevicePolicyHandle struct {
	svc        service.DevicePolicyService
	deviceRepo domain.DeviceRepository
	hub        *websocket.Hub
}

func NewDevicePolicyHandler(svc service.DevicePolicyService,
	deviceRepo domain.DeviceRepository,
	hub *websocket.Hub,
) *DevicePolicyHandle {
	return &DevicePolicyHandle{
		svc:        svc,
		deviceRepo: deviceRepo,
		hub:        hub,
	}
}

// CreatePolicy 处理创建自动化规则的请求。
func (h *DevicePolicyHandle) CreatePolicy(c *gin.Context) {
	var req domain.CreatePolicyRequest
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
		_ = c.Error(error_code.ServerError)
		return
	}
	policyID, err := h.svc.CreatePolicy(c.Request.Context(), userID, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	h.hub.PublishEvent(websocket.WsEvent{
		Type:      "policy_create",
		UserID:    userID,
		DeviceUID: req.DeviceUID,
		ObjectID:  policyID,
		Data:      req,
	})
	response.Success(c, "策略创建成功")
}

// GetPoliciesByUID 查询指定设备的所有已配置策略。
func (h *DevicePolicyHandle) GetPoliciesByUID(c *gin.Context) {
	deviceUID := c.Query("device_uid")
	if deviceUID == "" {
		_ = c.Error(error_code.InvalidParam)
		return
	}
	val, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	userID := val.(int64)
	policies, err := h.svc.GetPoliciesByUID(c.Request.Context(), userID, deviceUID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, policies)
}

// DeletePolicy 删除指定的策略 ID。
func (h *DevicePolicyHandle) DeletePolicy(c *gin.Context) {
	val, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	userID := val.(int64)
	var req struct {
		PolicyID int64 `json:"PolicyId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(error_code.InvalidParam)
		return
	}
	err := h.svc.DeletePolicy(c.Request.Context(), userID, req.PolicyID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	h.hub.PublishEvent(websocket.WsEvent{
		Type:     "policy_delete",
		UserID:   userID,
		ObjectID: req.PolicyID,
	})
	response.Success(c, nil)
}

// GetUnreadCountHandler 获取当前用户未处理报警的总数。
func (h *DevicePolicyHandle) GetUnreadCountHandler(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	count, err := h.svc.GetUnreadAlertCount(c.Request.Context(), userID.(int64))
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, count)
}

// GetPendingAlertsHandler 获取当前用户待处理的报警列表。
func (h *DevicePolicyHandle) GetPendingAlertsHandler(c *gin.Context) {
	val, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	userID, ok := val.(int64)
	if !ok {
		return
	}
	logs, err := h.svc.GetPendingAlerts(c.Request.Context(), userID)
	if err != nil {
		return
	}
	response.Success(c, logs)
}

// MarkAlertHandledHandler 处理单个报警
func (h *DevicePolicyHandle) MarkAlertHandledHandler(c *gin.Context) {
	logIDParam := c.Param("log_id")
	logID, err := strconv.ParseInt(logIDParam, 10, 64)
	if err != nil {
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
	err = h.svc.MarkAlertHandled(c.Request.Context(), userID, logID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	h.hub.PublishEvent(websocket.WsEvent{
		Type:     "alert_handled",
		UserID:   userID,
		ObjectID: logID,
	})
	response.Success(c, nil)
}
