package types_test

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestHash384IsZero 测试零哈希检测。
func TestHash384IsZero(t *testing.T) {
	cases := []struct {
		name string
		h    types.Hash384
		want bool
	}{
		{"zero hash", types.Hash384{}, true},
		{"non-zero hash", types.Hash384{1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.h.IsZero(); got != tc.want {
				t.Errorf("IsZero() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestHash512IsZero 测试 Hash512 零哈希检测。
func TestHash512IsZero(t *testing.T) {
	cases := []struct {
		name string
		h    types.Hash512
		want bool
	}{
		{"zero hash", types.Hash512{}, true},
		{"non-zero hash", types.Hash512{1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.h.IsZero(); got != tc.want {
				t.Errorf("IsZero() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestHash256IsZero 测试 Hash256 零哈希检测。
func TestHash256IsZero(t *testing.T) {
	cases := []struct {
		name string
		h    types.Hash256
		want bool
	}{
		{"zero hash", types.Hash256{}, true},
		{"non-zero hash", types.Hash256{1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.h.IsZero(); got != tc.want {
				t.Errorf("IsZero() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestPubKeyHashIsZero 测试 PubKeyHash 零哈希检测。
func TestPubKeyHashIsZero(t *testing.T) {
	cases := []struct {
		name string
		h    types.PubKeyHash
		want bool
	}{
		{"zero hash", types.PubKeyHash{}, true},
		{"non-zero hash", types.PubKeyHash{1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.h.IsZero(); got != tc.want {
				t.Errorf("IsZero() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestHashString 测试哈希类型 String 方法返回十六进制字符串。
func TestHashString(t *testing.T) {
	h384 := types.Hash384{0xab, 0xcd}
	if s := h384.String(); len(s) != 96 {
		t.Errorf("Hash384.String() length = %d, want 96", len(s))
	}
	h512 := types.Hash512{0xef}
	if s := h512.String(); len(s) != 128 {
		t.Errorf("Hash512.String() length = %d, want 128", len(s))
	}
	h256 := types.Hash256{0x12}
	if s := h256.String(); len(s) != 64 {
		t.Errorf("Hash256.String() length = %d, want 64", len(s))
	}
}

// TestAddressEncodeDecodeRoundtrip 测试地址编解码往返一致性。
func TestAddressEncodeDecodeRoundtrip(t *testing.T) {
	pkh := types.PubKeyHash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	addr := types.NewAddress(pkh, types.MainnetPrefix)
	encoded := addr.Encode()

	decoded, err := types.DecodeAddress(encoded, types.MainnetPrefix)
	if err != nil {
		t.Fatalf("DecodeAddress() error = %v", err)
	}
	if decoded.PKHash() != pkh {
		t.Errorf("PKHash mismatch: got %v, want %v", decoded.PKHash(), pkh)
	}
}

// TestAddressChecksumMismatch 测试错误校验码被正确拒绝。
func TestAddressChecksumMismatch(t *testing.T) {
	pkh := types.PubKeyHash{0xff}
	addr := types.NewAddress(pkh, types.MainnetPrefix)
	encoded := addr.Encode()
	// 篡改最后一个字节
	tampered := encoded[:len(encoded)-1] + "X"
	_, err := types.DecodeAddress(tampered, types.MainnetPrefix)
	if err == nil {
		t.Error("expected error for tampered address, got nil")
	}
}

// TestAddressWrongPrefix 测试前缀不匹配被拒绝。
func TestAddressWrongPrefix(t *testing.T) {
	pkh := types.PubKeyHash{0x01}
	addr := types.NewAddress(pkh, types.MainnetPrefix)
	encoded := addr.Encode()
	_, err := types.DecodeAddress(encoded, types.TestnetPrefix)
	if err == nil {
		t.Error("expected error for wrong prefix, got nil")
	}
}
