package types

import (
	"fmt"
	"strings"
)

// Amount 表示协议层金额，单位为 chx（最小不可分单位）。
// 所有链上金额（铸凭、手续费、销毁、奖励、UTXO 输出）均以 chx 整数存储和计算；
// 绝不能使用 float64 表示金额。
type Amount uint64

// Bi（= coin）仅用于展示/换算：1 Bi = ChxPerBi chx。

// String 将金额格式化为易读的 Bi 字符串，固定 8 位小数，
// 例如 Amount(ChxPerBi) -> "1.00000000 Bi"。
func (a Amount) String() string {
	whole := uint64(a) / ChxPerBi
	frac := uint64(a) % ChxPerBi
	return fmt.Sprintf("%d.%08d Bi", whole, frac)
}

// ParseAmount 将十进制 Bi 字符串（如 "1.5"、"0.00000001"、"12"）解析为 chx Amount。
// 小数位最多 8 位。超过 uint64 chx 范围或格式非法的输入将被拒绝。
// 解析过程中不使用浮点运算，因此换算结果精确无误。
func ParseAmount(s string) (Amount, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrAmountFormat
	}
	intPart, fracPart, hasFrac := strings.Cut(s, ".")
	if intPart == "" && fracPart == "" {
		return 0, ErrAmountFormat
	}
	if hasFrac && len(fracPart) > 8 {
		return 0, ErrAmountFormat
	}

	whole, err := parseDigits(intPart, true)
	if err != nil {
		return 0, err
	}
	// chx = whole*ChxPerBi + frac（frac 右侧补零到 8 位）。
	const maxBeforeMul = ^uint64(0) / ChxPerBi
	if whole > maxBeforeMul {
		return 0, ErrAmountOverflow
	}
	chx := whole * ChxPerBi

	if hasFrac && fracPart != "" {
		padded := fracPart + strings.Repeat("0", 8-len(fracPart))
		frac, err := parseDigits(padded, false)
		if err != nil {
			return 0, err
		}
		if chx > ^uint64(0)-frac {
			return 0, ErrAmountOverflow
		}
		chx += frac
	}
	return Amount(chx), nil
}

// parseDigits 将非负十进制数字字符串解析为 uint64。
// allowEmpty 允许空串（按 0 处理），用于 ".5" 这类值的整数部分。
func parseDigits(s string, allowEmpty bool) (uint64, error) {
	if s == "" {
		if allowEmpty {
			return 0, nil
		}
		return 0, ErrAmountFormat
	}
	var v uint64
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, ErrAmountFormat
		}
		d := uint64(c - '0')
		if v > (^uint64(0)-d)/10 {
			return 0, ErrAmountOverflow
		}
		v = v*10 + d
	}
	return v, nil
}
