package types

// maxVarUintLen is the maximum number of bytes a uint64 ULEB128 varint can use.
const maxVarUintLen = 10

// AppendVarUint appends v to dst as a canonical ULEB128 unsigned varint
// (DEC-0001 §1.2): each byte carries 7 data bits with the high bit signalling
// continuation, least-significant group first. The encoding is always shortest.
func AppendVarUint(dst []byte, v uint64) []byte {
	for v >= 0x80 {
		dst = append(dst, byte(v)|0x80)
		v >>= 7
	}
	return append(dst, byte(v))
}

// ReadVarUint reads a canonical ULEB128 unsigned varint from the front of src.
// It returns the decoded value and the number of bytes consumed. Non-minimal
// encodings (redundant trailing zero group), overflowing values and truncated
// inputs are rejected, never silently accepted (DEC-0001 边界).
func ReadVarUint(src []byte) (value uint64, n int, err error) {
	var shift uint
	for i := 0; i < len(src); i++ {
		b := src[i]
		if i == maxVarUintLen-1 {
			// 第 10 字节最多承载 1 位（9*7=63），且必须是终止字节。
			if b > 0x01 {
				return 0, 0, ErrVarintOverflow
			}
		}
		value |= uint64(b&0x7f) << shift
		if b < 0x80 {
			// 终止字节为 0 且非首字节 => 存在冗余高位零组，非最短编码。
			if b == 0 && i > 0 {
				return 0, 0, ErrVarintNotMinimal
			}
			return value, i + 1, nil
		}
		shift += 7
	}
	// 读完所有字节仍未遇到终止字节。
	if len(src) >= maxVarUintLen {
		return 0, 0, ErrVarintTooLong
	}
	return 0, 0, ErrVarintTruncated
}
