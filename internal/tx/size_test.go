package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// sizeFixtureHeader 构造一个用于尺寸测试的最小普通交易头。
func sizeFixtureHeader() *TxHeader {
	return &TxHeader{Version: 1}
}

// sizeFixtureOutputs 构造单个币金输出集。
func sizeFixtureOutputs(t *testing.T) []Output {
	t.Helper()
	pl, err := Coin{Amount: types.Amount(1)}.Payload()
	if err != nil {
		t.Fatalf("Coin.Payload: %v", err)
	}
	return []Output{{Serial: 0, Type: TypeCoin, Payload: pl}}
}

// validRef 构造一个合法的来源短引用。
func validRef() OutPoint {
	return OutPoint{Year: 2026, TxIDPart: make([]byte, MinTxIDPartLen), OutIndex: 0}
}

// TestTxSizeUnlockScriptCounted 校验解锁脚本长度计入交易尺寸。
func TestTxSizeUnlockScriptCounted(t *testing.T) {
	h := sizeFixtureHeader()
	outs := sizeFixtureOutputs(t)
	in1 := Inputs{Lead: LeadInput{Ref: validRef(), UnlockScript: nil}}
	in2 := Inputs{Lead: LeadInput{Ref: validRef(), UnlockScript: make([]byte, 100)}}
	s1, err := TxSize(h, in1, outs)
	if err != nil {
		t.Fatalf("TxSize(in1): %v", err)
	}
	s2, err := TxSize(h, in2, outs)
	if err != nil {
		t.Fatalf("TxSize(in2): %v", err)
	}
	if s2 <= s1 {
		t.Fatalf("解锁脚本应增大交易尺寸: s1=%d s2=%d", s1, s2)
	}
	// 空脚本与 100 字节脚本的 varint 长度前缀均为 1 字节（0 与 100 都 <128），
	// 故差额恰为脚本内容 100 字节。
	if s2-s1 != 100 {
		t.Fatalf("尺寸增量应为 100（脚本内容字节），got=%d", s2-s1)
	}
}

// TestTxSizeWithinLimit 校验常规小交易通过尺寸检查。
func TestTxSizeWithinLimit(t *testing.T) {
	h := sizeFixtureHeader()
	in := Inputs{Lead: LeadInput{Ref: validRef()}}
	if err := CheckTxSize(h, in, sizeFixtureOutputs(t)); err != nil {
		t.Fatalf("常规交易不应超限: %v", err)
	}
}

// TestTxSizeExceedsLimit 校验超过 MaxTxSize 的交易被拒绝（见证不计，此处用多个输入的解锁脚本撑大）。
func TestTxSizeExceedsLimit(t *testing.T) {
	h := sizeFixtureHeader()
	// 每个解锁脚本 8000 字节，9 个输入 → 约 72KB，超过 65535。
	rest := make([]RestInput, 8)
	for i := range rest {
		rest[i] = RestInput{Kind: InputCoin, Ref: validRef(), UnlockScript: make([]byte, 8000)}
	}
	in := Inputs{
		Lead: LeadInput{Ref: validRef(), UnlockScript: make([]byte, 8000)},
		Rest: rest,
	}
	if err := CheckTxSize(h, in, sizeFixtureOutputs(t)); err == nil {
		t.Fatal("超过 MaxTxSize 的交易应被拒绝")
	}
}

// TestTxSizeWitnessExcluded 校验尺寸仅基于交易体（Header+Inputs+Outputs），
// 不含见证：相同交易体的尺寸与是否存在外部见证无关（本层无见证字段，断言尺寸稳定）。
func TestTxSizeWitnessExcluded(t *testing.T) {
	h := sizeFixtureHeader()
	in := Inputs{Lead: LeadInput{Ref: validRef(), UnlockScript: make([]byte, 50)}}
	outs := sizeFixtureOutputs(t)
	body, err := CanonicalBody(h, in, outs)
	if err != nil {
		t.Fatalf("CanonicalBody: %v", err)
	}
	size, err := TxSize(h, in, outs)
	if err != nil {
		t.Fatalf("TxSize: %v", err)
	}
	if size != len(body) {
		t.Fatalf("TxSize 应等于交易体字节长度: size=%d body=%d", size, len(body))
	}
}
