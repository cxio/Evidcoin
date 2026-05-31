package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 构造一组测试用基础材料。

func testChainScope(bound []byte) ChainScope {
	gid := types.MustBlockID(make([]byte, 48))
	return ChainScope{
		ProtocolID: "Evidcoin@v1",
		ChainID:    "mainnet",
		GenesisID:  gid,
		BoundID:    bound,
	}
}

func testInputs() Inputs {
	part := make([]byte, MinTxIDPartLen)
	for i := range part {
		part[i] = byte(i + 1)
	}
	return Inputs{
		Lead: LeadInput{
			Ref:          OutPoint{Year: 2026, TxIDPart: part, OutIndex: 0},
			UnlockScript: []byte{0xaa, 0xbb},
		},
		Rest: []RestInput{
			{
				Kind:         InputCredit,
				Ref:          OutPoint{Year: 2026, TxIDPart: part, OutIndex: 1},
				UnlockScript: []byte{0xcc},
			},
		},
	}
}

func testOutputs() []SignableOutput {
	return []SignableOutput{
		{Receiver: []byte("recv-0"), Content: []byte("body-0"), LockScript: []byte("lock-0")},
		{Receiver: []byte("recv-1"), Content: []byte("body-1"), LockScript: []byte("lock-1")},
	}
}

// TestSignatureMessagePrefixLayout 校验签名消息的固定前缀布局（DomainTag||ChainScope||SigScope||TxHeaderCore）。
func TestSignatureMessagePrefixLayout(t *testing.T) {
	cs := testChainScope(nil)
	p := SigParams{
		Chain:      cs,
		ChkType:    ChkCoinSpend,
		AuthFlag:   0, // 无覆盖项
		InputIndex: 3,
		Version:    7,
		Timestamp:  0x0102030405060708,
		MintPKHash: nil,
		Inputs:     testInputs(),
		Outputs:    testOutputs(),
	}
	got, err := BuildSignatureMessage(p)
	if err != nil {
		t.Fatal(err)
	}

	// 独立构造期望前缀。
	want := crypto.SignatureMessageTag()
	want = cs.appendCanonical(want)
	want = append(want, byte(ChkCoinSpend), 0x00)
	want = types.AppendVarUint(want, 3)
	want = types.AppendUint16BE(want, 7)
	want = types.AppendInt64BE(want, 0x0102030405060708)
	want = types.AppendBytes(want, nil) // MintPKHash 缺省 => varint(0)

	// AuthFlag=0 时 CoveredInputs/CoveredOutputs 均为空，want 即整条消息。
	if string(got) != string(want) {
		t.Fatalf("signature message mismatch\n got=%x\nwant=%x", got, want)
	}
}

// TestSignatureMessageBoundIDPlaceholder 校验 BoundID 空时仍以 varint(0) 占位。
func TestSignatureMessageBoundIDPlaceholder(t *testing.T) {
	empty := testChainScope(nil)
	withBound := testChainScope([]byte{0x11, 0x22, 0x33})

	be := empty.appendCanonical(nil)
	bw := withBound.appendCanonical(nil)

	// 空 BoundID 必须出现一个 0x00 占位字节（紧随 48 字节 GenesisID 之后）。
	if be[len(be)-1] != 0x00 {
		t.Fatalf("empty BoundID must encode varint(0) placeholder, got tail %x", be[len(be)-1])
	}
	if string(be) == string(bw) {
		t.Fatal("BoundID presence must change ChainScope bytes")
	}
}

// TestSignatureMessageMintPKHash 校验铸凭公钥哈希三处编码对照（存在/缺省/非法）。
func TestSignatureMessageMintPKHash(t *testing.T) {
	base := SigParams{
		Chain:     testChainScope(nil),
		ChkType:   ChkCoinSpend,
		Version:   1,
		Timestamp: 1,
		Inputs:    testInputs(),
		Outputs:   testOutputs(),
	}

	// 缺省。
	none, err := BuildSignatureMessage(base)
	if err != nil {
		t.Fatal(err)
	}
	// 存在（32 字节）。
	mint := make([]byte, 32)
	for i := range mint {
		mint[i] = 0x5a
	}
	withMint := base
	withMint.MintPKHash = mint
	got, err := BuildSignatureMessage(withMint)
	if err != nil {
		t.Fatal(err)
	}
	if string(none) == string(got) {
		t.Fatal("MintPKHash presence must change signature message")
	}
	// 非法长度。
	bad := base
	bad.MintPKHash = make([]byte, 16)
	if _, err := BuildSignatureMessage(bad); err != ErrMintPKHashLength {
		t.Fatalf("expected ErrMintPKHashLength, got %v", err)
	}
}

// TestSignatureMessageChainIdentityChanges 校验修改链身份会改变签名消息字节。
func TestSignatureMessageChainIdentityChanges(t *testing.T) {
	base := SigParams{
		Chain:     testChainScope(nil),
		ChkType:   ChkCoinSpend,
		Version:   1,
		Timestamp: 1,
		Inputs:    testInputs(),
		Outputs:   testOutputs(),
	}
	a, _ := BuildSignatureMessage(base)

	other := base
	other.Chain.ChainID = "testnet"
	b, _ := BuildSignatureMessage(other)
	if string(a) == string(b) {
		t.Fatal("changing ChainID must change signature message")
	}
}
