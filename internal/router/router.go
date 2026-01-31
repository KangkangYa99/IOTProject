package router

import (
	"IOTProject/internal/transport/http"
	"IOTProject/internal/transport/middleware"

	"github.com/gin-gonic/gin"
)

func InitRouter(userHandle *http.UserHandler) *gin.Engine {
	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(middleware.ErrorHandler())
	UserPublic := r.Group("/user")
	{
		UserPublic.POST("/register", userHandle.Register)
		UserPublic.POST("/login", userHandle.Login)
	}
	UserAuth := r.Group("/user")
	UserAuth.Use(middleware.JWTAUTH())
	{
		UserAuth.GET("/info", userHandle.GetUserInfo)
		UserAuth.POST("/logout", userHandle.Logout)
	}
	return r
}
