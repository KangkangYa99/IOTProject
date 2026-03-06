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
	repo   domain.UserRepository
	redis  *redis.Client
	locker utils.Locker
}

func NewUserService(
	repo domain.UserRepository,
	rdb *redis.Client,
	locker utils.Locker,
) *UserService {
	return &UserService{
		repo:   repo,
		redis:  rdb,
		locker: locker,
	}
}

// ValidPassword 校验密码强度是否符合安全规范
// 检查项包括：最小长度限制、必须包含小写字母、必须包含数字，以及常见弱密码黑名单过滤。
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

// RegisterUser 执行新用户注册业务流程
// 包含多重分布式锁保证并发安全、唯一性字段（用户名/手机/邮箱）查重、密码强度校验及哈希加密。
// 成功后将用户信息持久化至数据库，并返回用户基本信息。
func (s *UserService) RegisterUser(ctx context.Context, req domain.RegisterRequest) (*domain.RegisterResponse, error) {
	//分布式锁
	locks := map[string]string{
		"lock:reg:user:" + req.Username:     "username",
		"lock:reg:phone:" + req.PhoneNumber: "phone",
		"lock:reg:email:" + req.Email:       "email",
	}
	var acquiredLocks []string
	for lockKey := range locks {
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
	//判断用户名，手机号码，邮箱是否被注册
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
	//验证密码强度
	if err = s.ValidPassword(req.Password); err != nil {
		return nil, err
	}
	//sha256加密
	hashedPwd, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, error_code.ServerError
	}
	userInfo := &domain.User{
		Username:     req.Username,
		PasswordHash: hashedPwd,
		PhoneNumber:  req.PhoneNumber,
		Email:        req.Email,
		RoleID:       1,
		Status:       1,
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

// Login 处理用户身份认证及登录态颁发
// 支持多维度身份识别，校验账号状态及密码哈希；认证通过后生成 JWT 令牌，
// 并同步维护 Redis 中的 Token 白名单集合，设置 3 天有效期。
func (s *UserService) Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error) {
	user, err := s.repo.FindByIdentity(ctx, req.Identity)
	if err != nil {
		return nil, error_code.PasswordFail
	}
	//1正常 0禁止登录
	if user.Status != 1 {
		return nil, error_code.NoPermission.WithDetails("账号状态异常，请联系管理员")
	}
	//验证密码
	if !crypto.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, error_code.PasswordFail
	}
	//生成token
	token, err := utils.GenerateToken(user.UserID)
	if err != nil {
		return nil, error_code.ServerError
	}
	tokenSetKey := fmt.Sprintf("auth:tokens:%d", user.UserID)
	err = s.redis.SAdd(ctx, tokenSetKey, token).Err()
	if err != nil {
		fmt.Printf("Redis 记录 Token 失败: %v\n", err)
	}
	//三天时间
	s.redis.Expire(ctx, tokenSetKey, 3*24*time.Hour)
	return &domain.LoginResponse{
		AccessToken: token,
		ExpiresIn:   3 * 86400,
	}, nil
}

// Logout 执行用户登出及会话清理
// 从 Redis 的活跃 Token 集合中移除当前令牌，若该用户所有设备均已离线，则物理删除对应的 Key。
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

// UpdateProfile 变更用户个人资料或密码
// 支持局部更新手机号与邮箱（包含查重逻辑）；若涉及新密码变更，则触发强度校验与加密，
// 最终调用 Repository 执行数据库增量更新。
func (s *UserService) UpdateProfile(ctx context.Context, userID int64, req *domain.UpdateUserRequest) error {
	//获取旧密码
	user, err := s.repo.GetUserInfoByID(ctx, userID)
	if err != nil {
		return error_code.UserNotExists
	}
	if !crypto.CheckPasswordHash(req.OldPassword, user.PasswordHash) {
		return error_code.PasswordFail
	}

	//检查手机号，邮箱是否被注册
	_, pEx, eEx, err := s.repo.CheckUserExists(ctx, "", req.PhoneNumber, req.Email)
	if err != nil {
		return err
	}
	if pEx {
		return error_code.UserNumberExists
	}
	if eEx {
		return error_code.UserEmailExists
	}

	updateData := &domain.UpdateUser{
		UserID:      userID,
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
	}
	if req.NewPassword != "" {
		// 增加一个简单的密码强度校验
		if err = s.ValidPassword(req.NewPassword); err != nil {
			return err
		}
		hashed, err2 := crypto.HashPassword(req.NewPassword)
		if err2 != nil {
			return error_code.ServerError
		}
		updateData.PasswordHash = hashed
	}
	if err := s.repo.UpdateUser(ctx, updateData); err != nil {
		if errors.Is(err, error_code.ErrUserExists) {
			return error_code.UserExists
		}
		return err
	}
	return nil
}

// GetUserByID 根据用户唯一 ID 获取详细信息
// 直接从 Repository 层获取用户实体数据，常用于个人中心展示或内部逻辑调用。
func (s *UserService) GetUserByID(ctx context.Context, userID int64) (*domain.User, error) {
	user, err := s.repo.GetUserInfoByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
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
	userInfo := &domain.User{
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
