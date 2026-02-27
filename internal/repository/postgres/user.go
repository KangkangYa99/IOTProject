package postgres

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/error_code"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) domain.UserInterface {
	return &UserRepository{
		db: db,
	}
}
func (r *UserRepository) CreateUser(ctx context.Context, user *domain.RegisterInfo) error {
	query := `INSERT INTO users (username, password_hash, phone_number, email, role_id,status)
				VALUES ($1, $2, $3, $4, $5,$6) RETURNING user_id`
	err := r.db.QueryRow(ctx, query,
		user.Username,
		user.PasswordHash,
		user.PhoneNumber,
		user.Email,
		user.RoleID,
		0,
	).Scan(&user.UserID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: %v", error_code.ErrUserExists, err)
		}
		return fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	return nil
}
func (r *UserRepository) CheckUserExists(ctx context.Context, username, phone, email string) (bool, bool, bool, error) {
	var usernameExists, phoneExists, emailExists bool

	query := `
        SELECT 
            EXISTS(SELECT 1 FROM users WHERE username = $1),
            EXISTS(SELECT 1 FROM users WHERE phone_number = $2),
            EXISTS(SELECT 1 FROM users WHERE email = $3)
    `
	err := r.db.QueryRow(ctx, query, username, phone, email).Scan(
		&usernameExists,
		&phoneExists,
		&emailExists,
	)
	if err != nil {
		return false, false, false, fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	return usernameExists, phoneExists, emailExists, nil
}
func (r *UserRepository) UpdateUser(ctx context.Context, user *domain.UpdateUser) error {
	query := `
        UPDATE users 
        SET 
            password_hash = COALESCE(NULLIF($1, ''), password_hash),
            phone_number  = COALESCE(NULLIF($2, ''), phone_number),
            email         = COALESCE(NULLIF($3, ''), email),
            updated_at    = $4
        WHERE user_id = $5`
	result, err := r.db.Exec(ctx, query,
		user.PasswordHash,
		user.PhoneNumber,
		user.Email,
		time.Now(),
		user.UserID,
	)
	if err != nil {
		return fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	row := result.RowsAffected()
	if row == 0 {
		return fmt.Errorf("%w", error_code.UserNotExists)
	}
	return nil
}
func (r *UserRepository) FindByIdentity(ctx context.Context, identity string) (*domain.User, error) {
	var user domain.User
	query := `
		SELECT user_id, password_hash, role_id, status 
		FROM users 
		WHERE (username = $1 OR phone_number = $1 OR email = $1)
		LIMIT 1
	`
	err := r.db.QueryRow(ctx, query, identity).Scan(
		&user.UserID,
		&user.PasswordHash,
		&user.RoleID,
		&user.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, error_code.UserNotExists
		}
		return nil, fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}

	return &user, nil
}
func (r *UserRepository) GetUserInfoByID(ctx context.Context, UserID int64) (*domain.User, error) {
	var user domain.User
	query := `
        SELECT user_id, username, phone_number, email, avatar_url, role_id, status, created_at, last_login_at 
        FROM users 
        WHERE user_id = $1
    `
	err := r.db.QueryRow(ctx, query, UserID).Scan(
		&user.UserID,
		&user.Username,
		&user.PhoneNumber,
		&user.Email,
		&user.AvatarURL, // 对应 *string
		&user.RoleID,
		&user.Status,
		&user.CreatedAt,
		&user.LastLoginAt, // 对应 *time.Time
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, error_code.UserNotExists
		}
		return nil, fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	return &user, nil
}
func (r *UserRepository) GetUserRoleByID(ctx context.Context, userID int64) (int, error) {
	var roleID int
	query := `SELECT role_id FROM users WHERE user_id = $1`
	err := r.db.QueryRow(ctx, query, userID).Scan(&roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, error_code.UserNotExists
		}
		return 0, fmt.Errorf("%w: %v", error_code.ErrDB, err)
	}
	return roleID, nil
}
