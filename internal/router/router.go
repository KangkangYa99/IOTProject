package router

import (
	"IOTProject/internal/domain"
	"IOTProject/internal/transport/http"
	"IOTProject/internal/transport/middleware"
	"IOTProject/internal/websocket"
	"IOTProject/pkg/utils"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type HandlerConfig struct {
	UserHandle         *http.UserHandler
	DeviceHandle       *http.DeviceHandle
	DeviceDataHandle   *http.DeviceDataHandle
	DevicePolicyHandle *http.DevicePolicyHandle
	UserRepo           domain.UserRepository
	WsHandler          *websocket.WsHandler
}

func InitRouter(cfg HandlerConfig) *gin.Engine {
	r := gin.Default()

	r.Use(gin.Recovery())
	r.Use(middleware.ErrorHandler())
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))
	r.Static("/static", "./static")
	r.SetTrustedProxies([]string{"127.0.0.1"})

	loginLimit := utils.RateLimitMiddleware(utils.RateLimitConfig{
		KeyPrefix: "login",
		Limit:     5,
		Time:      1 * time.Minute,
	})

	// --- 1. 用户路由 (User) ---
	UserPublic := r.Group("/user")
	{
		UserPublic.POST("/register", cfg.UserHandle.Register)
		UserPublic.POST("/resetpassword", cfg.UserHandle.ResetPasswordHandler)
		UserPublic.POST("/login", loginLimit, cfg.UserHandle.Login)
		UserPublic.POST("/Logout", cfg.UserHandle.Logout)
		UserPublic.GET("/GetCaptcha", cfg.UserHandle.GetCaptcha)
	}
	UserAuth := r.Group("/user")
	UserAuth.Use(middleware.JWTAUTH(cfg.UserRepo))
	{
		UserAuth.POST("/update", cfg.UserHandle.UpdateProfile)
		UserAuth.POST("/changepassword", cfg.UserHandle.ChangePassword)
		UserAuth.GET("/info", cfg.UserHandle.GetUserInfo)
		UserAuth.POST("/logout", cfg.UserHandle.Logout)
		UserAuth.POST("/uploadavatar", cfg.UserHandle.UploadAvatar)
	}

	// --- 2. 设备路由 (Device) ---
	DeviceAuth := r.Group("/device")
	DeviceAuth.Use(middleware.JWTAUTH(cfg.UserRepo))
	{
		DeviceAuth.POST("/bindDevice", cfg.DeviceHandle.BindDevice)
		DeviceAuth.POST("/unbindDevice", cfg.DeviceHandle.UnBindDevice)
		DeviceAuth.POST("/updatedevicename", cfg.DeviceHandle.UpdateDeviceNameHandler)
		DeviceAuth.GET("/getDevicesInfo", cfg.DeviceHandle.GetDevicesInfo)
	}

	// --- 3. 设备数据路由 (DeviceData) ---
	devicedataPublic := r.Group("/devicedata")
	{
		devicedataPublic.POST("/uploaddata", cfg.DeviceDataHandle.SaveDeviceData)
	}
	devicedataAuth := r.Group("/devicedata")
	devicedataAuth.Use(middleware.JWTAUTH(cfg.UserRepo))
	{
		devicedataAuth.GET("/history", cfg.DeviceDataHandle.GetHistory)
	}

	// --- 4. 策略报警路由 (DevicePolicy) ---
	devicepolicyAuth := r.Group("/devicepolicy")
	devicepolicyAuth.Use(middleware.JWTAUTH(cfg.UserRepo))
	{
		devicepolicyAuth.POST("/createpolicy", cfg.DevicePolicyHandle.CreatePolicy)
		devicepolicyAuth.GET("/getepolicy", cfg.DevicePolicyHandle.GetPoliciesByUID)
		devicepolicyAuth.POST("/deleteepolicy", cfg.DevicePolicyHandle.DeletePolicy)
		devicepolicyAuth.GET("/getunreadcount", cfg.DevicePolicyHandle.GetUnreadCountHandler)
		devicepolicyAuth.GET("/getpendingaler", cfg.DevicePolicyHandle.GetPendingAlertsHandler)
		devicepolicyAuth.PUT("/markaler/:log_id", cfg.DevicePolicyHandle.MarkAlertHandledHandler)
	}

	// --- 5. WebSocket 路由 ---
	r.GET("/ws", cfg.WsHandler.ServeHTTP)
	r.GET("/ws/device", cfg.WsHandler.ServeDeviceHTTP)
	return r
}
