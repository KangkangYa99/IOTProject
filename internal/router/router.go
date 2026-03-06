package router

import (
	"IOTProject/internal/domain"
	"IOTProject/internal/transport/http"
	"IOTProject/internal/transport/middleware"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func InitRouter(
	userHandle *http.UserHandler,
	deviceHandle *http.DeviceHandle,
	devicedataHandle *http.DeviceDataHandle,
	userRepo domain.UserRepository,
) *gin.Engine {
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
	UserPublic := r.Group("/user")
	{
		UserPublic.POST("/register", userHandle.Register)
		UserPublic.POST("/adminregister", middleware.JWTAUTH(userRepo), userHandle.AdminCreateUser)
		UserPublic.POST("/login", userHandle.Login)
		UserPublic.POST("/Logout", userHandle.Logout)
	}
	UserAuth := r.Group("/user")
	UserAuth.Use(middleware.JWTAUTH(userRepo))
	{
		UserAuth.POST("/update", userHandle.UpdateProfile)
		UserAuth.GET("/info", userHandle.GetUserInfo)
		UserAuth.POST("/logout", userHandle.Logout)
	}

	DeviceAuth := r.Group("/device")
	DeviceAuth.Use(middleware.JWTAUTH(userRepo))
	{
		DeviceAuth.POST("/bindDevice", deviceHandle.BindDevice)
		DeviceAuth.POST("/unbindDevice", deviceHandle.UnBindDevice)
		DeviceAuth.GET("/getDevicesInfo", deviceHandle.GetDevicesInfo)
	}

	devicedataAuth := r.Group("/devicedata")
	devicedataAuth.Use(middleware.JWTAUTH(userRepo))
	{
		devicedataAuth.GET("/history", devicedataHandle.GetHistory)
	}
	return r
}
