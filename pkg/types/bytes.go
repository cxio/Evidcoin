package types

// 可变长度字节序列、列表与可选字段编码（DEC-0001 §1.3）。

// AppendBytes 将 b 作为“长度前缀 + 数据”的字节序列追加到 dst：
// varint(len(b)) || b。空切片编码为单字节长度 0x00。
func AppendBytes(dst, b []byte) []byte {
	dst = AppendVarUint(dst, uint64(len(b)))
	return append(dst, b...)
}

// ReadBytes 从 src 前缀读取一个带长度前缀的字节序列。
// 返回切片为新副本，不与 src 共享底层内存。
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

// AppendOptional 追加一个可选值：缺失时写入 0x00；存在时写入 0x01，
// 再追加由 appendValue 生成的值（DEC-0001 optional 编码）。
func AppendOptional(dst []byte, present bool, appendValue func([]byte) []byte) []byte {
	if !present {
		return append(dst, 0x00)
	}
	dst = append(dst, 0x01)
	return appendValue(dst)
}

// ReadOptionalMarker 从 src 读取一个可选字段存在标记，并报告后续是否有值。
// 标记不是 0x00/0x01 时视为非法并拒绝。
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

// AppendList 追加列表编码：先写 varint(count)，再写每个由 appendElem 生成的元素。
// 元素的具体编码规则由调用方负责。
func AppendList[T any](dst []byte, items []T, appendElem func([]byte, T) []byte) []byte {
	dst = AppendVarUint(dst, uint64(len(items)))
	for i := range items {
		dst = appendElem(dst, items[i])
	}
	return dst
}
