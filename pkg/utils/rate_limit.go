package utils

import (
	"IOTProject/pkg/error_code"
	"IOTProject/pkg/redis"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig 定义限流器的配置参数
type RateLimitConfig struct {
	KeyPrefix string        // Redis key 前缀 (如: "login", "ws")
	Limit     int64         // 允许的最大次数
	Time      time.Duration // 时间窗口长度
}

// RateLimitMiddleware 核心限流中间件
var rateLimitScript = `
local count = redis.call("INCR", KEYS[1])
if count == 1 then
    redis.call("EXPIRE", KEYS[1], ARGV[3])
end
if count == tonumber(ARGV[1]) then
    redis.call("SET", KEYS[2], "1", "EX", ARGV[2])
    return 0
end
if count > tonumber(ARGV[1]) then
    return -1
end
return count
`

func RateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		prefix := config.KeyPrefix
		if prefix == "" {
			prefix = "default"
		}
		countKey := fmt.Sprintf("limit:count:%s:%s", prefix, clientIP)
		lockKey := fmt.Sprintf("limit:lock:%s:%s", prefix, clientIP)
		ctx := c.Request.Context()
		if exists, _ := redis.RedisClient.Exists(ctx, lockKey).Result(); exists > 0 {
			ttl, _ := redis.RedisClient.TTL(ctx, lockKey).Result()
			_ = c.Error(error_code.RequestTooFrequentError.WithDetails(fmt.Sprintf("请 %d 秒后再试", int(ttl.Seconds()))))
			c.Abort()
			return
		}
		result, err := redis.RedisClient.Eval(ctx, rateLimitScript,
			[]string{countKey, lockKey},
			config.Limit, 60, int(config.Time.Seconds())).Int64()
		if err == nil && result == -1 {
			ttl, _ := redis.RedisClient.TTL(ctx, lockKey).Result()
			_ = c.Error(error_code.RequestTooFrequentError.WithDetails(fmt.Sprintf("请 %d 秒后再试", int(ttl.Seconds()))))
			c.Abort()
			return
		}
		c.Next()
	}
}
