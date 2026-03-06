package domain

import (
	"context"
	"time"
)

// User 数据库对应的用户实体模型
type User struct {
	UserID       int64     `gorm:"primaryKey;column:user_id"`
	Username     string    `gorm:"column:username"`
	PasswordHash string    `gorm:"column:password_hash" json:"-"`
	PhoneNumber  string    `gorm:"column:phone_number"`
	Email        string    `gorm:"column:email"`
	RoleID       int       `gorm:"column:role_id"`
	Status       int       `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// RegisterRequest 用户自主注册的请求模型
// 包含基础的账号信息，并使用了 Gin Validator 标签强制校验字段长度、手机号位数及邮箱格式。
type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,min=6,max=50"`
	PhoneNumber string `json:"phone" binding:"required,len=11"`
	Email       string `json:"email" binding:"required,email"`
}

// RegisterResponse 注册成功后的回执模型
// 仅向客户端返回用户唯一标识和用户名，不包含敏感信息。
type RegisterResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

// LoginRequest 登录请求模型
// Identity 字段具备多维性，支持用户输入用户名、手机号或邮箱作为唯一登录标识。
type LoginRequest struct {
	Identity string `json:"identity" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=30"`
}

// LoginResponse 登录成功后的响应模型
// 返回 JWT 访问令牌（AccessToken）以及以秒为单位的过期偏移量（ExpiresIn）。
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type UpdateUser struct {
	UserID       int64  `gorm:"primaryKey"`
	PhoneNumber  string `json:"phone" gorm:"column:phone_number"`
	Email        string `json:"email" gorm:"column:email"`
	PasswordHash string `json:"-" gorm:"column:password_hash"`
	OldPassword  string `json:"old_password" gorm:"-"`
}

// UpdateUserRequest 用户修改资料的 HTTP 请求模型
// 字段标记为 omitempty，允许用户仅提交需要修改的项（如只改手机号而不改密码）。
type UpdateUserRequest struct {
	OldPassword string `json:"old_password" gorm:"-"`
	NewPassword string `json:"new_password" binding:"omitempty,min=6" gorm:"column:password_hash"`
	PhoneNumber string `json:"phone" binding:"omitempty,len=11" gorm:"column:phone_number"`
	Email       string `json:"email" binding:"omitempty,email" gorm:"column:email"`
}

type AdminCreateUserRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,min=6,max=50"`
	PhoneNumber string `json:"phone" binding:"required,len=11"`
	Email       string `json:"email" binding:"required,email"`
	RoleID      int    `json:"role_id" binding:"required,min=1,max=3"`
	Status      int    `json:"status"`
}
type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	CheckUserExists(ctx context.Context, username, phone, email string) (bool, bool, bool, error)
	FindByIdentity(ctx context.Context, identity string) (*User, error)
	UpdateUser(ctx context.Context, user *UpdateUser) error
	GetUserInfoByID(ctx context.Context, UserID int64) (*User, error)
	GetUserRoleByID(ctx context.Context, userID int64) (int, error)
}
