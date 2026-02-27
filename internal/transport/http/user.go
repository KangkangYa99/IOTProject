package http

import (
	"IOTProject/internal/domain"
	"IOTProject/internal/service"
	"IOTProject/pkg/error_code"
	"IOTProject/pkg/response"
	"fmt"
	"net/http"
	"strings"

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
func (h *UserHandler) AdminCreateUser(c *gin.Context) {
	_, exists := c.Get("userID")

	if !exists {
		response.Fail(c, 401, "未登录")
		return
	}
	currentUserRole, exists := c.Get("roleID")
	if !exists {
		response.Fail(c, 401, "权限信息缺失")
		return
	}
	roleID := currentUserRole.(int)
	// 权限验证：只有管理员(2)及以上才能创建用户
	if roleID < 2 {
		response.Fail(c, 403, "权限不足，无法创建用户")
		return
	}
	var req domain.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	res, err := h.svc.AdminCreateUser(c.Request.Context(), req, roleID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, res)
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
	uid, exists := c.Get("userID")
	if exists {
		userID, ok := uid.(int64)
		if ok {
			_ = h.svc.Logout(c.Request.Context(), userID, token)
		}
	}
	response.Success(c, "登出成功")
}
