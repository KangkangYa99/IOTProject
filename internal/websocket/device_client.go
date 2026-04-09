package websocket

import (
	"IOTProject/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"log"
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
					log.Printf(">>> 收到设备 %s 数据，准备入库", d.DeviceUID)
					var data domain.DeviceData
					if err = json.Unmarshal(message, &data); err != nil {
						log.Printf("解析设备数据失败: %v", err)
						continue
					}
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
