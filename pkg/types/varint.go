package types

// maxVarUintLen 是 uint64 ULEB128 变长整数可使用的最大字节数。
const maxVarUintLen = 10

// AppendVarUint 将 v 以规范 ULEB128 无符号变长整数格式追加到 dst。
// 按 DEC-0001 §1.2：每个字节承载 7 位数据，高位 bit 表示是否续接，
// 且从低有效位分组开始编码。编码始终采用最短形式。
func AppendVarUint(dst []byte, v uint64) []byte {
	for v >= 0x80 {
		dst = append(dst, byte(v)|0x80)
		v >>= 7
	}
	return append(dst, byte(v))
}

// ReadVarUint 从 src 前缀读取一个规范 ULEB128 无符号变长整数。
// 返回解码值与已消费字节数。对非最短编码（冗余高位零组）、
// 数值溢出与截断输入一律拒绝，不做静默容错（DEC-0001 边界）。
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
