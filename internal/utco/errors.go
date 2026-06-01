package utco

import "errors"

// UTCO 状态层错误定义（第 09 章、DEC-0201）。

var (
	// ErrNotFound 表示按完整 OutPoint 查询或转移时未命中状态集。
	ErrNotFound = errors.New("utco: outpoint not found")
	// ErrDuplicate 表示插入的 OutPoint 已存在于状态集。
	ErrDuplicate = errors.New("utco: outpoint already exists")
	// ErrAlreadySpent 表示对已转出的凭信再次转移（违反一次性转移）。
	ErrAlreadySpent = errors.New("utco: credit already transferred")
)
