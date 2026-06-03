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
// 时 X 字节相同，但挑战种子因 Stakes 字段不同而不同。
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

	// 挑战种子因 Stakes 字段不同而不同。
	s0 := ComputeChallengeSeed(pre0)
	s1 := ComputeChallengeSeed(pre1)
	if bytes.Equal(s0, s1) {
		t.Fatal("challenge seed must differ when Stakes differs")
	}
}

// TestMintHashPreimageLayout 断言挑战种子前像字段顺序：
// MintPubKey || MintTxID || Stakes(BE u64) || RefMintHash || X（无域标签）。
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

	// 手工拼装期望前像（不含域标签）。
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

	// ComputeChallengeSeed 必须等于 crypto.HashMintChallengeSeed(前像)。
	gotSeed := ComputeChallengeSeed(pre)
	wantSeed := crypto.HashMintChallengeSeed(want)
	if !bytes.Equal(gotSeed, wantSeed) {
		t.Fatal("ComputeChallengeSeed must equal crypto.HashMintChallengeSeed(canonical preimage)")
	}
}

// TestComputeMintHash 断言 ComputeMintHash 对哈希列表进行拼接后计算
// BLAKE3-256(DomainTag("mint.hash") || hashList...)。
func TestComputeMintHash(t *testing.T) {
	hashList := [][]byte{
		bytes.Repeat([]byte{0x11}, 32),
		bytes.Repeat([]byte{0x22}, 32),
		bytes.Repeat([]byte{0x33}, 32),
	}
	got := ComputeMintHash(hashList)

	// 手工拼接哈希列表后计算期望结果。
	var concat []byte
	for _, h := range hashList {
		concat = append(concat, h...)
	}
	want := crypto.HashMint(concat)
	if got != want {
		t.Fatalf("ComputeMintHash = %x, want %x", got[:4], want[:4])
	}

	// 空哈希列表与 nil 等价。
	emptyGot := ComputeMintHash(nil)
	emptyWant := crypto.HashMint(nil)
	if emptyGot != emptyWant {
		t.Fatalf("ComputeMintHash(nil) = %x, want %x", emptyGot[:4], emptyWant[:4])
	}
}

// TestRankMintCandidates 断言四级升序：Nonce → MintHash → TxID → PubKey，按无符号字节序。
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

	// 零级（一级）：Nonce 不同，值小者胜（MintHash 更大也因 Nonce 更小而胜）。
	a0 := MintCandidate{Nonce: 1, MintHash: mh(0x01), TxID: tx(0xFF), MintPubKey: []byte{0xFF}}
	b0 := MintCandidate{Nonce: 2, MintHash: mh(0x00), TxID: tx(0x00), MintPubKey: []byte{0x00}}
	// 二级（Nonce 相同）：MintHash 不同，值小者胜。
	c := MintCandidate{Nonce: 3, MintHash: mh(0x01), TxID: tx(0xFF), MintPubKey: []byte{0xFF}}
	d := MintCandidate{Nonce: 3, MintHash: mh(0x02), TxID: tx(0x00), MintPubKey: []byte{0x00}}
	// 三级：Nonce 与 MintHash 相同，TxID 小者胜。
	e := MintCandidate{Nonce: 4, MintHash: mh(0x03), TxID: tx(0x01), MintPubKey: []byte{0xFF}}
	f := MintCandidate{Nonce: 4, MintHash: mh(0x03), TxID: tx(0x02), MintPubKey: []byte{0x00}}
	// 四级：Nonce / MintHash / TxID 相同，PubKey 小者胜。
	g := MintCandidate{Nonce: 5, MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x01}}
	h := MintCandidate{Nonce: 5, MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x02}}

	in := []MintCandidate{h, f, d, b0, g, e, c, a0}
	RankMintCandidates(in)

	wantOrder := []MintCandidate{a0, b0, c, d, e, f, g, h}
	for i := range wantOrder {
		wi := wantOrder[i]
		gi := in[i]
		if gi.Nonce != wi.Nonce || gi.MintHash != wi.MintHash ||
			gi.TxID != wi.TxID || !bytes.Equal(gi.MintPubKey, wi.MintPubKey) {
			t.Fatalf("rank[%d] mismatch: got nonce=%d mh=%x tx=%x pk=%x",
				i, gi.Nonce, gi.MintHash[:1], gi.TxID.Bytes()[:1], gi.MintPubKey)
		}
	}
}

// TestCompareMintCandidates 单独断言比较函数的符号，含四级排序所有路径。
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
			name: "nonce less",
			a:    MintCandidate{Nonce: 1, MintHash: mh(0xFF), TxID: tx(0xFF), MintPubKey: []byte{0xFF}},
			b:    MintCandidate{Nonce: 2, MintHash: mh(0x00), TxID: tx(0x00), MintPubKey: []byte{0x00}},
			want: -1,
		},
		{
			name: "nonce greater",
			a:    MintCandidate{Nonce: 5, MintHash: mh(0x01)},
			b:    MintCandidate{Nonce: 3, MintHash: mh(0xFF)},
			want: 1,
		},
		{
			name: "mint hash less (same nonce)",
			a:    MintCandidate{Nonce: 0, MintHash: mh(0x01), TxID: tx(0xFF), MintPubKey: []byte{0xFF}},
			b:    MintCandidate{Nonce: 0, MintHash: mh(0x02), TxID: tx(0x00), MintPubKey: []byte{0x00}},
			want: -1,
		},
		{
			name: "tx id tiebreak",
			a:    MintCandidate{Nonce: 0, MintHash: mh(0x03), TxID: tx(0x01), MintPubKey: []byte{0xFF}},
			b:    MintCandidate{Nonce: 0, MintHash: mh(0x03), TxID: tx(0x02), MintPubKey: []byte{0x00}},
			want: -1,
		},
		{
			name: "pubkey tiebreak",
			a:    MintCandidate{Nonce: 0, MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x01}},
			b:    MintCandidate{Nonce: 0, MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x02}},
			want: -1,
		},
		{
			name: "fully equal",
			a:    MintCandidate{Nonce: 7, MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x01}},
			b:    MintCandidate{Nonce: 7, MintHash: mh(0x04), TxID: tx(0x05), MintPubKey: []byte{0x01}},
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

