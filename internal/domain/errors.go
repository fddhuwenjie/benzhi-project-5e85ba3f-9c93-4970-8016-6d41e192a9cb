package domain

import "errors"

var (
	ErrNotFound         = errors.New("试验档案不存在")
	ErrConflict         = errors.New("revision 冲突，请刷新后重试")
	ErrIdempotency      = errors.New("request_id 已用于其他操作")
	ErrInvalidState     = errors.New("当前状态不允许此操作")
	ErrArchived         = errors.New("已放行档案为只读，禁止修改")
	ErrValidation       = errors.New("输入校验失败")
	ErrUnauthorizedRole = errors.New("当前角色无权执行此操作")
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

func Invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
