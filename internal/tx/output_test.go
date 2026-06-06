package tx

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestOutputConfigStandard 校验标准类输出 Config 字节布局：
// bit[3:0] 类型值，高 4 位无摘要标记时全为 0。
func TestOutputConfigStandard(t *testing.T) {
	tests := []struct {
		name string
		typ  OutputType
		want byte
	}{
		{"coin", TypeCoin, 0x01},
		{"credit", TypeCredit, 0x02},
		{"proof", TypeProof, 0x03},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := Output{Type: tc.typ}
			got, err := o.Config()
			if err != nil {
				t.Fatalf("Config: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Config = %#x, 期望 %#x", got, tc.want)
			}
			// 无摘要标记时高 4 位应全为 0。
			if got&0xF0 != 0 {
				t.Fatalf("无摘要标记时 Config 高 4 位应全为 0, got=%#x", got)
			}
		})
	}
}

// TestOutputConfigDigestFlags 校验摘要标记正确写入 Config 高 4 位（DEC-0101）。
func TestOutputConfigDigestFlags(t *testing.T) {
	tests := []struct {
		name  string
		flags uint8
		typ   OutputType
		want  byte
	}{
		{"content+script", DigestContent | DigestScript, TypeCoin, 0x61},
		{"account", DigestAccount, TypeCredit, 0x82},
		{"all three", DigestAccount | DigestContent | DigestScript, TypeProof, 0xE3},
		{"none", 0, TypeCoin, 0x01},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := Output{DigestFlags: tc.flags, Type: tc.typ}
			got, err := o.Config()
			if err != nil {
				t.Fatalf("Config: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Config = %#x, 期望 %#x", got, tc.want)
			}
		})
	}
}

// TestOutputDigestFlagsBit4Rejected 校验 bit4 置位被拒绝（未用位）。
func TestOutputDigestFlagsBit4Rejected(t *testing.T) {
	o := Output{DigestFlags: 0x10, Type: TypeCoin} // bit4 非法
	if _, err := o.Config(); err == nil {
		t.Fatal("bit4 置位应被拒绝")
	}
}

// TestOutputReservedTypeRejected 校验预留类型值 0（及未知类型）被拒绝。
func TestOutputReservedTypeRejected(t *testing.T) {
	if _, err := (Output{Type: TypeReserved}).Config(); err == nil {
		t.Fatal("预留类型值 0 应被拒绝")
	}
	if _, err := (Output{Type: OutputType(5)}).Config(); err == nil {
		t.Fatal("未知类型值 5 应被拒绝")
	}
}

// TestOutputInState 校验存证不进入 UTXO/UTCO，币金/凭信进入；摘要标记不影响归属。
func TestOutputInState(t *testing.T) {
	if !(Output{Type: TypeCoin}).InState() {
		t.Error("币金输出应进入 UTXO")
	}
	if !(Output{Type: TypeCredit}).InState() {
		t.Error("凭信输出应进入 UTCO")
	}
	if (Output{Type: TypeProof}).InState() {
		t.Error("存证输出不应进入状态集")
	}
	// 摘要标记不改变状态归属。
	if !(Output{DigestFlags: DigestContent | DigestScript, Type: TypeCoin}).InState() {
		t.Error("带摘要标记的币金输出仍应进入 UTXO")
	}
}

// TestOutputLockScriptLimit 校验锁定脚本超过 MaxLockScript 被拒绝。
func TestOutputLockScriptLimit(t *testing.T) {
	o := Output{Type: TypeCoin, LockScript: bytes.Repeat([]byte{0x00}, types.MaxLockScript+1)}
	if _, err := o.appendCanonical(nil); err == nil {
		t.Fatal("超长 LockScript 应被拒绝")
	}
}

// TestOutputCanonicalLayout 校验标准类输出编码顺序：Config || Payload || LockScript。
func TestOutputCanonicalLayout(t *testing.T) {
	payload := []byte("payload-bytes")
	lock := []byte{0x51, 0x52}
	o := Output{Type: TypeCoin, Payload: payload, LockScript: lock}
	b, err := o.appendCanonical(nil)
	if err != nil {
		t.Fatalf("appendCanonical: %v", err)
	}
	var want []byte
	want = append(want, 0x01)            // Config: 币金
	want = append(want, payload...)      // Payload 直接追加（自界定）
	want = types.AppendBytes(want, lock) // LockScript: varint(len)||bytes
	if !bytes.Equal(b, want) {
		t.Fatalf("输出编码不匹配\n got=%x\nwant=%x", b, want)
	}
}
