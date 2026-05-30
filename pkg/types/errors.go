package types

import (
	"errors"
	"fmt"
)

// 编码与类型错误。协议解码器必须拒绝非规范输入，而非容忍后重编码。

var (
	// ErrShortBuffer is returned when a fixed-width decode runs out of input.
	ErrShortBuffer = errors.New("types: buffer too short")
	// ErrVarintTooLong is returned when a varint exceeds the 10-byte uint64 bound.
	ErrVarintTooLong = errors.New("types: varint too long")
	// ErrVarintOverflow is returned when a varint does not fit into a uint64.
	ErrVarintOverflow = errors.New("types: varint overflow")
	// ErrVarintTruncated is returned when a varint lacks its terminating byte.
	ErrVarintTruncated = errors.New("types: varint truncated")
	// ErrVarintNotMinimal is returned when a varint is not shortest-encoded.
	ErrVarintNotMinimal = errors.New("types: varint not minimal")
	// ErrBigIntTooLarge is returned when a BigInt magnitude exceeds 127 bytes.
	ErrBigIntTooLarge = errors.New("types: bigint magnitude exceeds 127 bytes")
	// ErrBigIntNotMinimal is returned when a BigInt magnitude has a leading zero.
	ErrBigIntNotMinimal = errors.New("types: bigint magnitude not minimal")
	// ErrBigIntNegativeZero is returned when a BigInt encodes a negative zero.
	ErrBigIntNegativeZero = errors.New("types: bigint negative zero")
	// ErrInvalidOptionalMarker is returned for an optional marker other than 0x00/0x01.
	ErrInvalidOptionalMarker = errors.New("types: invalid optional marker")
	// ErrAmountOverflow is returned when a parsed amount exceeds uint64 range.
	ErrAmountOverflow = errors.New("types: amount overflow")
	// ErrAmountFormat is returned for a malformed decimal amount string.
	ErrAmountFormat = errors.New("types: invalid amount format")
)

// lengthError reports a fixed-length construction mismatch.
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
