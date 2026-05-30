package types

import "math/big"

// BigInt 二进制序列化（DEC-0001 §3）。布局：slen || magnitude。
// slen 的 bit7 为符号位（0 正 / 1 负），低 7 位为 magnitude 字节长度（0–127）。
// magnitude 为绝对值的大端无符号最短表示；零值编码为 slen=0x00 且空 magnitude。

// AppendBigInt 按规范 BigInt 字节布局将 x 追加到 dst。
// 当绝对值长度超过 127 字节时返回错误。
func AppendBigInt(dst []byte, x *big.Int) ([]byte, error) {
	mag := new(big.Int).Abs(x).Bytes() // 大端、无前导零；零值为空切片
	if len(mag) > 127 {
		return nil, ErrBigIntTooLarge
	}
	slen := byte(len(mag))
	if x.Sign() < 0 {
		slen |= 0x80
	}
	dst = append(dst, slen)
	return append(dst, mag...), nil
}

// ReadBigInt 从 src 前缀读取规范 BigInt，返回解析值与已消费字节数。
// 对非最短绝对值表示（前导零）以及负零编码一律拒绝。
func ReadBigInt(src []byte) (*big.Int, int, error) {
	if len(src) < 1 {
		return nil, 0, ErrShortBuffer
	}
	slen := src[0]
	neg := slen&0x80 != 0
	n := int(slen & 0x7f)
	end := 1 + n
	if end > len(src) {
		return nil, 0, ErrShortBuffer
	}
	mag := src[1:end]
	if n > 0 && mag[0] == 0x00 {
		return nil, 0, ErrBigIntNotMinimal
	}
	if n == 0 && neg {
		return nil, 0, ErrBigIntNegativeZero
	}
	x := new(big.Int).SetBytes(mag)
	if neg {
		x.Neg(x)
	}
	return x, end, nil
}
