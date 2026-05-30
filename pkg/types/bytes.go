package types

// 可变长度字节序列、列表与可选字段编码（DEC-0001 §1.3）。

// AppendBytes appends b to dst as a length-prefixed byte sequence:
// varint(len(b)) || b. An empty slice encodes as a single 0x00 length.
func AppendBytes(dst, b []byte) []byte {
	dst = AppendVarUint(dst, uint64(len(b)))
	return append(dst, b...)
}

// ReadBytes reads a length-prefixed byte sequence from the front of src.
// The returned slice is a fresh copy and does not alias src.
func ReadBytes(src []byte) (b []byte, n int, err error) {
	length, ln, err := ReadVarUint(src)
	if err != nil {
		return nil, 0, err
	}
	end := ln + int(length)
	if end < ln || end > len(src) {
		return nil, 0, ErrShortBuffer
	}
	out := make([]byte, length)
	copy(out, src[ln:end])
	return out, end, nil
}

// AppendOptional appends an optional value: 0x00 when absent, or 0x01 followed
// by the value produced by appendValue when present (DEC-0001 optional 编码).
func AppendOptional(dst []byte, present bool, appendValue func([]byte) []byte) []byte {
	if !present {
		return append(dst, 0x00)
	}
	dst = append(dst, 0x01)
	return appendValue(dst)
}

// ReadOptionalMarker reads a single optional presence marker from src and
// reports whether a value follows. A marker other than 0x00/0x01 is rejected.
func ReadOptionalMarker(src []byte) (present bool, n int, err error) {
	if len(src) < 1 {
		return false, 0, ErrShortBuffer
	}
	switch src[0] {
	case 0x00:
		return false, 1, nil
	case 0x01:
		return true, 1, nil
	default:
		return false, 0, ErrInvalidOptionalMarker
	}
}

// AppendList appends a list: varint(count) followed by each element as produced
// by appendElem. The caller is responsible for element encoding.
func AppendList[T any](dst []byte, items []T, appendElem func([]byte, T) []byte) []byte {
	dst = AppendVarUint(dst, uint64(len(items)))
	for i := range items {
		dst = appendElem(dst, items[i])
	}
	return dst
}
