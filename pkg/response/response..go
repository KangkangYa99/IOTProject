package response

import (
	"IOTProject/pkg/error_code"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response[T any] struct {
	Code    int    `json:"code"`    // 自定义业务状态码
	Data    T      `json:"data"`    // 数据载体
	Message string `json:"message"` // 提示信息
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response[any]{
		Code:    200,
		Data:    data,
		Message: "success",
	})
}
func Fail(c *gin.Context, code int, msg string) {
	httpStatus := GetHTTPStatus(code)
	c.JSON(httpStatus, Response[any]{
		Code:    code,
		Data:    nil,
		Message: msg,
	})
}
func GetHTTPStatus(code int) int {
	switch {
	// 身份认证类错误 -> 401 Unauthorized
	// 包含：未登录、Token无效、Token过期
	case code == error_code.NotLoginCode || code == error_code.InvalidTokenCode || code == error_code.TokenOutErrorCode:
		return http.StatusUnauthorized

		// 限流/频率控制 -> 429 Too Many Requests
	case code == error_code.RequestTooFrequentCode:
		return http.StatusTooManyRequests
	// 权限类错误 -> 403 Forbidden
	// 包含：权限不足、不是设备拥有者、设备未绑定等
	case code == error_code.NoPermissionCode || (code >= 30000 && code < 40000):
		return http.StatusForbidden

	// 业务逻辑/参数错误 -> 400 Bad Request
	// 包含：用户已存在、密码太简单、验证码错误、参数绑定失败
	case code >= 20000 && code < 30000:
		return http.StatusBadRequest

	// 服务器内部错误 -> 500 Internal Server Error
	// 包含：系统错误、数据库崩溃
	case code >= 10000 && code < 20000:
		return http.StatusInternalServerError

	// 默认返回 400
	default:
		return http.StatusBadRequest
	}
}
