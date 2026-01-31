package middleware

import (
	"IOTProject/pkg/error_code"
	my "IOTProject/pkg/redis"
	"IOTProject/pkg/response"
	"IOTProject/pkg/utils"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

func JWTAUTH() gin.HandlerFunc {
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
		tokenKey := fmt.Sprintf("auth:token:%d", claims.UserID)
		latestToken, err := my.RedisClient.Get(c.Request.Context(), tokenKey).Result()
		if err == nil && latestToken != parts[1] {
			_ = c.Error(error_code.TokenOutError) // 或者自定义一个 error_code.UserKickedOut
			c.Abort()
			return
		}

		blackKey := "jwt_blacklist:" + parts[1]
		exists, _ := my.RedisClient.Exists(c.Request.Context(), blackKey).Result()
		if exists > 0 {
			_ = c.Error(error_code.TokenOutError)
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("roleID", claims.RoleID)
		c.Next()

	}
}
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			if myErr, ok := err.(*error_code.APIError); ok {
				response.Fail(c, myErr.Code, myErr.Message)
			} else {
				response.Fail(c, 500, "Internal Server Error")
			}
			c.Abort()
		}
	}
}
