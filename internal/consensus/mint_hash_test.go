package consensus

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 复用测试向量：固定铸造者公钥与铸凭交易 ID。
var (
	testMintPubKey = []byte{0x01, 0x02, 0x03, 0x04}
	testMintTxID   = types.MustTxID(bytes.Repeat([]byte{0xAB}, 48))
)

// TestMintHashXEncoding 断言 X = BE(minimal_unsigned(BlockHeight × Mix))，
// 用无损大整数最短编码，且仅由区块高度与 Mix 决定，与 Stakes 无关。
func TestMintHashXEncoding(t *testing.T) {
	tests := []struct {
		name   string
		height uint32
	}{
		{"height zero", 0},
		{"height one", 1},
		{"height small", 7},
		{"height large", 80000},
		{"height max uint32", 0xFFFFFFFF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeMintX(tt.height)
			// 期望：BlockHeight × Mix 的大端最短无符号编码。
			prod := new(big.Int).Mul(
				new(big.Int).SetUint64(uint64(tt.height)),
				new(big.Int).SetUint64(mintMix),
			)
			want := prod.Bytes() // big.Int.Bytes 即大端最短无符号编码；零值为空。
			if !bytes.Equal(got, want) {
				t.Fatalf("encodeMintX(%d) = %x, want %x", tt.height, got, want)
			}
		})
	}
}

// TestMintHashXIndependentOfStakes 断言同一 BlockHeight 下 Stakes=0 与 Stakes>0
// 时 X 字节相同，但完整铸凭哈希因 Stakes 字段不同而不同。
func TestMintHashXIndependentOfStakes(t *testing.T) {
	const height = uint32(12345)
	xZero := encodeMintX(height)
	xNonZero := encodeMintX(height)
	if !bytes.Equal(xZero, xNonZero) {
		t.Fatalf("X must be independent of Stakes: %x vs %x", xZero, xNonZero)
	}

	var refZero types.MintHash
	pre0 := MintHashPreimage{
		MintPubKey:  testMintPubKey,
		MintTxID:    testMintTxID,
		Stakes:      0,
		RefMintHash: refZero,
		BlockHeight: height,
	}
	pre1 := pre0
	pre1.Stakes = 999

	h0 := ComputeMintHash(pre0)
	h1 := ComputeMintHash(pre1)
	if h0 == h1 {
		t.Fatal("mint hash must differ when Stakes differs")
	}
}

// TestMintHashPreimageLayout 断言前像字段顺序：
// DomainTag("mint.hash") || MintPubKey || MintTxID || Stakes(BE u64) || RefMintHash || X。
func TestMintHashPreimageLayout(t *testing.T) {
	const height = uint32(42)
	var ref types.MintHash
	copy(ref[:], bytes.Repeat([]byte{0xCD}, 32))

	pre := MintHashPreimage{
		MintPubKey:  testMintPubKey,
		MintTxID:    testMintTxID,
		Stakes:      0x0102030405060708,
		RefMintHash: ref,
		BlockHeight: height,
	}

	// 手工拼装期望前像（不含域标签，域标签由 crypto.HashMint 内部添加）。
	var want []byte
	want = append(want, testMintPubKey...)
	want = append(want, testMintTxID.Bytes()...)
	want = types.AppendUint64BE(want, 0x0102030405060708)
	want = append(want, ref.Bytes()...)
	want = append(want, encodeMintX(height)...)

	got := pre.CanonicalBytes()
	if !bytes.Equal(got, want) {
		t.Fatalf("preimage = %x, want %x", got, want)
	}

	// ComputeMintHash 必须等于 crypto.HashMint(前像)。
	if ComputeMintHash(pre) != crypto.HashMint(want) {
		t.Fatal("ComputeMintHash must equal crypto.HashMint(canonical preimage)")
	}
}

// TestRankMintCandidates 断言三级升序：MintHash → TxID → PubKey，按无符号字节序。
func TestRankMintCandidates(t *testing.T) {
	mh := func(b byte) types.MintHash {
		var h types.MintHash
		for i := range h {
			h[i] = b
		}
		return h
	}
	tx := func(b byte) types.TxID {
		return types.MustTxID(bytes.Repeat([]byte{b}, 48))
	}

	// 一级：MintHash 不同，值小者胜。
	a := MintCandidate{MintHash: mh(0x01), TxID: tx(0xFF), MintPubKey: []byte{0xFF}}
	b := MintCandidate{MintHash: mh(0x02), TxID: tx(0x00), MintPubKey: []byte{0x00}}
	// 二级：MintHash 相同，TxID 小者胜。
	c := MintCandidate{MintHash: mh(0x03), TxID: tx(0x01), MintPubKey: []byte{0xFF}}
	d := MintCandidate{MintHash: mh(0x03), TxID: tx(0x02), MintPubKey: []byte{0x00}}
	// 三级：MintHash 与 TxID 相同，PubKey 小者胜。
	e := MintCandidate{MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x01}}
	f := MintCandidate{MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x02}}

	in := []MintCandidate{f, d, b, e, c, a}
	RankMintCandidates(in)

	wantOrder := []MintCandidate{a, b, c, d, e, f}
	for i := range wantOrder {
		if in[i].MintHash != wantOrder[i].MintHash ||
			in[i].TxID != wantOrder[i].TxID ||
			!bytes.Equal(in[i].MintPubKey, wantOrder[i].MintPubKey) {
			t.Fatalf("rank[%d] mismatch: got mh=%x tx=%x pk=%x",
				i, in[i].MintHash[:1], in[i].TxID.Bytes()[:1], in[i].MintPubKey)
		}
	}
}

// TestCompareMintCandidates 单独断言比较函数的符号。
func TestCompareMintCandidates(t *testing.T) {
	mh := func(b byte) types.MintHash {
		var h types.MintHash
		for i := range h {
			h[i] = b
		}
		return h
	}
	tx := func(b byte) types.TxID {
		return types.MustTxID(bytes.Repeat([]byte{b}, 48))
	}
	tests := []struct {
		name string
		a, b MintCandidate
		want int // 负=a<b，0=相等，正=a>b
	}{
		{
			name: "mint hash less",
			a:    MintCandidate{MintHash: mh(0x01), TxID: tx(0xFF), MintPubKey: []byte{0xFF}},
			b:    MintCandidate{MintHash: mh(0x02), TxID: tx(0x00), MintPubKey: []byte{0x00}},
			want: -1,
		},
		{
			name: "tx id tiebreak",
			a:    MintCandidate{MintHash: mh(0x03), TxID: tx(0x01), MintPubKey: []byte{0xFF}},
			b:    MintCandidate{MintHash: mh(0x03), TxID: tx(0x02), MintPubKey: []byte{0x00}},
			want: -1,
		},
		{
			name: "pubkey tiebreak",
			a:    MintCandidate{MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x01}},
			b:    MintCandidate{MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x02}},
			want: -1,
		},
		{
			name: "fully equal",
			a:    MintCandidate{MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x01}},
			b:    MintCandidate{MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x01}},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareMintCandidates(tt.a, tt.b)
			if (got < 0) != (tt.want < 0) || (got == 0) != (tt.want == 0) || (got > 0) != (tt.want > 0) {
				t.Fatalf("CompareMintCandidates sign = %d, want sign %d", got, tt.want)
			}
		})
	}
}
