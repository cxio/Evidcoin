package types

import (
	"errors"
	"fmt"
)

// 编码与类型错误。协议解码器必须拒绝非规范输入，而非容忍后重编码。

var (
	// ErrShortBuffer 表示定宽解码时输入字节不足。
	ErrShortBuffer = errors.New("types: buffer too short")
	// ErrVarintTooLong 表示变长整数超过 uint64 的 10 字节上限。
	ErrVarintTooLong = errors.New("types: varint too long")
	// ErrVarintOverflow 表示变长整数数值无法装入 uint64。
	ErrVarintOverflow = errors.New("types: varint overflow")
	// ErrVarintTruncated 表示变长整数缺少终止字节。
	ErrVarintTruncated = errors.New("types: varint truncated")
	// ErrVarintNotMinimal 表示变长整数未采用最短编码。
	ErrVarintNotMinimal = errors.New("types: varint not minimal")
	// ErrBigIntTooLarge 表示 BigInt 绝对值长度超过 127 字节。
	ErrBigIntTooLarge = errors.New("types: bigint magnitude exceeds 127 bytes")
	// ErrBigIntNotMinimal 表示 BigInt 绝对值存在前导零，非最短表示。
	ErrBigIntNotMinimal = errors.New("types: bigint magnitude not minimal")
	// ErrBigIntNegativeZero 表示 BigInt 编码为负零。
	ErrBigIntNegativeZero = errors.New("types: bigint negative zero")
	// ErrInvalidOptionalMarker 表示可选字段标记不是 0x00 或 0x01。
	ErrInvalidOptionalMarker = errors.New("types: invalid optional marker")
	// ErrAmountOverflow 表示解析后的金额超出 uint64 范围。
	ErrAmountOverflow = errors.New("types: amount overflow")
	// ErrAmountFormat 表示金额十进制字符串格式非法。
	ErrAmountFormat = errors.New("types: invalid amount format")
)

// lengthError 表示定长类型构造时长度不匹配。
type lengthError struct {
	typ  string
	want int
	got  int
}

func newLengthError(typ string, want, got int) error {
	return &lengthError{typ: typ, want: want, got: got}
}

func (e *lengthError) Error() string {
	return fmt.Sprintf("types: %s requires %d bytes, got %d", e.typ, e.want, e.got)
}
