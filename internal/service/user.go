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

	"github.com/jackc/pgconn"
	"github.com/redis/go-redis/v9"
)

type UserService struct {
	repo  domain.UserInterface
	redis *redis.Client
}

func NewUserService(repo domain.UserInterface, rdb *redis.Client) UserService {
	return UserService{
		repo:  repo,
		redis: rdb,
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
	//需要锁定的三个核心维度
	locks := map[string]string{
		"lock:reg:user:" + req.Username:     "username",
		"lock:reg:phone:" + req.PhoneNumber: "phone",
		"lock:reg:email:" + req.Email:       "email",
	}

	var acquiredLocks []string
	for lockKey := range locks {
		ok, err := s.redis.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
		if err != nil || !ok {
			for _, k := range acquiredLocks {
				s.redis.Del(ctx, k)
			}
			return nil, error_code.RequestTooFrequentError
		}
		acquiredLocks = append(acquiredLocks, lockKey)
	}

	defer func() {
		for _, k := range acquiredLocks {
			s.redis.Del(ctx, k)
		}
	}()

	uEx, pEx, eEx, err := s.repo.CheckUserExists(ctx, req.Username, req.PhoneNumber, req.Email)
	if err != nil {
		return nil, error_code.DatabaseError
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
	roleID := 1
	if req.RoleID == 2 || req.RoleID == 3 {
		expectedToken := fmt.Sprintf("IOT%d", req.RoleID)
		if req.AdminToken != expectedToken {
			return nil, error_code.NoPermission
		}
		roleID = req.RoleID
	}

	userInfo := &domain.RegisterInfo{
		Username:     req.Username,
		PasswordHash: hashedPwd,
		PhoneNumber:  req.PhoneNumber,
		Email:        req.Email,
		AdminToken:   req.AdminToken,
		RoleID:       roleID,
	}
	err = s.repo.CreateUser(ctx, userInfo)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			field := "用户数据"
			switch pgErr.ConstraintName {
			case "users_username_key":
				field = "用户名"
			case "users_email_key":
				field = "邮箱"
			case "users_phone_number_key":
				field = "手机号"
			}
			return nil, error_code.DatabaseError.WithDetails(field + "已存在")
		}
		return nil, error_code.DatabaseError
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
		updateData.PassWordHash = hashed
	}
	err := s.repo.UpdateUser(ctx, updateData)
	if err != nil {
		if err.Error() == "user not found" {
			return error_code.UserNotExists
		}
		return error_code.DatabaseError
	}
	return nil
}
func (s *UserService) Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error) {
	user, err := s.repo.FindByIdentity(ctx, req.Identity)
	if err != nil {
		return nil, error_code.PasswordFail
	}
	if user.Status != 0 {
		return nil, error_code.NoPermission.WithDetails("账号状态异常")
	}
	if !crypto.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, error_code.PasswordFail
	}
	if !crypto.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, error_code.PasswordFail
	}
	token, err := utils.GenerateToken(user.UserID, user.RoleID)
	if err != nil {
		return nil, error_code.ServerError
	}
	tokenKey := fmt.Sprintf("auth:token:%d", user.UserID)
	err = s.redis.Set(ctx, tokenKey, token, 24*time.Hour).Err()
	if err != nil {
		fmt.Printf("Redis 记录 Token 失败: %v\n", err)
	}
	return &domain.LoginResponse{
		AccessToken: token,
		ExpiresIn:   86400, // 24小时
	}, nil
}
func (s *UserService) GetUserByID(ctx context.Context, userID int64) (*domain.User, error) {
	user, err := s.repo.GetUserInfoByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}
