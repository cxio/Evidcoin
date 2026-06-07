package tx

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// coinOut 构造一个位于 serial 序位的币金输出，便于输出树测试。
func coinOut(t *testing.T, serial uint32, amount uint64) Output {
	t.Helper()
	pl, err := Coin{Amount: types.Amount(amount)}.Payload()
	if err != nil {
		t.Fatalf("Coin.Payload: %v", err)
	}
	return Output{Serial: serial, Type: TypeCoin, Payload: pl}
}

// TestHashOutputsEmptyRejected 校验普通交易空输出集被拒绝。
func TestHashOutputsEmptyRejected(t *testing.T) {
	if _, err := HashOutputs(nil); err == nil {
		t.Fatal("空输出集应被拒绝")
	}
}

// TestHashOutputsSingleLeafRoot 校验单输出根等于单叶树按 tree.branch 归一化的 32B 根。
// 参考叶 payload 使用 appendLeafPreimage（无摘要标记时等同规范编码）。
func TestHashOutputsSingleLeafRoot(t *testing.T) {
	o := coinOut(t, 0, 100)
	got, err := HashOutputs([]Output{o})
	if err != nil {
		t.Fatalf("HashOutputs: %v", err)
	}
	preimage, err := o.appendLeafPreimage(nil)
	if err != nil {
		t.Fatalf("appendLeafPreimage: %v", err)
	}
	tree, err := hashtree.BuildFromPayloads([][]byte{preimage})
	if err != nil {
		t.Fatalf("BuildFromPayloads: %v", err)
	}
	if got.Bytes() == nil || string(got.Bytes()) != string(tree.Root()) {
		t.Fatalf("单输出根不匹配\n got=%x\nwant=%x", got.Bytes(), tree.Root())
	}
}

// TestHashOutputsOrderMatters 校验输出顺序变化导致根变化。
func TestHashOutputsOrderMatters(t *testing.T) {
	a := coinOut(t, 0, 100)
	b := coinOut(t, 1, 200)
	r1, err := HashOutputs([]Output{a, b})
	if err != nil {
		t.Fatalf("HashOutputs(ab): %v", err)
	}
	// 交换顺序后须重排 Serial 以满足位置约束，根应不同。
	a2 := coinOut(t, 1, 100)
	b2 := coinOut(t, 0, 200)
	r2, err := HashOutputs([]Output{b2, a2})
	if err != nil {
		t.Fatalf("HashOutputs(ba): %v", err)
	}
	if r1 == r2 {
		t.Fatal("不同顺序的输出集根不应相同")
	}
}

// TestHashOutputsMultiPath 校验多输出（含奇数）可计算且与 hashtree 一致。
// 参考叶 payload 使用 appendLeafPreimage。
func TestHashOutputsMultiPath(t *testing.T) {
	for _, n := range []int{2, 3, 5} {
		outs := make([]Output, n)
		payloads := make([][]byte, n)
		for i := 0; i < n; i++ {
			outs[i] = coinOut(t, uint32(i), uint64(10*i+1))
			preimage, err := outs[i].appendLeafPreimage(nil)
			if err != nil {
				t.Fatalf("appendLeafPreimage: %v", err)
			}
			payloads[i] = preimage
		}
		got, err := HashOutputs(outs)
		if err != nil {
			t.Fatalf("HashOutputs(n=%d): %v", n, err)
		}
		tree, err := hashtree.BuildFromPayloads(payloads)
		if err != nil {
			t.Fatalf("BuildFromPayloads(n=%d): %v", n, err)
		}
		if string(got.Bytes()) != string(tree.Root()) {
			t.Fatalf("多输出根不匹配(n=%d)\n got=%x\nwant=%x", n, got.Bytes(), tree.Root())
		}
	}
}

// TestHashOutputsSerialMismatch 校验 Serial 与位置不符时拒绝。
func TestHashOutputsSerialMismatch(t *testing.T) {
	a := coinOut(t, 0, 100)
	b := coinOut(t, 5, 200) // 序位应为 1，却标 5
	if _, err := HashOutputs([]Output{a, b}); err == nil {
		t.Fatal("Serial 与位置不符应被拒绝")
	}
}

// TestLeafPreimageNoFlagsEqualsCanonical 校验无摘要标记时叶前像与规范编码相同。
// 此属性保证现有协议行为不变（DEC-0101）。
func TestLeafPreimageNoFlagsEqualsCanonical(t *testing.T) {
	tests := []struct {
		name string
		out  Output
	}{
		{
			name: "coin",
			out: func() Output {
				pl, _ := Coin{Amount: 500, Receiver: []byte{0x01, 0x02}, Memo: []byte("hi")}.Payload()
				return Output{Serial: 0, Type: TypeCoin, Payload: pl}
			}(),
		},
		{
			name: "credit",
			out: func() Output {
				pl, _ := Credit{Receiver: []byte{0xAA}, Creator: []byte("alice"), Title: []byte("t")}.Payload()
				return Output{Serial: 0, Type: TypeCredit, Payload: pl}
			}(),
		},
		{
			name: "proof",
			out: func() Output {
				pl, _ := Proof{Creator: []byte("bob"), Title: []byte("doc"), Content: []byte("data")}.Payload()
				return Output{Serial: 0, Type: TypeProof, Payload: pl, LockScript: []byte{0x01}}
			}(),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			canon, err := tc.out.appendCanonical(nil)
			if err != nil {
				t.Fatalf("appendCanonical: %v", err)
			}
			preimage, err := tc.out.appendLeafPreimage(nil)
			if err != nil {
				t.Fatalf("appendLeafPreimage: %v", err)
			}
			if !bytes.Equal(canon, preimage) {
				t.Fatalf("无标记时叶前像应等于规范编码\n canon=%x\npreimage=%x", canon, preimage)
			}
		})
	}
}

// TestLeafPreimageCoinDigestAccount 校验 Coin + DigestAccount 叶前像字节级正确性。
// 前像结构：Config || H_account(48B) || Amount(varint) || Memo(encoded) || LockScript(encoded)
func TestLeafPreimageCoinDigestAccount(t *testing.T) {
	receiver := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	memo := []byte("pay")
	coin := Coin{Amount: 1000, Receiver: receiver, Memo: memo}
	pl, _ := coin.Payload()
	o := Output{Serial: 0, Type: TypeCoin, DigestFlags: DigestAccount, Payload: pl}

	got, err := o.appendLeafPreimage(nil)
	if err != nil {
		t.Fatalf("appendLeafPreimage: %v", err)
	}

	// 手工构造期望前像：Config || H_account(48B) || Amount || Memo_encoded || LockScript(varint(0))
	var want []byte
	want = append(want, byte(DigestAccount)|byte(TypeCoin)) // Config = 0x81
	h := crypto.HashOutputDigestAccount(receiver)
	want = append(want, h.Bytes()...)                 // H_account(48B)
	want = types.AppendVarUint(want, uint64(coin.Amount)) // Amount
	want = types.AppendBytes(want, memo)              // Memo(varint(len)||bytes)
	want = types.AppendBytes(want, o.LockScript)      // LockScript(varint(0))

	if !bytes.Equal(got, want) {
		t.Fatalf("DigestAccount 叶前像不匹配\n got=%x\nwant=%x", got, want)
	}
}

// TestLeafPreimageCoinDigestContent 校验 Coin + DigestContent 叶前像字节级正确性。
// 前像结构：Config || Receiver(encoded) || H_content(48B) || LockScript(encoded)
// H_content 覆盖 Amount + Memo（内容段合并）。
func TestLeafPreimageCoinDigestContent(t *testing.T) {
	receiver := []byte{0x11, 0x22}
	memo := []byte("note")
	coin := Coin{Amount: 250, Receiver: receiver, Memo: memo}
	pl, _ := coin.Payload()
	o := Output{Serial: 0, Type: TypeCoin, DigestFlags: DigestContent, Payload: pl}

	got, err := o.appendLeafPreimage(nil)
	if err != nil {
		t.Fatalf("appendLeafPreimage: %v", err)
	}

	// content_bytes = Amount(varint) || Memo(varint(len)||bytes)
	var contentBytes []byte
	contentBytes = types.AppendVarUint(contentBytes, uint64(coin.Amount))
	contentBytes = types.AppendBytes(contentBytes, memo)

	var want []byte
	want = append(want, byte(DigestContent)|byte(TypeCoin)) // Config = 0x41
	want = types.AppendBytes(want, receiver)     // Receiver(varint(len)||bytes)
	h := crypto.HashOutputDigestContent(contentBytes)
	want = append(want, h.Bytes()...)            // H_content(48B)
	want = types.AppendBytes(want, o.LockScript) // LockScript(varint(0))

	if !bytes.Equal(got, want) {
		t.Fatalf("DigestContent 叶前像不匹配\n got=%x\nwant=%x", got, want)
	}
}

// TestLeafPreimageCreditDigestAccount 校验 Credit + DigestAccount 叶前像字节级正确性。
// 前像结构：Config || H_account(48B) || Content(encoded fields...) || LockScript(encoded)
func TestLeafPreimageCreditDigestAccount(t *testing.T) {
	receiver := []byte{0x99}
	credit := Credit{
		Receiver:    receiver,
		Creator:     []byte("alice"),
		Title:       []byte("award"),
		Description: []byte("desc"),
	}
	pl, _ := credit.Payload()
	o := Output{Serial: 0, Type: TypeCredit, DigestFlags: DigestAccount, Payload: pl}

	got, err := o.appendLeafPreimage(nil)
	if err != nil {
		t.Fatalf("appendLeafPreimage: %v", err)
	}

	// 手工计算 content = 去掉 Receiver 后的剩余 payload。
	// Credit payload 首字段为 Receiver(varint(len)||bytes)，recvN = 1+1 = 2（len=1, byte=0x99）
	_, recvN, err := types.ReadBytes(pl)
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	contentBytes := pl[recvN:]

	var want []byte
	want = append(want, byte(DigestAccount)|byte(TypeCredit)) // Config = 0x82
	h := crypto.HashOutputDigestAccount(receiver)
	want = append(want, h.Bytes()...)            // H_account(48B)
	want = append(want, contentBytes...)         // Content(encoded fields)
	want = types.AppendBytes(want, o.LockScript) // LockScript(varint(0))

	if !bytes.Equal(got, want) {
		t.Fatalf("Credit DigestAccount 叶前像不匹配\n got=%x\nwant=%x", got, want)
	}
}

// TestLeafPreimageProofDigestContent 校验 Proof + DigestContent 叶前像字节级正确性。
// 前像结构：Config || H_content(48B) || LockScript(encoded)
func TestLeafPreimageProofDigestContent(t *testing.T) {
	proof := Proof{
		Creator: []byte("charlie"),
		Title:   []byte("cert"),
		Content: []byte("data block"),
	}
	pl, _ := proof.Payload()
	lockScript := []byte{0xFA, 0xCE}
	o := Output{Serial: 0, Type: TypeProof, DigestFlags: DigestContent, Payload: pl, LockScript: lockScript}

	got, err := o.appendLeafPreimage(nil)
	if err != nil {
		t.Fatalf("appendLeafPreimage: %v", err)
	}

	var want []byte
	want = append(want, byte(DigestContent)|byte(TypeProof)) // Config = 0x43
	h := crypto.HashOutputDigestContent(pl)
	want = append(want, h.Bytes()...)         // H_content(48B)
	want = types.AppendBytes(want, lockScript) // LockScript(varint(len)||bytes)

	if !bytes.Equal(got, want) {
		t.Fatalf("Proof DigestContent 叶前像不匹配\n got=%x\nwant=%x", got, want)
	}
}

// TestLeafPreimageDigestScript 校验 DigestScript 叶前像字节级正确性。
// 前像结构：Config || Payload || H_script(48B)（lockscript 替换为摘要）
func TestLeafPreimageDigestScript(t *testing.T) {
	lockScript := []byte{0x01, 0x02, 0x03}
	pl, _ := Coin{Amount: 100}.Payload()
	o := Output{Serial: 0, Type: TypeCoin, DigestFlags: DigestScript, Payload: pl, LockScript: lockScript}

	got, err := o.appendLeafPreimage(nil)
	if err != nil {
		t.Fatalf("appendLeafPreimage: %v", err)
	}

	var want []byte
	want = append(want, byte(DigestScript)|byte(TypeCoin)) // Config = 0x21
	want = append(want, pl...)                              // Payload（Coin 无 DigestAccount/Content）
	h := crypto.HashOutputDigestScript(lockScript)
	want = append(want, h.Bytes()...) // H_script(48B)

	if !bytes.Equal(got, want) {
		t.Fatalf("DigestScript 叶前像不匹配\n got=%x\nwant=%x", got, want)
	}
}

// TestLeafPreimageDigestFlagsChangeHash 校验摘要标记置位时输出树根与无标记不同。
func TestLeafPreimageDigestFlagsChangeHash(t *testing.T) {
	pl, _ := Coin{Amount: 1, Receiver: []byte{0x01}}.Payload()
	noFlags := Output{Serial: 0, Type: TypeCoin, Payload: pl}
	withAcct := Output{Serial: 0, Type: TypeCoin, DigestFlags: DigestAccount, Payload: pl}
	withCont := Output{Serial: 0, Type: TypeCoin, DigestFlags: DigestContent, Payload: pl}
	withScpt := Output{Serial: 0, Type: TypeCoin, DigestFlags: DigestScript, Payload: pl}

	r0, _ := HashOutputs([]Output{noFlags})
	r1, _ := HashOutputs([]Output{withAcct})
	r2, _ := HashOutputs([]Output{withCont})
	r3, _ := HashOutputs([]Output{withScpt})

	if r0 == r1 {
		t.Fatal("DigestAccount 应改变输出树根")
	}
	if r0 == r2 {
		t.Fatal("DigestContent 应改变输出树根")
	}
	if r0 == r3 {
		t.Fatal("DigestScript 应改变输出树根")
	}
	if r1 == r2 || r1 == r3 || r2 == r3 {
		t.Fatal("不同摘要标记组合应产生不同输出树根")
	}
}

