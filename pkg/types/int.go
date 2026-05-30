package types

import "encoding/binary"

// 固定宽度大端整数编码（DEC-0001 §1.1）。适用于定宽字段白名单中的整数字段。

// AppendUint16BE 将 v 以 2 字节大端整数形式追加到 dst。
func AppendUint16BE(dst []byte, v uint16) []byte {
	return binary.BigEndian.AppendUint16(dst, v)
}

// AppendUint32BE 将 v 以 4 字节大端整数形式追加到 dst。
func AppendUint32BE(dst []byte, v uint32) []byte {
	return binary.BigEndian.AppendUint32(dst, v)
}

// AppendUint64BE 将 v 以 8 字节大端整数形式追加到 dst。
func AppendUint64BE(dst []byte, v uint64) []byte {
	return binary.BigEndian.AppendUint64(dst, v)
}

// AppendInt64BE 将 v 以 8 字节大端二进制补码整数形式追加到 dst。
// 协议中的有符号字段（如交易头 Timestamp）使用定宽大端编码，
// 不使用有符号 varint（DEC-0001 §1.2）。
func AppendInt64BE(dst []byte, v int64) []byte {
	return binary.BigEndian.AppendUint64(dst, uint64(v))
}

// ReadUint16BE 从 src 前缀读取 2 字节大端整数。
func ReadUint16BE(src []byte) (uint16, int, error) {
	if len(src) < 2 {
		return 0, 0, ErrShortBuffer
	}
	return binary.BigEndian.Uint16(src), 2, nil
}

// ReadUint32BE 从 src 前缀读取 4 字节大端整数。
func ReadUint32BE(src []byte) (uint32, int, error) {
	if len(src) < 4 {
		return 0, 0, ErrShortBuffer
	}
	return binary.BigEndian.Uint32(src), 4, nil
}

// ReadUint64BE 从 src 前缀读取 8 字节大端整数。
func ReadUint64BE(src []byte) (uint64, int, error) {
	if len(src) < 8 {
		return 0, 0, ErrShortBuffer
	}
	return binary.BigEndian.Uint64(src), 8, nil
}

// ReadInt64BE 从 src 前缀读取 8 字节大端二进制补码整数。
func ReadInt64BE(src []byte) (int64, int, error) {
	if len(src) < 8 {
		return 0, 0, ErrShortBuffer
	}
	return int64(binary.BigEndian.Uint64(src)), 8, nil
}
