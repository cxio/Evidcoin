package consensus

import (
	"bytes"
	"errors"
	"testing"

	rxpkg "github.com/cxio/evidcoin/internal/consensus/randomx"
	"github.com/cxio/evidcoin/pkg/types"
)

// deterministicHasher 是测试用确定性哈希器：Hash(seed, input) = XOR(seed[0], input[0]) || 其余全零
// 仅用于覆盖排序逻辑，不代表真实 RandomX 输出。
type deterministicHasher struct{}

func (deterministicHasher) Hash(seed, input []byte) ([]byte, error) {
	out := make([]byte, 32)
	if len(seed) > 0 && len(input) > 0 {
		out[0] = seed[0] ^ input[0]
	}
	return out, nil
}

// errorHasher 始终返回指定错误（测试错误传播）。
type errorHasher struct{ err error }

func (e errorHasher) Hash(_, _ []byte) ([]byte, error) { return nil, e.err }

func makeBlockID(b byte) types.BlockID {
	var id types.BlockID
	id[0] = b
	return id
}

func TestRandomXTiebreak(t *testing.T) {
	h := deterministicHasher{}

	cases := []struct {
		name      string
		forkPoint byte
		forkA     byte
		forkB     byte
		want      ForkSide
	}{
		{
			name:      "A 得分更小，A 胜出",
			forkPoint: 0x00,
			forkA:     0x01, // score = 0x01
			forkB:     0x02, // score = 0x02
			want:      ForkSideA,
		},
		{
			name:      "B 得分更小，B 胜出",
			forkPoint: 0x00,
			forkA:     0x05,
			forkB:     0x01,
			want:      ForkSideB,
		},
		{
			name:      "score 相同，比较 BlockID A<B",
			forkPoint: 0x00,
			forkA:     0x00,      // score = 0x00
			forkB:     0x00,      // score = 0x00（相同）；forkA[0]=0x01<forkB[0]=0x02 下面专门构造
			want:      ForkSideA, // 由下面直接调用覆盖
		},
	}

	for i, tc := range cases {
		if i == 2 {
			// score 相同场景：用 forkPoint=0x05, forkA[0]=0x05, forkB[0]=0x05
			// score = 0x05 XOR 0x05 = 0x00（全零，相同）；次字节 a2[1]=0x01 < b2[1]=0x02 → A 胜
			fp := makeBlockID(0x05)
			a2 := makeBlockID(0x05)
			b2 := makeBlockID(0x05)
			a2[1] = 0x01
			b2[1] = 0x02
			got, err := RandomXTiebreak(h, fp, a2, b2)
			if err != nil {
				t.Fatalf("case score-same: unexpected error: %v", err)
			}
			// score 全零相同 → 比 BlockID：a2[1]=0x01 < b2[1]=0x02 → A 胜
			if got != ForkSideA {
				t.Errorf("case score-same: got %d, want ForkSideA", got)
			}
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			fp := makeBlockID(tc.forkPoint)
			fa := makeBlockID(tc.forkA)
			fb := makeBlockID(tc.forkB)
			got, err := RandomXTiebreak(h, fp, fa, fb)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestRandomXTiebreakUnavailable(t *testing.T) {
	// 桩哈希器返回 ErrUnavailable → 函数应返回 ErrRandomXUnavailable
	stub := errorHasher{err: rxpkg.ErrUnavailable}
	_, err := RandomXTiebreak(stub, makeBlockID(0), makeBlockID(1), makeBlockID(2))
	if !errors.Is(err, ErrRandomXUnavailable) {
		t.Errorf("expected ErrRandomXUnavailable, got %v", err)
	}
}

func TestRandomXTiebreakFullyEqual(t *testing.T) {
	// seed XOR input 始终为 0x00（相同 score），且 BlockID 完全相同 → ForkSideNone
	h := deterministicHasher{}
	fp := makeBlockID(0x05)
	id := makeBlockID(0x05)
	got, err := RandomXTiebreak(h, fp, id, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != ForkSideNone {
		t.Errorf("expected ForkSideNone for identical IDs, got %d", got)
	}
}

func TestRandomXTiebreakScoreOrder(t *testing.T) {
	// 验证得分字节序比较是按字典序升序（较小者胜）
	// score_a=0x00...，score_b=0x01... → A 胜
	h := deterministicHasher{}
	fp := makeBlockID(0x00)
	// Hash(fp, forkA) = 0x00 XOR 0x00 = 0x00（首字节）→ smaller
	// Hash(fp, forkB) = 0x00 XOR 0xFF = 0xFF（首字节）→ larger
	forkA := makeBlockID(0x00)
	forkB := makeBlockID(0xFF)

	got, err := RandomXTiebreak(h, fp, forkA, forkB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != ForkSideA {
		t.Errorf("expected ForkSideA (score_a smaller), got %d", got)
	}

	// 验证 score 字典序（不是数值比较）：score_a 首字节 0x00，score_b 首字节 0x01
	scoreA, _ := h.Hash(fp[:], forkA[:])
	scoreB, _ := h.Hash(fp[:], forkB[:])
	if bytes.Compare(scoreA, scoreB) >= 0 {
		t.Errorf("expected scoreA < scoreB lexicographically")
	}
}
