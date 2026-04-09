package error_code

import (
	"errors"
	"fmt"
)

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	ServerErrorCode = 10000 + iota
	DatabaseErrorCode
	ServiceUnavailableCode
)
const (
	UserExistsCode = 20000 + iota
	PasswordIsEasyCode
	UserNotExistsCode
	InvalidParamCode
	UserNumberExistsCode
	UserEmailExistsCode
	PasswordFailCode
	UserDataAuthFailCode
	OldPasswordFailCode
	PassWordSameCode
	CheckPhoneFailCode
	DeviceIDNotFoundCode
	NotLoginCode
	NoPermissionCode
	InvalidTokenCode
	ShouldBindErrorCode
	TokenOutErrorCode
	RequestTooFrequentCode
	CodeFailCode
	InvalidFileTypeCode
	InvalidParamsCode
	FileNotFoundCode
	FileTooLargeCode
	FileUploadFailCode
	DevicePolicyIsExistCode
)

var (
	ErrUserExists          = errors.New("user already exists")
	ErrNotFound            = errors.New("record not found")
	ErrDB                  = errors.New("database error")
	ErrDeviceAlreadyExists = errors.New("device is registered")
	ErrInvalidSN           = errors.New("invalid UID")
	ErrDeviceNotFound      = errors.New("device not found")
)

const (
	DeviceNotFoundCode = 30000 + iota
	DeviceIsBindCode
	DeviceNotBindCode
	NotDeviceOwnerCode
	DeviceTypeFailCode
)

var (
	ServerError        = &APIError{Code: ServerErrorCode, Message: "服务器内部错误。"}
	ServiceUnavailable = &APIError{Code: ServiceUnavailableCode, Message: "服务器繁忙，请稍后再试。"}
	DatabaseError      = &APIError{Code: DatabaseErrorCode, Message: "数据库操作失败。"}
	UserExists         = &APIError{Code: UserExistsCode, Message: "用户已注册。"}
	InvalidParam       = &APIError{Code: InvalidParamCode, Message: "非法参数"}
	UserNotExists      = &APIError{Code: UserNotExistsCode, Message: "用户不存在。"}
	PasswordIsEasy     = &APIError{Code: PasswordIsEasyCode, Message: "密码过于简单。"}
	UserNumberExists   = &APIError{Code: UserNumberExistsCode, Message: "手机号已注册。"}
	UserEmailExists    = &APIError{Code: UserEmailExistsCode, Message: "邮箱已注册。"}
	UserDataAuthFail   = &APIError{Code: UserDataAuthFailCode, Message: "用户数据资料认证错误。"}

	CodeFail                = &APIError{Code: CodeFailCode, Message: "验证码错误，请重新输入。"}
	PasswordFail            = &APIError{Code: PasswordFailCode, Message: "账号或密码错误。"}
	OldPasswordFail         = &APIError{Code: OldPasswordFailCode, Message: "旧密码错误。"}
	PassWordSame            = &APIError{Code: PassWordSameCode, Message: "新密码与旧密码相同。"}
	DeviceIDNotFound        = &APIError{Code: DeviceIDNotFoundCode, Message: "获取验证码的设备ID不能为空。"}
	DeviceNotFound          = &APIError{Code: DeviceNotFoundCode, Message: "设备未注册。"}
	DeviceNotBind           = &APIError{Code: DeviceNotBindCode, Message: "设备未被绑定。"}
	DeviceIsBind            = &APIError{Code: DeviceIsBindCode, Message: "设备已被其他用户绑定。"}
	NotDeviceOwner          = &APIError{Code: NotDeviceOwnerCode, Message: "您不是设备拥有者。"}
	CheckPhoneFail          = &APIError{Code: CheckPhoneFailCode, Message: "手机号验证失败。"}
	NotLogin                = &APIError{Code: NotLoginCode, Message: "用户未登录。"}
	NoPermission            = &APIError{Code: NoPermissionCode, Message: "权限不足。"}
	InvalidToken            = &APIError{Code: InvalidTokenCode, Message: "无效的Token格式。"}
	TokenOutError           = &APIError{Code: TokenOutErrorCode, Message: "Token已被注销。"}
	ShouldBindError         = &APIError{Code: ShouldBindErrorCode, Message: "绑定参数错误。"}
	RequestTooFrequentError = &APIError{Code: RequestTooFrequentCode, Message: "提交过于频繁，稍后再试。"}
	DeviceTypeFailError     = &APIError{Code: DeviceTypeFailCode, Message: "不支持的传感器类型。"}
	InvalidFileType         = &APIError{Code: InvalidFileTypeCode, Message: "不支持的文件类型。"}
	InvalidParams           = &APIError{Code: InvalidParamsCode, Message: "参数错误。"}
	FileNotFound            = &APIError{Code: FileNotFoundCode, Message: "文件未找到。"}
	FileTooLarge            = &APIError{Code: FileTooLargeCode, Message: "文件大小超过限制。"}
	FileUploadFail          = &APIError{Code: FileUploadFailCode, Message: "文件上传失败，请重试。"}
	DevicePolicyIsExist     = &APIError{Code: DevicePolicyIsExistCode, Message: "该策略已经存在"}
)

func NewAPIError(code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}
func (e *APIError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}
func (e *APIError) SetError() *APIError {
	return &APIError{
		Code:    e.Code,
		Message: e.Message,
	}

}
func (e *APIError) WithDetails(details string) *APIError {
	return &APIError{
		Code:    e.Code,
		Message: fmt.Sprintf("%s (%s)", e.Message, details),
	}
}
