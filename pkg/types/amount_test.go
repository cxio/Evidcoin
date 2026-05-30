package types

import (
	"math"
	"testing"
)

func TestAmountConstants(t *testing.T) {
	if ChxPerBi != 100_000_000 {
		t.Fatalf("ChxPerBi = %d, want 100000000", ChxPerBi)
	}
}

func TestAmountString(t *testing.T) {
	tests := []struct {
		a    Amount
		want string
	}{
		{0, "0.00000000 Bi"},
		{1, "0.00000001 Bi"},
		{ChxPerBi, "1.00000000 Bi"},
		{ChxPerBi + 50_000_000, "1.50000000 Bi"},
		{math.MaxUint64, "184467440737.09551615 Bi"},
	}
	for _, tt := range tests {
		if got := tt.a.String(); got != tt.want {
			t.Errorf("Amount(%d).String() = %q, want %q", uint64(tt.a), got, tt.want)
		}
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		in   string
		want Amount
	}{
		{"0", 0},
		{"1", ChxPerBi},
		{"1.5", ChxPerBi + 50_000_000},
		{"0.00000001", 1},
		{"12.34567890", 12*ChxPerBi + 34_567_890},
		{".5", 50_000_000},
	}
	for _, tt := range tests {
		got, err := ParseAmount(tt.in)
		if err != nil {
			t.Errorf("ParseAmount(%q) error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseAmount(%q) = %d, want %d", tt.in, uint64(got), uint64(tt.want))
		}
	}
}

func TestParseAmountRejects(t *testing.T) {
	tests := []string{
		"",
		"abc",
		"1.234567890",       // 9 fractional digits
		"1.2.3",             // multiple separators
		"99999999999999999", // overflow whole part
		"-1",                // sign not allowed
	}
	for _, in := range tests {
		if _, err := ParseAmount(in); err == nil {
			t.Errorf("ParseAmount(%q) accepted, want error", in)
		}
	}
}
