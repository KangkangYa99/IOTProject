package service

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/crypto"
	"IOTProject/pkg/error_code"
	"IOTProject/pkg/utils"
	"context"
	"errors"
	"fmt"
	"time"
	"unicode"

	"github.com/redis/go-redis/v9"
)

type UserService struct {
	repo   domain.UserInterface
	redis  *redis.Client
	locker utils.Locker
}

func NewUserService(
	repo domain.UserInterface,
	rdb *redis.Client,
	locker utils.Locker,
) *UserService {
	return &UserService{
		repo:   repo,
		redis:  rdb,
		locker: locker,
	}
}
func (s *UserService) ValidPassword(password string) error {
	if len(password) < 6 {
		return error_code.InvalidParam.WithDetails("密码长度不能低于6位数")
	}
	var (
		hasLower bool
		hasDigit bool
	)
	for _, ch := range password {
		if unicode.IsLower(ch) {
			hasLower = true
		}
		if unicode.IsDigit(ch) {
			hasDigit = true
		}
	}
	if !hasLower {
		return error_code.InvalidParam.WithDetails("密码必须包含至少一个小写字母")
	}
	if !hasDigit {
		return error_code.InvalidParam.WithDetails("密码必须包含至少一个数字")
	}
	weakPasswords := []string{"123456", "password", "qwerty", "abc123"}
	for _, weak := range weakPasswords {
		if password == weak {
			return error_code.InvalidParam.WithDetails("密码过于简单，请选择更复杂的密码")
		}
	}

	return nil
}
func (s *UserService) RegisterUser(ctx context.Context, req domain.RegisterRequest) (*domain.RegisterResponse, error) {
	locks := map[string]string{
		"lock:reg:user:" + req.Username:     "username",
		"lock:reg:phone:" + req.PhoneNumber: "phone",
		"lock:reg:email:" + req.Email:       "email",
	}

	var acquiredLocks []string
	for _, lockKey := range locks {
		ok, err := s.locker.Acquire(ctx, lockKey, 10)
		if err != nil || !ok {
			for _, k := range acquiredLocks {
				_ = s.locker.Release(ctx, k)
			}
			return nil, error_code.RequestTooFrequentError
		}
		acquiredLocks = append(acquiredLocks, lockKey)
	}

	defer func() {
		for _, k := range acquiredLocks {
			_ = s.locker.Release(ctx, k)
		}
	}()

	uEx, pEx, eEx, err := s.repo.CheckUserExists(ctx, req.Username, req.PhoneNumber, req.Email)
	if err != nil {
		return nil, err
	}
	if uEx {
		return nil, error_code.UserExists
	}
	if pEx {
		return nil, error_code.UserNumberExists
	}
	if eEx {
		return nil, error_code.UserEmailExists
	}
	if err = s.ValidPassword(req.Password); err != nil {
		return nil, err
	}
	hashedPwd, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, error_code.ServerError
	}
	userInfo := &domain.RegisterInfo{
		Username:     req.Username,
		PasswordHash: hashedPwd,
		PhoneNumber:  req.PhoneNumber,
		Email:        req.Email,
		RoleID:       1,
	}
	err = s.repo.CreateUser(ctx, userInfo)
	if err != nil {
		if errors.Is(err, error_code.ErrUserExists) {
			return nil, error_code.UserExists
		}
		if errors.Is(err, error_code.ErrDB) {
			return nil, error_code.DatabaseError
		}
		return nil, err
	}

	return &domain.RegisterResponse{
		UserID:   userInfo.UserID,
		Username: userInfo.Username,
	}, nil
}
func (s *UserService) AdminCreateUser(ctx context.Context, req domain.AdminCreateUserRequest, currentUserRole int) (*domain.RegisterResponse, error) {
	switch currentUserRole {
	case 1: // 普通员工
		return nil, error_code.NoPermission.WithDetails("无法创建用户")
	case 2: // 管理员
		if req.RoleID != 1 {
			return nil, error_code.NoPermission.WithDetails("管理员只能创建普通员工")
		}
	case 3: // 经理
		if req.RoleID != 1 && req.RoleID != 2 {
			return nil, error_code.NoPermission.WithDetails("经理只能创建普通员工或管理员")
		}
	default:
		return nil, error_code.NoPermission.WithDetails("无效的用户角色")
	}
	if err := s.ValidPassword(req.Password); err != nil {
		return nil, err
	}
	uEx, pEx, eEx, err := s.repo.CheckUserExists(ctx, req.Username, req.PhoneNumber, req.Email)
	if err != nil {
		return nil, err
	}
	if uEx {
		return nil, error_code.UserExists
	}
	if pEx {
		return nil, error_code.UserNumberExists
	}
	if eEx {
		return nil, error_code.UserEmailExists
	}
	hashedPwd, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, error_code.ServerError
	}
	userInfo := &domain.RegisterInfo{
		Username:     req.Username,
		PasswordHash: hashedPwd,
		PhoneNumber:  req.PhoneNumber,
		Email:        req.Email,
		RoleID:       req.RoleID,
	}
	err = s.repo.CreateUser(ctx, userInfo)
	if err != nil {
		if errors.Is(err, error_code.ErrUserExists) {
			return nil, error_code.UserExists
		}
		if errors.Is(err, error_code.ErrDB) {
			return nil, error_code.DatabaseError
		}
		return nil, err
	}
	return &domain.RegisterResponse{
		UserID:   userInfo.UserID,
		Username: userInfo.Username,
	}, nil
}
func (s *UserService) UpdateProfile(ctx context.Context, userID int64, req domain.UpdateUserRequest) error {
	updateData := &domain.UpdateUser{
		UserID:      userID,
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
	}
	if req.NewPassword != "" {
		hashed, err := crypto.HashPassword(req.NewPassword)
		if err != nil {
			return error_code.ServerError
		}
		updateData.PasswordHash = hashed
	}
	err := s.repo.UpdateUser(ctx, updateData)
	if err != nil {
		return err
	}
	return nil
}
func (s *UserService) Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error) {
	user, err := s.repo.FindByIdentity(ctx, req.Identity)
	if err != nil {
		return nil, error_code.PasswordFail
	}
	if !user.Status.Valid || user.Status.Int64 != 0 {
		return nil, error_code.NoPermission.WithDetails("账号状态异常")
	}

	if !crypto.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, error_code.PasswordFail
	}
	token, err := utils.GenerateToken(user.UserID)
	if err != nil {
		return nil, error_code.ServerError
	}
	tokenSetKey := fmt.Sprintf("auth:tokens:%d", user.UserID)
	err = s.redis.SAdd(ctx, tokenSetKey, token).Err()
	if err != nil {
		fmt.Printf("Redis 记录 Token 失败: %v\n", err)
	}
	s.redis.Expire(ctx, tokenSetKey, 3*24*time.Hour)
	return &domain.LoginResponse{
		AccessToken: token,
		ExpiresIn:   3 * 86400,
	}, nil
}
func (s *UserService) GetUserByID(ctx context.Context, userID int64) (*domain.User, error) {
	user, err := s.repo.GetUserInfoByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}
func (s *UserService) Logout(ctx context.Context, userID int64, token string) error {
	tokenSetKey := fmt.Sprintf("auth:tokens:%d", userID)
	err := s.redis.SRem(ctx, tokenSetKey, token).Err()
	if err != nil {
		return fmt.Errorf("failed to remove token: %w", err)
	}
	count, err := s.redis.SCard(ctx, tokenSetKey).Result()
	if err == nil && count == 0 {
		s.redis.Del(ctx, tokenSetKey)
	}
	return nil
}
