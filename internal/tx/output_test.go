package tx

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestOutputConfigStandard 校验标准类输出 Config 字节布局：
// bit[3:0] 类型值，bit7=0，bit4-bit6 未使用（始终为 0）。
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
			// bit7 必须为 0（非自定义类）；bit4-bit6 未使用，必须为 0。
			if got&0xF0 != 0 {
				t.Fatalf("标准类 Config 高 4 位应全为 0, got=%#x", got)
			}
		})
	}
}

// TestOutputConfigCustom 校验自定义类输出 Config：bit7=1，低 7 位为类 ID 长度计数。
func TestOutputConfigCustom(t *testing.T) {
	id := bytes.Repeat([]byte{0xEE}, 10)
	o := Output{IsCustom: true, CustomID: id}
	got, err := o.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if got != (0x80 | 10) {
		t.Fatalf("自定义类 Config = %#x, 期望 %#x", got, 0x80|10)
	}

	// 编码须包含 CustomID 字节本身。
	b, err := o.appendCanonical(nil)
	if err != nil {
		t.Fatalf("appendCanonical: %v", err)
	}
	if b[0] != (0x80|10) || !bytes.Contains(b, id) {
		t.Fatalf("自定义类编码缺少 Config 或 CustomID: %x", b)
	}
}

// TestOutputCustomIDTooLong 校验自定义类 ID 超过 127 字节被拒绝。
func TestOutputCustomIDTooLong(t *testing.T) {
	o := Output{IsCustom: true, CustomID: bytes.Repeat([]byte{0x00}, 128)}
	if _, err := o.Config(); err == nil {
		t.Fatal("CustomID 长度 128 应被拒绝")
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

// TestOutputInState 校验自定义类与存证不进入 UTXO/UTCO，币金/凭信进入。
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
	if (Output{IsCustom: true, CustomID: []byte{0x01}}).InState() {
		t.Error("自定义类输出不应进入状态集")
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
