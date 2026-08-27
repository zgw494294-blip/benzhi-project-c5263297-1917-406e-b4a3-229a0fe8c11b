package domain

import (
	"errors"
	"strings"
)

type ValidationError struct{ Issues []ValidationIssue }

func (e ValidationError) Error() string {
	parts := make([]string, 0, len(e.Issues))
	for _, x := range e.Issues {
		parts = append(parts, x.Field+"："+x.Message)
	}
	return strings.Join(parts, "；")
}
func (e ValidationError) Unwrap() error { return ErrInvalidInput }

var (
	ErrNotFound         = errors.New("资源不存在")
	ErrVersionConflict  = errors.New("版本冲突")
	ErrInvalidState     = errors.New("状态不允许此操作")
	ErrDuplicatePoint   = errors.New("吊点编号重复")
	ErrInvalidInput     = errors.New("输入不合法")
	ErrBlockingFindings = errors.New("仍有未解决阻断问题")
	ErrStaleAssessment  = errors.New("核验结论已过期")
	ErrReviewOrder      = errors.New("复核顺序不正确")
	ErrFrozen           = errors.New("方案已冻结")
)
