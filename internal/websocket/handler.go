package websocket

import (
	"IOTProject/internal/domain"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // 生产环境记得改回域名检查
}

type WsHandler struct {
	hub      *Hub
	listener domain.EventListener
}

func NewWsHandler(hub *Hub, listener domain.EventListener) *WsHandler {
	return &WsHandler{
		hub:      hub,
		listener: listener,
	}
}

func (h *WsHandler) ServeHTTP(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	tempID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(10000))
	log.Printf("[WS CONNECT] 新连接接入: %s, RemoteAddr: %s", tempID, conn.RemoteAddr().String())
	client := &Client{
		ID:     tempID,
		UserID: 0,
		Conn:   conn,
		Hub:    h.hub,
		send:   make(chan []byte, 256),
		isAuth: false,
	}
	go client.WritePump()
	go client.ReadPump()
}
func (h *WsHandler) ServeDeviceHTTP(c *gin.Context) {
	deviceUID := c.Query("device_uid")
	if deviceUID == "" {
		log.Println("[WS DEVICE] 连接失败: 缺少 device_uid")
		c.JSON(400, gin.H{"error": "device_uid is required"})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS DEVICE] 升级失败: %v", err)
		return
	}
	if tcpConn, ok := conn.UnderlyingConn().(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(3 * time.Minute)
	}
	device := &DeviceClient{
		DeviceUID: deviceUID,
		Conn:      conn,
		Hub:       h.hub,
		send:      make(chan []byte, 256),
		Listener:  h.listener,
	}
	h.hub.deviceReg <- device
	go device.WritePump()
	go device.ReadPump()
	log.Printf("[WS DEVICE] 设备 %s 已连接, 地址: %s", deviceUID, conn.RemoteAddr().String())
}
