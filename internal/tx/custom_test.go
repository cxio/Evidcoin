package tx

import (
	"bytes"
	"testing"
)

// TestCustomOutputBasics 校验自定义类输出：bit7 置位、不入状态集、ID 与载荷正确承载。
func TestCustomOutputBasics(t *testing.T) {
	id := []byte("appid")
	payload := []byte{0x11, 0x22}
	o, err := NewCustomOutput(id, payload, nil)
	if err != nil {
		t.Fatalf("NewCustomOutput: %v", err)
	}
	if !o.IsCustom {
		t.Fatal("应为自定义类")
	}
	if o.InState() {
		t.Fatal("自定义类不进入 UTXO/UTCO")
	}
	cfg, err := o.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if cfg != (0x80 | byte(len(id))) {
		t.Fatalf("Config 应为 0x80|len(id)，got=0x%02x", cfg)
	}
	if !bytes.Equal(o.CustomID, id) || !bytes.Equal(o.Payload, payload) {
		t.Fatal("CustomID/Payload 承载错误")
	}
}

// TestCustomIDTooLong 校验私有 ID 超过 127 字节被拒绝。
func TestCustomIDTooLong(t *testing.T) {
	if _, err := NewCustomOutput(bytes.Repeat([]byte{0}, 128), nil, nil); err == nil {
		t.Fatal("128 字节私有 ID 应被拒绝")
	}
}

// TestCustomCanonicalEncodes 校验自定义类输出能产出规范编码（编码合法性校验通过）。
func TestCustomCanonicalEncodes(t *testing.T) {
	o, err := NewCustomOutput([]byte("x"), []byte{0x09}, []byte{0x05})
	if err != nil {
		t.Fatalf("NewCustomOutput: %v", err)
	}
	if _, err := o.appendCanonical(nil); err != nil {
		t.Fatalf("appendCanonical: %v", err)
	}
}
