package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey []byte

func InitJWT(secret string) {
	secretKey = []byte(secret)
}

type MyClaims struct {
	UserID int64 `json:"user_id"`
	RoleID int   `json:"role_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userID int64, roleID int) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(2 * time.Hour)
	claims := MyClaims{
		UserID: userID,
		RoleID: roleID,
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
	token, err := jwt.ParseWithClaims(tokenstring, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, err
}
