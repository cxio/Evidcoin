// Package tx 定义 Evidcoin 的交易数据结构与哈希计算。
package tx

import (
	"encoding/binary"
	"errors"
)

// AppendVarint 将 v 编码为 varint 并追加到 buf，返回新的 buf。
// 使用 protobuf 风格的 varint 编码（每字节 7 位有效，最高位为延续标志）。
func AppendVarint(buf []byte, v uint64) []byte {
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	return append(buf, byte(v))
}

// ReadVarint 从 data 读取一个 varint，返回值、消耗字节数与错误。
func ReadVarint(data []byte) (uint64, int, error) {
	var v uint64
	var s uint
	for i, b := range data {
		if i == 10 {
			return 0, 0, errors.New("varint overflow")
		}
		if b < 0x80 {
			v |= uint64(b) << s
			return v, i + 1, nil
		}
		v |= uint64(b&0x7f) << s
		s += 7
	}
	return 0, 0, errors.New("varint truncated")
}

// EncodeInt64 将 int64 编码为大端 8 字节序列（用于铸凭哈希计算）。
func EncodeInt64(v int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}
