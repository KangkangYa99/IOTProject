package response

import (
	"github.com/gin-gonic/gin"
	"net/http"
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
	if code == 20010 || code == 20012 { // NotLoginCode 和 InvalidTokenCode
		return http.StatusUnauthorized
	} else if code >= 20000 && code < 30000 {
		return http.StatusUnauthorized
	} else if code >= 30000 && code < 40000 {
		return http.StatusForbidden
	} else if code >= 10000 && code < 20000 {
		return http.StatusInternalServerError
	}
	return http.StatusBadRequest
}
