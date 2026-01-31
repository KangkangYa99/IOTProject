package http

import (
	"IOTProject/internal/domain"
	"IOTProject/internal/service"
	"IOTProject/pkg/error_code"
	myredis "IOTProject/pkg/redis"
	"IOTProject/pkg/response"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}
func (h *UserHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {

		_ = c.Error(error_code.InvalidParam)
		fmt.Printf("Register Error Details: %v\n", err)
		return
	}
	res, err := h.svc.RegisterUser(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, res)
}
func (h *UserHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(error_code.ShouldBindError)
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, res)
}
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	uid, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	userID, ok := uid.(int64)
	if !ok {
		_ = c.Error(error_code.ServerError)
		return
	}
	user, err := h.svc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, user)
}
func (h *UserHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		response.Success(c, nil)
		return
	}
	token := parts[1]
	blackKey := "jwt_blacklist:" + token
	_ = myredis.RedisClient.Set(c.Request.Context(), blackKey, "1", 24*time.Hour).Err()
	uid, _ := c.Get("userID")
	tokenKey := fmt.Sprintf("auth:token:%v", uid)
	_ = myredis.RedisClient.Del(c.Request.Context(), tokenKey).Err()
	response.Success(c, "登出成功")
}
