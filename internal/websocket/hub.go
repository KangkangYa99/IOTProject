package websocket

import (
	"IOTProject/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

type WsEvent struct {
	Type         string      `json:"type"`
	UserID       int64       `json:"user_id"`
	DeviceUID    string      `json:"device_uid"`
	ObjectID     int64       `json:"object_id,omitempty"`
	ObjectString string      `json:"object_string,omitempty"`
	Data         interface{} `json:"data"`
}

type PushTask struct {
	UserID  int64
	Message []byte
}

func (h *Hub) SetEventListener(l domain.EventListener) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.listener = l
}

type Hub struct {
	clients     map[int64]map[string]*Client
	devices     map[string]*DeviceClient
	mu          sync.RWMutex
	listener    domain.EventListener
	register    chan *Client
	unregister  chan *Client
	deviceReg   chan *DeviceClient
	deviceUnreg chan *DeviceClient
	pushQueue   chan PushTask
	eventQueue  chan WsEvent
	broadcast   chan []byte
}

func NewHub(listener domain.EventListener) *Hub {
	return &Hub{
		clients:     make(map[int64]map[string]*Client),
		devices:     make(map[string]*DeviceClient),
		listener:    listener,
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		deviceReg:   make(chan *DeviceClient),
		deviceUnreg: make(chan *DeviceClient),
		pushQueue:   make(chan PushTask, 10000),
		eventQueue:  make(chan WsEvent, 1000),
		broadcast:   make(chan []byte, 1024),
	}
}

func (h *Hub) Run() {
	for {
		select {
		// --- 设备管理 ---
		case device := <-h.deviceReg:
			h.handleDeviceRegister(device)
		case device := <-h.deviceUnreg:
			h.handleDeviceUnregister(device)

		// --- 客户端管理 ---
		case client := <-h.register:
			h.handleClientRegister(client)
		case client := <-h.unregister:
			h.handleClientUnregister(client)

		// --- 消息分发 ---
		case event := <-h.eventQueue:
			h.broadcastToUser(event.UserID, event)
		case task := <-h.pushQueue:
			h.pushToUser(task.UserID, task.Message)
		case msg := <-h.broadcast:
			h.broadcastAll(msg)
		}
	}
}
func (h *Hub) handleDeviceRegister(device *DeviceClient) {
	h.mu.Lock()
	h.devices[device.DeviceUID] = device
	h.mu.Unlock()
	log.Printf("[HUB] 设备上线: %s", device.DeviceUID)
	go h.listener.UpdateDeviceStatus(context.Background(), device.DeviceUID, "online")
	h.notifyDeviceStatus(device.DeviceUID, "on")
}

func (h *Hub) handleDeviceUnregister(device *DeviceClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if curr, ok := h.devices[device.DeviceUID]; ok && curr == device {
		delete(h.devices, device.DeviceUID)
		log.Printf("[HUB] 设备真正下线: %s", device.DeviceUID)
		go h.listener.UpdateDeviceStatus(context.Background(), device.DeviceUID, "offline")
		h.notifyDeviceStatus(device.DeviceUID, "off")
	} else {
		log.Printf("[HUB] 忽略旧连接的下线请求: %s (当前已有新连接)", device.DeviceUID)
	}
}

func (h *Hub) handleClientRegister(client *Client) {
	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[string]*Client)
	}
	h.clients[client.UserID][client.ID] = client
}

func (h *Hub) handleClientUnregister(client *Client) {
	if !client.registered {
		return
	}
	if userClients, ok := h.clients[client.UserID]; ok {
		delete(userClients, client.ID)
		close(client.send)
		client.registered = false
		if len(userClients) == 0 {
			delete(h.clients, client.UserID)
		}
	}
}

// notifyDeviceStatus 异步查询并通知前端
func (h *Hub) notifyDeviceStatus(uid string, status string) {
	go func() {
		// 1. 查出谁拥有这个设备
		userIDPtr, err := h.listener.GetDeviceOwner(context.Background(), uid)
		if err != nil || userIDPtr == nil {
			return
		}
		event := WsEvent{
			Type:      "device_status",
			UserID:    *userIDPtr,
			DeviceUID: uid,
			Data: map[string]interface{}{
				"online": status,
			},
		}
		h.PublishEvent(event)
	}()
}

// broadcastToUser 推送结构化事件
func (h *Hub) broadcastToUser(userID int64, event WsEvent) {
	msg, _ := json.Marshal(event)
	h.pushToUser(userID, msg)
}

// pushToUser 推送原始字节消息
func (h *Hub) pushToUser(userID int64, msg []byte) {
	if userClients, ok := h.clients[userID]; ok {
		for _, client := range userClients {
			select {
			case client.send <- msg:
			default: // 防止慢客户端阻塞 Hub
				log.Printf("[WARN] 客户端 %s 消息积压，丢弃消息", client.ID)
			}
		}
	}
}

func (h *Hub) broadcastAll(msg []byte) {
	for userID := range h.clients {
		h.pushToUser(userID, msg)
	}
}

func (h *Hub) PublishEvent(e WsEvent) {
	select {
	case h.eventQueue <- e:
	default:
	}
}

func (h *Hub) SendToUser(u int64, m []byte) {
	h.pushQueue <- PushTask{UserID: u, Message: m}
}

// SendToDevice 实现 domain.MessagePusher 接口，用于控制硬件
func (h *Hub) SendToDevice(deviceUID string, message []byte) error {
	h.mu.RLock()
	device, ok := h.devices[deviceUID]
	h.mu.RUnlock()
	if !ok {
		return fmt.Errorf("设备 %s 不在线或未注册", deviceUID)
	}
	select {
	case device.send <- message:
		return nil
	default:
		return fmt.Errorf("设备 %s 消息管道积压", deviceUID)
	}
}
