package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// prefixLen 计算给定参数下签名消息覆盖区之前的固定前缀长度，
// 便于测试从整条消息中切出 CoveredInputs||CoveredOutputs 部分。
func messagePrefix(p SigParams) []byte {
	dst := crypto.SignatureMessageTag()
	dst = p.Chain.appendCanonical(dst)
	dst = append(dst, byte(p.ChkType), p.AuthFlag.Byte())
	dst = types.AppendVarUint(dst, p.InputIndex)
	dst = types.AppendUint16BE(dst, p.Version)
	dst = types.AppendInt64BE(dst, p.Timestamp)
	dst = types.AppendBytes(dst, p.MintPKHash)
	return dst
}

func TestAuthFlagValidate(t *testing.T) {
	tests := []struct {
		name string
		flag AuthFlag
		want error
	}{
		{"empty", 0, nil},
		{"sigin only", SigInAll, nil},
		{"sigin self only", SigInSelf, nil},
		{"main+script", SigOutAll | AuxScript, nil},
		{"self+output", SigOutSelf | AuxOutput, nil},
		{"output conflict script", SigOutAll | AuxOutput | AuxScript, ErrOutputAuxConflict},
		{"output conflict receiver", SigOutAll | AuxOutput | AuxReceiver, ErrOutputAuxConflict},
		{"main without aux", SigOutAll, ErrMainWithoutAux},
		{"aux without main", AuxContent, ErrAuxWithoutMain},
		{"output without main", AuxOutput, ErrAuxWithoutMain},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.flag.Validate(); err != tt.want {
				t.Fatalf("Validate(%08b) = %v, want %v", byte(tt.flag), err, tt.want)
			}
		})
	}
}

func TestCoveredInputsSigInAll(t *testing.T) {
	in := testInputs()
	p := SigParams{
		Chain:    testChainScope(nil),
		ChkType:  ChkCoinSpend,
		AuthFlag: SigInAll | SigOutSelf | AuxScript,
		Version:  1, Timestamp: 1,
		InputIndex: 0,
		Inputs:     in,
		Outputs:    testOutputs(),
	}
	msg, err := BuildSignatureMessage(p)
	if err != nil {
		t.Fatal(err)
	}
	covered := msg[len(messagePrefix(p)):]

	// 期望 CoveredInputs：varint(count=2) || Lead || Rest[0]。
	want := types.AppendVarUint(nil, 2)
	want, _ = in.Lead.appendCanonical(want)
	want, _ = in.Rest[0].appendCanonical(want)
	if string(covered[:len(want)]) != string(want) {
		t.Fatalf("CoveredInputs mismatch\n got=%x\nwant=%x", covered[:len(want)], want)
	}
}

func TestCoveredInputsSigInSelf(t *testing.T) {
	in := testInputs()
	p := SigParams{
		Chain:    testChainScope(nil),
		ChkType:  ChkCoinSpend,
		AuthFlag: SigInSelf,
		Version:  1, Timestamp: 1,
		InputIndex: 1,
		Inputs:     in,
		Outputs:    testOutputs(),
	}
	msg, err := BuildSignatureMessage(p)
	if err != nil {
		t.Fatal(err)
	}
	covered := msg[len(messagePrefix(p)):]

	want := types.AppendVarUint(nil, 1) // input_index
	enc, _ := in.Rest[0].appendCanonical(nil)
	want = append(want, enc...)
	if string(covered) != string(want) {
		t.Fatalf("CoveredInputs(SELF) mismatch\n got=%x\nwant=%x", covered, want)
	}
}

func TestCoveredInputsSigInSelfRange(t *testing.T) {
	p := SigParams{
		Chain:    testChainScope(nil),
		ChkType:  ChkCoinSpend,
		AuthFlag: SigInSelf,
		Version:  1, Timestamp: 1,
		InputIndex: 9,
		Inputs:     testInputs(),
		Outputs:    testOutputs(),
	}
	if _, err := BuildSignatureMessage(p); err != ErrInputIndexRange {
		t.Fatalf("expected ErrInputIndexRange, got %v", err)
	}
}

func TestCoveredOutputsSigOutAll(t *testing.T) {
	outs := testOutputs()
	p := SigParams{
		Chain:    testChainScope(nil),
		ChkType:  ChkCoinSpend,
		AuthFlag: SigOutAll | AuxReceiver,
		Version:  1, Timestamp: 1,
		InputIndex: 0,
		Inputs:     testInputs(),
		Outputs:    outs,
	}
	msg, err := BuildSignatureMessage(p)
	if err != nil {
		t.Fatal(err)
	}
	covered := msg[len(messagePrefix(p)):]

	// 期望：varint(count=2) || [varint(0)||receiver0] || [varint(1)||receiver1]。
	want := types.AppendVarUint(nil, 2)
	want = types.AppendVarUint(want, 0)
	want = types.AppendBytes(want, outs[0].Receiver)
	want = types.AppendVarUint(want, 1)
	want = types.AppendBytes(want, outs[1].Receiver)
	if string(covered) != string(want) {
		t.Fatalf("CoveredOutputs(ALL) mismatch\n got=%x\nwant=%x", covered, want)
	}
}

func TestCoveredOutputsSigOutSelf(t *testing.T) {
	outs := testOutputs()
	p := SigParams{
		Chain:    testChainScope(nil),
		ChkType:  ChkCoinSpend,
		AuthFlag: SigOutSelf | AuxOutput,
		Version:  1, Timestamp: 1,
		InputIndex: 1,
		Inputs:     testInputs(),
		Outputs:    outs,
	}
	msg, err := BuildSignatureMessage(p)
	if err != nil {
		t.Fatal(err)
	}
	covered := msg[len(messagePrefix(p)):]

	// AuxOutput 展开为 SCRIPT||CONTENT||RECEIVER 三段（同序位输出 1）。
	want := types.AppendBytes(nil, outs[1].LockScript)
	want = types.AppendBytes(want, outs[1].Content)
	want = types.AppendBytes(want, outs[1].Receiver)
	if string(covered) != string(want) {
		t.Fatalf("CoveredOutputs(SELF) mismatch\n got=%x\nwant=%x", covered, want)
	}
}

func TestCoveredOutputsSigOutSelfRange(t *testing.T) {
	p := SigParams{
		Chain:    testChainScope(nil),
		ChkType:  ChkCoinSpend,
		AuthFlag: SigOutSelf | AuxScript,
		Version:  1, Timestamp: 1,
		InputIndex: 5, // >= len(outputs)=2
		Inputs:     testInputs(),
		Outputs:    testOutputs(),
	}
	if _, err := BuildSignatureMessage(p); err != ErrSigOutSelfRange {
		t.Fatalf("expected ErrSigOutSelfRange, got %v", err)
	}
}

func TestAuxFieldSelection(t *testing.T) {
	o := SignableOutput{Receiver: []byte("R"), Content: []byte("C"), LockScript: []byte("S")}
	tests := []struct {
		name string
		flag AuthFlag
		want []byte
	}{
		{"script", AuxScript, types.AppendBytes(nil, o.LockScript)},
		{"content", AuxContent, types.AppendBytes(nil, o.Content)},
		{"receiver", AuxReceiver, types.AppendBytes(nil, o.Receiver)},
		{"script+content", AuxScript | AuxContent, appendTwo(o.LockScript, o.Content)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendOutputFields(nil, tt.flag, o)
			if string(got) != string(tt.want) {
				t.Fatalf("appendOutputFields %s\n got=%x\nwant=%x", tt.name, got, tt.want)
			}
		})
	}
}

func appendTwo(a, b []byte) []byte {
	dst := types.AppendBytes(nil, a)
	return types.AppendBytes(dst, b)
}

// TestSignableFromPayloads 校验三类信元的覆盖视图分解与 Payload 编码一致。
func TestSignableFromPayloads(t *testing.T) {
	coin := Coin{Amount: 500, Receiver: []byte("rc"), Memo: []byte("m")}
	so, err := SignableFromCoin(coin, []byte("ls"))
	if err != nil {
		t.Fatal(err)
	}
	wantContent := types.AppendVarUint(nil, 500)
	wantContent = types.AppendBytes(wantContent, coin.Memo)
	if string(so.Content) != string(wantContent) {
		t.Fatalf("coin content mismatch\n got=%x\nwant=%x", so.Content, wantContent)
	}
	if string(so.Receiver) != string(coin.Receiver) {
		t.Fatal("coin receiver mismatch")
	}

	proof := Proof{Creator: []byte("cr"), Title: []byte("t"), Content: []byte("c")}
	sp, err := SignableFromProof(proof, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(sp.Receiver) != string(proof.Creator) {
		t.Fatal("proof creator must serve as receiver segment")
	}
}
