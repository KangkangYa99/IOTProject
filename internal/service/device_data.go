package service

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/error_code"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type DeviceDataService struct {
	repo             domain.DeviceDataRepository
	deviceRepo       domain.DeviceRepository
	devicePolicyRepo domain.DevicePolicyRepository
	hub              domain.MessagePusher
	redis            *redis.Client
}

func NewDeviceDataService(
	repo domain.DeviceDataRepository,
	deviceRepo domain.DeviceRepository,
	devicePolicyRepo domain.DevicePolicyRepository,
	hub domain.MessagePusher,
	redis *redis.Client,
) *DeviceDataService {
	return &DeviceDataService{
		repo:             repo,
		deviceRepo:       deviceRepo,
		devicePolicyRepo: devicePolicyRepo,
		hub:              hub,
		redis:            redis,
	}
}

// SaveDeviceData 处理设备上报数据：入库
func (s *DeviceDataService) SaveDeviceData(ctx context.Context, data *domain.DeviceData) error {
	if data.DeviceUID == "" {
		return error_code.DeviceNotFound
	}
	if err := s.repo.SaveSensorData(ctx, data); err != nil {
		return err
	}
	go func() {
		ctx = context.Background()
		if err := s.ExecutePolicyEngine(ctx, data); err != nil {
			log.Printf("策略执行失败: %v", err)
		}
	}()
	userIDPtr, err := s.deviceRepo.GetDeviceOwner(ctx, data.DeviceUID)
	if err != nil {
		log.Printf("数据入库成功，但未找到设备 %s 的拥有者: %v", data.DeviceUID, err)
		return nil
	}
	if userIDPtr == nil {
		return nil
	}
	userID := *userIDPtr
	msg, _ := json.Marshal(map[string]interface{}{
		"type": "device_update",
		"data": data,
	})
	s.hub.SendToUser(userID, msg)
	return nil
}

// GetHistory 根据设备 UID 获取历史传感器数据
// 1. 权限校验：通过 deviceRepo 确认设备是否存在且归属于当前请求用户 (UserID)。
// 2. 参数清洗：对请求参数进行有效性检查，并对分页限制 (Limit) 设定默认值。
func (s *DeviceDataService) GetHistory(ctx context.Context, req *domain.DataHistoryRequest) ([]domain.SensorHistoryItem, error) {
	ownerID, err := s.deviceRepo.GetDeviceOwner(ctx, req.DeviceUID)
	if err != nil {
		return nil, error_code.DeviceNotFound
	}
	if ownerID == nil || *ownerID != req.UserID {
		return nil, error_code.NotDeviceOwner
	}
	if req.DeviceUID == "" {
		return nil, error_code.InvalidParam
	}
	if req.Limit <= 0 {
		req.Limit = 100
	}
	return s.repo.GetHistoryByDeviceID(ctx, req)
}

// ExecutePolicyEngine 核心策略引擎：根据设备最新数据遍历所有匹配策略并执行。
func (s *DeviceDataService) ExecutePolicyEngine(ctx context.Context, data *domain.DeviceData) error {
	device, err := s.deviceRepo.GetDeviceInfoByUID(ctx, data.DeviceUID)
	if err != nil {
		log.Printf("[ERROR] 获取设备失败: %v", err)
		return err
	}
	if device == nil {
		log.Printf("[ERROR] 设备对象为空")
		return error_code.DeviceNotFound
	}
	if device.UserID == nil {
		log.Printf("[DEBUG] 设备 %s 已解绑，跳过策略执行", data.DeviceUID)
	}
	policies, err := s.devicePolicyRepo.GetByDeviceID(ctx, device.DeviceID)
	if err != nil {
		log.Printf("[ERROR] 获取策略失败: %v", err)
		return err
	}

	for _, policy := range policies {
		val, exists := s.extractValue(data, policy.SensorType)
		if !exists {
			log.Printf("[DEBUG] 设备 %s 无可用策略", data.DeviceUID)
			continue
		}
		if s.evaluate(val, policy.Operator, policy.ThresholdValue) {
			log.Printf("[DEBUG] 策略触发！设备UID: %s, 传感器: %s, 当前值: %.2f, 操作符: %s, 阈值: %.2f",
				data.DeviceUID,
				policy.SensorType,
				val,
				policy.Operator,
				policy.ThresholdValue,
			)
			log.Printf("[DEBUG] 执行动作类型: %s, 目标: %s, 动作值: %s",
				policy.ActionType,
				policy.ActionTarget,
				policy.ActionValue,
			)
			go s.handleAction(policy, val)
		}
	}
	return nil
}

// handleAction 根据策略类型分发报警或控制指令。
func (s *DeviceDataService) handleAction(policy domain.DevicePolicy, currentVal float64) {
	actionType := policy.ActionType
	if (actionType == domain.ActionAlert || actionType == domain.ActionBoth) && policy.ActionMessage != "" {
		s.triggerAlert(policy, currentVal)
	}
	if actionType == domain.ActionControl || actionType == domain.ActionBoth {
		s.triggerControl(policy)
	}
}

// triggerAlert 执行报警逻辑：Redis 防抖 -> 入库
func (s *DeviceDataService) triggerAlert(policy domain.DevicePolicy, val float64) {
	key := fmt.Sprintf("alert_cooldown:policy:%d", policy.PolicyID)
	success, err := s.redis.SetNX(context.Background(), key, "1", 6*time.Second).Result()
	if err != nil {
		return
	}
	if !success {
		log.Printf("[DEBUG] 策略 %d 处于冷却期，已过滤此次重复报警", policy.PolicyID)
		return
	}
	go func() {
		alertLog := &domain.AlertLog{
			UserID:     policy.UserID,
			DeviceID:   policy.DeviceID,
			PolicyID:   policy.PolicyID,
			SensorType: policy.SensorType,
			CurrentVal: val,
			Threshold:  policy.ThresholdValue,
			Message:    policy.ActionMessage,
			Status:     0,
		}
		if err = s.devicePolicyRepo.Save(context.Background(), alertLog); err != nil {
			log.Printf("[ERROR] 报警入库失败: %v", err)
			return
		}
		alarmData := map[string]interface{}{
			"log_id":      alertLog.LogID,
			"policy_id":   policy.PolicyID,
			"device_id":   policy.DeviceID,
			"sensor_type": policy.SensorType,
			"current_val": val,
			"threshold":   policy.ThresholdValue,
			"message":     policy.ActionMessage,
			"timestamp":   time.Now().Format(time.RFC3339),
		}
		pushData := map[string]interface{}{
			"type": "alarm",
			"data": alarmData,
		}
		msgBytes, _ := json.Marshal(pushData)
		s.hub.SendToUser(policy.UserID, msgBytes)
		log.Printf("[ACTION] 报警入库并推送成功: LogID %d", alertLog.LogID)
	}()
}

// extractValue 辅助方法：通过反射或反射式枚举从 Data 中提取特定传感器数值。
func (s *DeviceDataService) extractValue(data *domain.DeviceData, sensorType string) (float64, bool) {
	switch sensorType {
	case "temperature":
		return data.Temperature, true
	case "humidity":
		return data.Humidity, true
	case "light_intensity":
		return data.LightIntensity, true
	case "noise_level":
		return data.NoiseLevel, true
	case "carbon_monoxide_level":
		return data.CarbonMonoxideLevel, true
	default:
		return 0, false
	}
}

// triggerControl 根据策略触发设备控制动作
func (s *DeviceDataService) triggerControl(policy domain.DevicePolicy) {
	// 1. Redis 冷却（防抖）：6秒内同一策略不重复触发，保护继电器
	key := fmt.Sprintf("control_cooldown:policy:%d", policy.PolicyID)
	success, err := s.redis.SetNX(context.Background(), key, "1", 6*time.Second).Result()
	if err != nil || !success {
		return
	}

	// 2. 构造下发给硬件的 JSON 指令
	var pushMsg map[string]interface{}
	switch policy.ActionTarget {
	case "fan":
		pushMsg = map[string]interface{}{
			"type":  "fan_ctrl",
			"state": policy.ActionValue,
		}
	case "light":
		pushMsg = map[string]interface{}{
			"type":  "light_ctrl",
			"state": policy.ActionValue,
			"r":     255,
			"g":     0,
			"b":     0,
		}
	default:
		log.Printf("[WARN] 未知的控制目标: %s，跳过下发", policy.ActionTarget)
		return
	}

	// 3. 序列化指令
	msgBytes, err := json.Marshal(pushMsg)
	if err != nil {
		log.Printf("[ERROR] JSON 序列化失败: %v", err)
		return
	}

	// 4. 获取设备 UID 并下发
	deviceUID, err := s.deviceRepo.GetUIDByID(context.Background(), policy.DeviceID)
	if err != nil {
		log.Printf("[ERROR] 策略 %d 触发失败：无法获取设备 UID, err: %v", policy.PolicyID, err)
		return
	}
	// 5. 通过 WebSocket 发送指令
	err = s.hub.SendToDevice(deviceUID, msgBytes)
	if err != nil {
		return
	}
	log.Printf("[ACTION] 策略触发成功！设备UID: %s, 目标: %s, 指令: %s",
		deviceUID, policy.ActionTarget, policy.ActionValue)
}

// evaluate 逻辑评估函数：计算传感器数值与策略阈值。
func (s *DeviceDataService) evaluate(val float64, op string, threshold float64) bool {
	switch op {
	case ">":
		return val > threshold
	case "<":
		return val < threshold
	case ">=":
		return val >= threshold
	case "<=":
		return val <= threshold
	case "==":
		return val == threshold
	default:
		return false
	}
}
