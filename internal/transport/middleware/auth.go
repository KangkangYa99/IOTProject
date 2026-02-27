package middleware

import (
	"IOTProject/internal/domain"
	"IOTProject/pkg/error_code"
	my "IOTProject/pkg/redis"
	"IOTProject/pkg/response"
	"IOTProject/pkg/utils"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func JWTAUTH(userRepo domain.UserInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			_ = c.Error(error_code.NotLogin)
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			_ = c.Error(error_code.InvalidToken)
			c.Abort()
			return
		}
		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			_ = c.Error(error_code.InvalidToken)
			c.Abort()
			return
		}
		tokenSetKey := fmt.Sprintf("auth:tokens:%d", claims.UserID)
		exists, err := my.RedisClient.SIsMember(c.Request.Context(), tokenSetKey, parts[1]).Result()
		if err != nil || !exists {
			_ = c.Error(error_code.TokenOutError)
			c.Abort()
			return
		}

		roleID, err := userRepo.GetUserRoleByID(c.Request.Context(), claims.UserID)
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		roleKey := fmt.Sprintf("user:role:%d", claims.UserID)
		my.RedisClient.Set(c.Request.Context(), roleKey, strconv.Itoa(roleID), 2*time.Hour)

		c.Set("userID", claims.UserID)
		c.Set("roleID", roleID)
		c.Next()

	}
}
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			var myErr *error_code.APIError
			if errors.As(err, &myErr) {
				response.Fail(c, myErr.Code, myErr.Message)
			}
			c.Abort()
		}
	}
}
