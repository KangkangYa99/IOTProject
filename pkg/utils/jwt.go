package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	secretKey                  []byte
	ErrJWTSecretNotInitialized = errors.New("jwt secret not initialized")
)

func InitJWT(secret string) {
	secretKey = []byte(secret)
}

type MyClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userID int64) (string, error) {
	if len(secretKey) == 0 {
		return "", ErrJWTSecretNotInitialized
	}
	nowTime := time.Now()
	expireTime := nowTime.Add(3 * 24 * time.Hour)
	claims := MyClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			NotBefore: jwt.NewNumericDate(nowTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}
func ParseToken(tokenstring string) (*MyClaims, error) {
	if len(secretKey) == 0 {
		return nil, ErrJWTSecretNotInitialized
	}
	token, err := jwt.ParseWithClaims(tokenstring, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*MyClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
