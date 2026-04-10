package websocket

import (
	"IOTProject/internal/domain"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type DeviceClient struct {
	DeviceUID string
	Conn      *websocket.Conn
	Hub       *Hub
	send      chan []byte
	Listener  domain.EventListener
}
type SignedData struct {
	domain.DeviceData
	Timestamp int64  `json:"timestamp"`
	Sign      string `json:"sign"`
}

func (d *DeviceClient) ReadPump() {
	const heartbeatTimeout = 60 * time.Second
	defer func() {
		log.Printf("设备 %s 正在注销并关闭连接...", d.DeviceUID)
		d.Hub.deviceUnreg <- d
		d.Conn.Close()
	}()
	d.Conn.SetReadLimit(1024)
	d.Conn.SetReadDeadline(time.Now().Add(heartbeatTimeout))
	for {
		messageType, message, err := d.Conn.ReadMessage()
		if err != nil {
			log.Printf("设备 %s 读取错误: %v", d.DeviceUID, err)
			break
		}
		d.Conn.SetReadDeadline(time.Now().Add(heartbeatTimeout))

		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err = json.Unmarshal(message, &msg); err == nil {
				t, _ := msg["type"].(string)
				if t == "heartbeat" {
					log.Printf("[HEARTBEAT] 设备 %s 活跃中", d.DeviceUID)
					go d.Listener.UpdateDeviceStatus(context.Background(), d.DeviceUID, "online")
					d.Hub.notifyDeviceStatus(d.DeviceUID, "on")
					ack := fmt.Sprintf(`{"type":"heartbeat_ack","device":"%s"}`, d.DeviceUID)
					d.Conn.WriteMessage(websocket.TextMessage, []byte(ack))
					continue
				}
				if t == "data" {
					var sd SignedData
					if err = json.Unmarshal(message, &sd); err != nil {
						log.Printf("解析带签名数据失败: %v", err)
						continue
					}
					sd.DeviceData.DeviceUID = d.DeviceUID
					const secret = "23wlw4IOT"
					rawStr := fmt.Sprintf("%s%.1f%.1f%d%s",
						d.DeviceUID,
						sd.Temperature,
						sd.Humidity,
						sd.Timestamp,
						secret,
					)
					expectedSign := fmt.Sprintf("%x", md5.Sum([]byte(rawStr)))
					if !strings.EqualFold(sd.Sign, expectedSign) {
						log.Printf("[SECURITY] 签名校验失败！疑似篡改数据。UID: %s", d.DeviceUID)
						continue
					}
					if time.Now().Unix()-sd.Timestamp > 60 {
						log.Printf("[SECURITY] 数据包已过期（重放攻击预防）。UID: %s", d.DeviceUID)
						continue
					}
					log.Printf(">>> 收到设备 %s 数据，准备入库", d.DeviceUID)
					var data domain.DeviceData

					data.DeviceUID = d.DeviceUID
					if d.Listener != nil {
						if err = d.Listener.SaveData(context.Background(), &data); err != nil {
							log.Printf("业务层处理数据失败: %v", err)
						}

					}
				}
			}
		}
	}
}
func (d *DeviceClient) WritePump() {
	defer func() {
		d.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-d.send:
			if !ok {
				d.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := d.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("设备 %s 发送消息失败: %v", d.DeviceUID, err)
				return
			}
		}
	}
}
