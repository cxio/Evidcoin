package rewards

import "errors"

// internal/rewards 包错误定义（第 14 章 / DEC-0401）。

var (
	// ErrReceiverCountMismatch 表示接收地址数量与奖励输出数量不一致。
	ErrReceiverCountMismatch = errors.New("rewards: receiver count does not match reward output count")
	// ErrInvalidServiceIndex 表示服务索引超出 [0,2] 范围（Blockqs/Depots/STUN）。
	ErrInvalidServiceIndex = errors.New("rewards: service index out of range [0,2]")
	// ErrWindowTooShort 表示兑奖窗口块数不足 48。
	ErrWindowTooShort = errors.New("rewards: award window must contain exactly 48 slots")
)
