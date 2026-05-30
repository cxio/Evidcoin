package types

import "math/big"

// BigInt 二进制序列化（DEC-0001 §3）。布局：slen || magnitude。
// slen 的 bit7 为符号位（0 正 / 1 负），低 7 位为 magnitude 字节长度（0–127）。
// magnitude 为绝对值的大端无符号最短表示；零值编码为 slen=0x00 且空 magnitude。

// AppendBigInt appends x to dst using the canonical BigInt byte layout.
// It returns an error when the magnitude exceeds 127 bytes.
func AppendBigInt(dst []byte, x *big.Int) ([]byte, error) {
	mag := new(big.Int).Abs(x).Bytes() // big-endian, no leading zeros; empty for 0
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

// ReadBigInt reads a canonical BigInt from the front of src and returns the
// value together with the number of bytes consumed. Non-minimal magnitudes
// (leading zero) and negative-zero encodings are rejected.
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
