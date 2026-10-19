package errs

import (
	"errors"
	"log/slog"
)

type CodeError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *CodeError) Error() string {
	return e.Message
}

func NewError(code int, message string) *CodeError {
	return &CodeError{Code: code, Message: message}
}

func FromError(err error) *CodeError {
	if err == nil {
		return nil
	}
	var ce *CodeError
	if errors.As(err, &ce) {
		return ce
	}
	// 内部错误只落日志，响应统一返回通用文案，避免泄露 SQL/驱动细节
	slog.Error("内部错误", "err", err)
	return NewError(CodeInternalError, "服务器内部错误")
}

const (
	CodeSuccess         = 0
	CodeInternalError   = 500
	CodeParamsError     = 400
	CodeNotLogin        = 401
	CodeNoPermission    = 403
	CodeNotFound        = 404
	CodeTooManyRequests = 429
)

func ParamsError(message string) *CodeError {
	return NewError(CodeParamsError, message)
}

func NotLogin() *CodeError {
	return NewError(CodeNotLogin, "请先登录")
}

func NoPermission() *CodeError {
	return NewError(CodeNoPermission, "无权限")
}

func NotFound(message string) *CodeError {
	return NewError(CodeNotFound, message)
}

func TooManyRequests() *CodeError {
	return NewError(CodeTooManyRequests, "操作太频繁，请稍后再试")
}
