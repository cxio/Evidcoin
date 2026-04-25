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
