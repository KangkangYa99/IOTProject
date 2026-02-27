package router

import (
	"IOTProject/internal/domain"
	"IOTProject/internal/transport/http"
	"IOTProject/internal/transport/middleware"

	"github.com/gin-gonic/gin"
)

func InitRouter(
	userHandle *http.UserHandler,
	deviceHandle *http.DeviceHandle,
	userRepo domain.UserInterface,
) *gin.Engine {
	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(middleware.ErrorHandler())

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

	return r
}
