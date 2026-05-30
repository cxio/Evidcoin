package types

import "encoding/binary"

// 固定宽度大端整数编码（DEC-0001 §1.1）。适用于定宽字段白名单中的整数字段。

// AppendUint16BE appends v as a 2-byte big-endian integer to dst.
func AppendUint16BE(dst []byte, v uint16) []byte {
	return binary.BigEndian.AppendUint16(dst, v)
}

// AppendUint32BE appends v as a 4-byte big-endian integer to dst.
func AppendUint32BE(dst []byte, v uint32) []byte {
	return binary.BigEndian.AppendUint32(dst, v)
}

// AppendUint64BE appends v as an 8-byte big-endian integer to dst.
func AppendUint64BE(dst []byte, v uint64) []byte {
	return binary.BigEndian.AppendUint64(dst, v)
}

// AppendInt64BE appends v as an 8-byte big-endian two's-complement integer.
// Signed protocol fields (e.g. transaction header Timestamp) use fixed-width
// big-endian, never signed varint (DEC-0001 §1.2).
func AppendInt64BE(dst []byte, v int64) []byte {
	return binary.BigEndian.AppendUint64(dst, uint64(v))
}

// ReadUint16BE reads a 2-byte big-endian integer from the front of src.
func ReadUint16BE(src []byte) (uint16, int, error) {
	if len(src) < 2 {
		return 0, 0, ErrShortBuffer
	}
	return binary.BigEndian.Uint16(src), 2, nil
}

// ReadUint32BE reads a 4-byte big-endian integer from the front of src.
func ReadUint32BE(src []byte) (uint32, int, error) {
	if len(src) < 4 {
		return 0, 0, ErrShortBuffer
	}
	return binary.BigEndian.Uint32(src), 4, nil
}

// ReadUint64BE reads an 8-byte big-endian integer from the front of src.
func ReadUint64BE(src []byte) (uint64, int, error) {
	if len(src) < 8 {
		return 0, 0, ErrShortBuffer
	}
	return binary.BigEndian.Uint64(src), 8, nil
}

// ReadInt64BE reads an 8-byte big-endian two's-complement integer from src.
func ReadInt64BE(src []byte) (int64, int, error) {
	if len(src) < 8 {
		return 0, 0, ErrShortBuffer
	}
	return int64(binary.BigEndian.Uint64(src)), 8, nil
}
