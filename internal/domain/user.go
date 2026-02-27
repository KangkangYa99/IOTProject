package domain

import (
	"context"
	"database/sql"
	"time"
)

type User struct {
	UserID       int64
	Username     string
	PasswordHash string
	PhoneNumber  string
	AvatarURL    *string
	Email        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
	RoleID       int
	Status       sql.NullInt64
}
type RegisterInfo struct {
	UserID       int64
	Username     string
	PasswordHash string
	PhoneNumber  string
	Email        string
	RoleID       int
	Status       sql.NullInt64
}
type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,min=6,max=50"`
	PhoneNumber string `json:"phone" binding:"required,len=11"`
	Email       string `json:"email" binding:"required,email"`
	Status      int    `json:"status"`
}

type RegisterResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}
type AdminCreateUserRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,min=6,max=50"`
	PhoneNumber string `json:"phone" binding:"required,len=11"`
	Email       string `json:"email" binding:"required,email"`
	RoleID      int    `json:"role_id" binding:"required,min=1,max=3"`
	Status      int    `json:"status"`
}
type UpdateUser struct {
	UserID       int64
	PasswordHash string
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
	GetUserRoleByID(ctx context.Context, userID int64) (int, error)
}
