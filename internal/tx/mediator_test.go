package tx

import (
	"bytes"
	"testing"
)

// TestMediatorIsProofClass 校验介管脚本输出归属存证类（类型 3）且不入状态集。
func TestMediatorIsProofClass(t *testing.T) {
	script := []byte{0x01, 0x02, 0x03}
	o := NewMediatorOutput(script)
	if o.Type != TypeProof {
		t.Fatalf("介管脚本类型应为存证(3)，got=%d", o.Type)
	}
	if o.InState() {
		t.Fatal("介管脚本不可作为输入源，不应入状态集")
	}
	if !bytes.Equal(o.LockScript, script) {
		t.Fatalf("介管脚本应置于 LockScript，got=%x", o.LockScript)
	}
}

// TestMediatorCanonicalConfig 校验介管脚本输出的公共头 Config 为存证类型值。
func TestMediatorCanonicalConfig(t *testing.T) {
	o := NewMediatorOutput([]byte{0xAA})
	cfg, err := o.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if cfg != byte(TypeProof) {
		t.Fatalf("介管脚本 Config 应为 0x03，got=0x%02x", cfg)
	}
}
