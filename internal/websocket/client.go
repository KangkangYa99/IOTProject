package websocket

import (
	"IOTProject/pkg/utils"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID         string
	UserID     int64
	Conn       *websocket.Conn
	Hub        *Hub
	send       chan []byte
	isAuth     bool
	registered bool
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
func (c *Client) handleAuth(token string) {
	claims, err := utils.ParseToken(token)
	if err != nil {
		c.send <- []byte(`{"type":"error", "message":"鉴权失败"}`)
		return
	}
	if c.isAuth {
		c.Hub.unregister <- c
	}
	c.UserID = claims.UserID
	c.isAuth = true
	c.registered = true
	c.Hub.register <- c
	c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.send <- []byte(`{"type":"auth_success"}`)
}
func (c *Client) handleBusiness(message []byte) {
	c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))

	// 1. 定义控制指令结构（扩展支持三色灯）
	var ctrl struct {
		Type      string `json:"type"`
		DeviceUID string `json:"device_uid"`
		// 风扇字段
		Fan string `json:"fan"`
		// 三色灯字段
		Light string `json:"light"`
		R     int    `json:"r"`
		G     int    `json:"g"`
		B     int    `json:"b"`
	}

	if err := json.Unmarshal(message, &ctrl); err != nil {
		log.Printf("业务消息解析失败: %v", err)
		return
	}
	if ctrl.Type == "heartbeat" {
		return
	}
	if ctrl.Type == "control" {
		c.Hub.mu.RLock()
		device, ok := c.Hub.devices[ctrl.DeviceUID]
		c.Hub.mu.RUnlock()

		if !ok {
			c.send <- []byte(fmt.Sprintf(`{"type":"error","message":"设备 %s 不在线"}`, ctrl.DeviceUID))
			return
		}

		var deviceMsg string

		// 优先判断 Light，如果 JSON 里有 light 字段且不为空
		if ctrl.Light != "" {
			deviceMsg = fmt.Sprintf(`{"type":"light_ctrl","state":"%s","r":%d,"g":%d,"b":%d}`,
				ctrl.Light, ctrl.R, ctrl.G, ctrl.B)
			log.Printf("[CONTROL] 用户 %d -> 设备 %s: 灯光 %s (R:%d G:%d B:%d)",
				c.UserID, ctrl.DeviceUID, ctrl.Light, ctrl.R, ctrl.G, ctrl.B)
		} else if ctrl.Fan != "" { // 如果没有 light，再看有没有 fan
			deviceMsg = fmt.Sprintf(`{"type":"fan_ctrl","state":"%s"}`, ctrl.Fan)
			log.Printf("[CONTROL] 用户 %d -> 设备 %s: 风扇 %s", c.UserID, ctrl.DeviceUID, ctrl.Fan)
		}

		if deviceMsg != "" {
			device.send <- []byte(deviceMsg)
		}
	}
}

func (c *Client) handleGetVerifyCode(message []byte) {
	var codeReq struct {
		DeviceID string `json:"device_id"`
		Action   string `json:"action"`
	}
	if err := json.Unmarshal(message, &codeReq); err != nil {
		c.send <- []byte(`{"type":"error", "message":"请求格式错误"}`)
		return
	}
	if !c.isAuth {
		c.Conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	}
	id, b64s, _, err := utils.GenerateCaptchaImage()
	if err != nil {
		c.send <- []byte(`{"type":"error", "message":"验证码生成失败"}`)
		return
	}
	resp := fmt.Sprintf(`{"type":"verify_code", "image":"%s", "captcha_id":"%s"}`, b64s, id)
	c.send <- []byte(resp)
}
func (c *Client) dispatch(message []byte) {
	var msg struct {
		Type  string `json:"type"`
		Token string `json:"token,omitempty"`
	}
	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}

	switch msg.Type {
	case "get_verify_code":
		c.handleGetVerifyCode(message)
		return
	case "auth":
		c.handleAuth(msg.Token)
		return
	}

	if !c.isAuth {
		c.send <- []byte(`{"error":"Unauthorized"}`)
		return
	}

	// 处理业务逻辑，传入原始 message 字节数组
	c.handleBusiness(message)
}
func (c *Client) ReadPump() {
	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	defer func() {
		log.Printf("[WS DISCONNECT] 客户端 %s (User:%d) 断开连接", c.ID, c.UserID)
		if c.isAuth {
			c.Hub.unregister <- c
		}
		c.Conn.Close()
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		log.Printf("[WS RECV] 客户端 %s (User:%d): %s", c.ID, c.UserID, string(message))
		c.dispatch(message)
	}
}
