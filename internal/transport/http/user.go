package http

import (
	"IOTProject/internal/domain"
	"IOTProject/internal/service"
	"IOTProject/pkg/error_code"
	"IOTProject/pkg/response"
	"image"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Register 处理用户注册请求
// 绑定 JSON 请求参数，调用 Service 执行分布式锁校验、密码加密及数据库持久化，成功后返回新用户信息。
func (h *UserHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(error_code.ShouldBindError)
		return
	}
	res, err := h.svc.RegisterUser(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, res)
}

// Login 处理用户登录请求
// 绑定 JSON 请求参数，调用 Service 执行身份验证，成功后返回包含 JWT Token 的响应数据。
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

// ResetPasswordHandler 处理用户通过身份信息重置密码的 HTTP 请求。
// 路径: POST /api/auth/password/reset
func (h *UserHandler) ResetPasswordHandler(c *gin.Context) {
	var req domain.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(error_code.InvalidParam)
		return
	}
	err := h.svc.ResetPassword(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, nil)
}

// UpdateProfile 处理用户资料及密码修改请求
// 从 Context 获取当前登录用户 ID，支持修改手机号、邮箱及通过旧密码验证修改新密码。
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(error_code.ShouldBindError)
		return
	}
	val, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	uid, ok := val.(int64)
	if !ok {
		_ = c.Error(error_code.ServerError)
		return
	}
	err := h.svc.UpdateProfile(c.Request.Context(), uid, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, "修改成功")
}
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req domain.UpdatePasswordRequest
	// 1. 绑定 JSON 参数（对应前端：旧密码、新密码、验证手机号）
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(error_code.ShouldBindError)
		return
	}
	// 2. 获取当前登录用户的 userID
	val, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	uid, ok := val.(int64)
	if !ok {
		_ = c.Error(error_code.ServerError)
		return
	}
	// 3. 调用 Service 层执行修改密码逻辑
	err := h.svc.ChangePassword(c.Request.Context(), uid, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// 4. 返回成功响应
	response.Success(c, "密码修改成功")
}

// Logout 处理用户退出登录请求
// 从 Context 获取当前 Token 和 UserID，并调用 Service 将其从 Redis 白名单中移除，实现安全下线。
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

// GetUserInfo 获取当前登录用户的个人资料
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
	vo := domain.UserVO{
		UserID:      user.UserID,
		Username:    user.Username,
		RoleID:      user.RoleID,
		PhoneNumber: user.PhoneNumber,
		Email:       user.Email,
		AvatarURL:   user.AvatarURL,
		CreatedAt:   user.CreatedAt,
	}
	response.Success(c, vo)
}

// GetCaptcha 获取验证码
func (h *UserHandler) GetCaptcha(c *gin.Context) {

	var req domain.CaptchaRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		_ = c.Error(error_code.ShouldBindError)
		return
	}
	if req.DeviceID == "" {
		_ = c.Error(error_code.DeviceIDNotFound)
		return
	}

	data, err := h.svc.GenerateCaptchaService(c.Request.Context(), req)

	if err != nil {
		_ = c.Error(error_code.ServiceUnavailable)
	}
	response.Success(c, data)
}

// UploadAvatar 上传头像
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	id, exists := c.Get("userID")
	if !exists {
		_ = c.Error(error_code.NotLogin)
		return
	}
	userID := id.(int64)

	// 限制请求体大小 (5MB)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)

	file, err := c.FormFile("avatar")
	if err != nil {
		_ = c.Error(error_code.InvalidParams.WithDetails("未找到 avatar 字段"))
		return
	}

	f, err := file.Open()
	if err != nil {
		_ = c.Error(error_code.FileUploadFail)
		return
	}
	defer f.Close()

	// 校验图片内容
	img, format, err := image.Decode(f)
	if err != nil {
		log.Printf("[SECURITY] 图片解码失败，可能是伪造文件: %v", err)
		_ = c.Error(error_code.InvalidFileType.WithDetails("非法图片格式或内容已损坏"))
		return
	}

	// 校验图片分辨率，防止像素炸弹
	if img.Bounds().Dx() > 4096 || img.Bounds().Dy() > 4096 {
		_ = c.Error(error_code.InvalidParams.WithDetails("图片尺寸过大"))
		return
	}

	log.Printf("[INFO] 图片校验成功，格式: %s", format)

	//保存文件
	filename := uuid.New().String() + filepath.Ext(file.Filename)
	dst := "./static/uploads/" + filename

	if err = c.SaveUploadedFile(file, dst); err != nil {
		_ = c.Error(error_code.FileUploadFail)
		return
	}

	//更新数据库
	avatarURL := "/static/uploads/" + filename
	if err = h.svc.UpdateUserAvatar(c.Request.Context(), domain.UpdateAvatarRequest{
		UserID:    userID,
		AvatarURL: avatarURL,
	}); err != nil {
		_ = os.Remove(dst)
		_ = c.Error(err)
		return
	}
	response.Success(c, gin.H{"url": avatarURL})
}
