package postgres

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/error_code"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type UserRepository struct {
	gorm *gorm.DB
}

func NewUserRepository(gorm *gorm.DB) domain.UserRepository {
	return &UserRepository{
		gorm: gorm,
	}
}

// CreateUser 在数据库中创建新用户
// 默认设置用户状态为正常(1)，并处理 PostgreSQL 的唯一约束冲突（错误码 23505），
// 若用户名、手机号或邮箱已存在，则返回自定义的 ErrUserExists 错误。
func (r *UserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	user.Status = 1
	err := r.gorm.WithContext(ctx).Create(user).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: %v", error_code.ErrUserExists, err)
		}
		return fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	return nil
}

// CheckUserExists 并行检查用户名、手机号和邮箱在数据库中是否已存在
// 使用高效的 EXISTS 子查询一次性返回三个维度的存在性状态，
// 返回值依次对应：用户名是否存在、手机号是否存在、邮箱是否存在。
func (r *UserRepository) CheckUserExists(ctx context.Context, username, phone, email string) (bool, bool, bool, error) {
	var results []struct {
		Username    string
		PhoneNumber string
		Email       string
	}
	err := r.gorm.WithContext(ctx).Model(&domain.User{}).
		Select("username, phone_number, email").
		Where("username = ? OR phone_number = ? OR email = ?", username, phone, email).
		Find(&results).Error
	if err != nil {
		return false, false, false, err
	}
	var uEx, pEx, eEx bool
	for _, row := range results {
		if row.Username == username {
			uEx = true
		}
		if row.PhoneNumber == phone {
			pEx = true
		}
		if row.Email == email {
			eEx = true
		}
	}
	return uEx, pEx, eEx, nil
}

// FindByIdentity 根据身份标识查找唯一用户
// 支持通过用户名、手机号或邮箱（三选一）进行匹配，
// 若记录不存在则返回 UserNotExists 错误，常用于登录逻辑中的用户定位。
func (r *UserRepository) FindByIdentity(ctx context.Context, identity string) (*domain.User, error) {
	var user domain.User
	err := r.gorm.WithContext(ctx).
		Where("username = ?", identity).
		Or("phone_number = ?", identity).
		Or("email = ?", identity).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, error_code.UserNotExists
		}
		return nil, fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}

	return &user, nil
}

// UpdatePasswordHash 根据用户 ID 更新其加密后的密码哈希值。
func (r *UserRepository) UpdatePasswordHash(ctx context.Context, userID int64, newHash string) error {
	return r.gorm.WithContext(ctx).
		Model(&domain.User{}).
		Where("user_id = ?", userID).
		Update("password_hash", newHash).Error
}

// UpdateUser 更新指定用户的个人资料
func (r *UserRepository) UpdateUser(ctx context.Context, user *domain.UpdateUser) error {
	updates := make(map[string]interface{})
	if user.PhoneNumber != "" {
		updates["phone_number"] = user.PhoneNumber
	}
	if user.Email != "" {
		updates["email"] = user.Email
	}
	if user.PasswordHash != "" {
		updates["password_hash"] = user.PasswordHash
	}
	if len(updates) == 0 {
		return nil
	}
	result := r.gorm.WithContext(ctx).
		Model(&domain.User{}).
		Where("user_id = ?", user.UserID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// GetUserInfoByID 根据用户 ID 获取用户详细信息
func (r *UserRepository) GetUserInfoByID(ctx context.Context, userID int64) (*domain.User, error) {
	var user domain.User
	err := r.gorm.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, error_code.UserNotExists
		}
		return nil, fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	return &user, nil
}

func (r *UserRepository) GetUserRoleByID(ctx context.Context, userID int64) (int, error) {
	var roleID int

	res := r.gorm.WithContext(ctx).Table("users").Where("user_id = ?", userID).Select("role_id").Scan(&roleID)
	if res.Error != nil {
		return 0, fmt.Errorf("%w: %v", error_code.ErrDB, res.Error)
	}
	if res.RowsAffected == 0 {
		return 0, error_code.UserNotExists
	}
	return roleID, nil
}

// UpdateAvatar 更新用户的头像路径
func (r *UserRepository) UpdateAvatar(ctx context.Context, req *domain.UpdateAvatarRequest) error {
	result := r.gorm.WithContext(ctx).
		Model(&domain.User{}).
		Where("user_id = ?", req.UserID).
		Update("avatar_url", req.AvatarURL)

	if result.Error != nil {
		return fmt.Errorf("%w: %v", error_code.ErrDB, result.Error)
	}
	if result.RowsAffected == 0 {
		return error_code.UserNotExists
	}
	return nil
}
