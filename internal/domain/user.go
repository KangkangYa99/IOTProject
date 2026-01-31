package domain

import (
	"context"
	"time"
)

type User struct {
	UserID       int64      `json:"user_id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	PhoneNumber  string     `json:"phone_number"`
	AvatarURL    *string    `json:"avatar_url"`
	Email        string     `json:"email"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	RoleID       int        `json:"role_id"`
	Status       int        `json:"status"`
}
type RegisterInfo struct {
	UserID       int64
	Username     string
	PasswordHash string
	PhoneNumber  string
	Email        string
	AdminToken   string
	RoleID       int
	Status       int
}
type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,min=6,max=50"`
	PhoneNumber string `json:"phone" binding:"required,len=11"`
	Email       string `json:"email" binding:"required,email"`
	AdminToken  string `json:"admin_token"`
	RoleID      int    `json:"role_id"`
}

type RegisterResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}
type UpdateUser struct {
	UserID       int64
	PassWordHash string
	PhoneNumber  string
	Email        string
}
type UpdateUserRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password" binding:"omitempty,min=6"`
	PhoneNumber string `json:"phone" binding:"omitempty,len=11"`
	Email       string `json:"email" binding:"omitempty,email"`
}
type LoginRequest struct {
	Identity string `json:"identity" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=30"`
}
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}
type UserInterface interface {
	CreateUser(ctx context.Context, user *RegisterInfo) error
	CheckUserExists(ctx context.Context, username, phone, email string) (bool, bool, bool, error)
	UpdateUser(ctx context.Context, user *UpdateUser) error
	FindByIdentity(ctx context.Context, identity string) (*User, error)
	GetUserInfoByID(ctx context.Context, UserID int64) (*User, error)
}
