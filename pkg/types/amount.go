package types

import (
	"fmt"
	"strings"
)

// Amount represents a protocol-layer monetary value in chx, the smallest
// indivisible unit. All on-chain amounts (mint, fee, burn, reward, UTXO output)
// are carried and computed as chx integers; float64 must never represent money.
type Amount uint64

// Bi (= coin) is a display/conversion unit only: 1 Bi = ChxPerBi chx.

// String renders the amount in human-readable Bi with 8 fractional digits,
// e.g. Amount(ChxPerBi) -> "1.00000000 Bi".
func (a Amount) String() string {
	whole := uint64(a) / ChxPerBi
	frac := uint64(a) % ChxPerBi
	return fmt.Sprintf("%d.%08d Bi", whole, frac)
}

// ParseAmount parses a decimal Bi string (e.g. "1.5", "0.00000001", "12") into
// a chx Amount. At most 8 fractional digits are allowed. Values exceeding the
// uint64 chx range and malformed inputs are rejected. No float arithmetic is
// used so the conversion is exact.
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
	// chx = whole*ChxPerBi + frac (frac padded to 8 digits).
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

// parseDigits parses a non-negative decimal digit string into uint64.
// allowEmpty permits an empty string (treated as 0) for the integer part of
// values like ".5".
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
