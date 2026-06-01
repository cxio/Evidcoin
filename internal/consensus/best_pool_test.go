package consensus

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// mkCandidate 构造一个仅由首字节区分的候选者（MintHash 全部填 b）。
func mkCandidate(b byte) MintCandidate {
	var h types.MintHash
	for i := range h {
		h[i] = b
	}
	return MintCandidate{
		MintHash:   h,
		TxID:       types.MustTxID(bytes.Repeat([]byte{b}, 48)),
		MintPubKey: []byte{b},
	}
}

// TestBestPoolKeepsAscending 断言池内始终按 MintHash 升序（值小者优）。
func TestBestPoolKeepsAscending(t *testing.T) {
	p := NewBestPool()
	for _, b := range []byte{0x05, 0x01, 0x03, 0x02, 0x04} {
		p.Add(mkCandidate(b))
	}
	got := p.Candidates()
	if len(got) != 5 {
		t.Fatalf("size = %d, want 5", len(got))
	}
	for i := 1; i < len(got); i++ {
		if CompareMintCandidates(got[i-1], got[i]) >= 0 {
			t.Fatalf("not ascending at %d: %x then %x", i, got[i-1].MintHash[:1], got[i].MintHash[:1])
		}
	}
	if got[0].MintHash[0] != 0x01 || got[4].MintHash[0] != 0x05 {
		t.Fatalf("order wrong: first=%x last=%x", got[0].MintHash[:1], got[4].MintHash[:1])
	}
}

// TestBestPoolDedup 断言重复候选去重（三级全等视为同一候选）。
func TestBestPoolDedup(t *testing.T) {
	p := NewBestPool()
	c := mkCandidate(0x07)
	p.Add(c)
	p.Add(c)
	p.Add(mkCandidate(0x07))
	if p.Len() != 1 {
		t.Fatalf("dedup failed, len = %d, want 1", p.Len())
	}
}

// TestBestPoolCapacityEvictsWorst 断言容量上限 20，新优者挤出最差者（最大 MintHash）。
func TestBestPoolCapacityEvictsWorst(t *testing.T) {
	p := NewBestPool()
	// 填入 20 个候选，MintHash 首字节 10..29。
	for b := byte(10); b < 30; b++ {
		p.Add(mkCandidate(b))
	}
	if p.Len() != BestPoolCapacity {
		t.Fatalf("len = %d, want %d", p.Len(), BestPoolCapacity)
	}

	// 加入一个更差的（首字节 0xFF）——应被拒绝，不进池。
	if p.Add(mkCandidate(0xFF)) {
		t.Fatal("worse candidate must not enter full pool")
	}
	if p.Len() != BestPoolCapacity {
		t.Fatalf("len changed after rejecting worse, len = %d", p.Len())
	}

	// 加入一个更优的（首字节 0x01）——应进池并挤出当前最差（首字节 29）。
	if !p.Add(mkCandidate(0x01)) {
		t.Fatal("better candidate must enter full pool")
	}
	got := p.Candidates()
	if p.Len() != BestPoolCapacity {
		t.Fatalf("len = %d, want %d after eviction", p.Len(), BestPoolCapacity)
	}
	if got[0].MintHash[0] != 0x01 {
		t.Fatalf("best should be 0x01, got %x", got[0].MintHash[:1])
	}
	// 最差者（曾经的 29）应被挤出。
	for _, c := range got {
		if c.MintHash[0] == 29 {
			t.Fatal("worst candidate (29) should have been evicted")
		}
	}
}

// TestBestPoolAuthorizedSyncers 断言授权同步成员为后 15 名；前 5 名不在授权集内。
func TestBestPoolAuthorizedSyncers(t *testing.T) {
	p := NewBestPool()
	for b := byte(1); b <= 20; b++ {
		p.Add(mkCandidate(b))
	}
	auth := p.AuthorizedSyncers()
	if len(auth) != 15 {
		t.Fatalf("authorized = %d, want 15", len(auth))
	}
	// 授权集首项应为整体第 6 名（首字节 6）。
	if auth[0].MintHash[0] != 6 {
		t.Fatalf("first authorized = %x, want 6", auth[0].MintHash[:1])
	}
	if auth[14].MintHash[0] != 20 {
		t.Fatalf("last authorized = %x, want 20", auth[14].MintHash[:1])
	}

	// 前 5 名不得出现在授权集内。
	for _, a := range auth {
		if a.MintHash[0] <= 5 {
			t.Fatalf("top-5 member %x must not be authorized", a.MintHash[:1])
		}
	}
}

// TestBestPoolAuthorizedFewerThanSix 断言成员不足 6 个时授权集为空（无后 15 名）。
func TestBestPoolAuthorizedFewerThanSix(t *testing.T) {
	p := NewBestPool()
	for b := byte(1); b <= 5; b++ {
		p.Add(mkCandidate(b))
	}
	if len(p.AuthorizedSyncers()) != 0 {
		t.Fatalf("expected empty authorized set with 5 members")
	}
	// 第 6 个成员加入后，授权集应含 1 名（第 6 名）。
	p.Add(mkCandidate(6))
	auth := p.AuthorizedSyncers()
	if len(auth) != 1 || auth[0].MintHash[0] != 6 {
		t.Fatalf("expected single authorized member 6, got %d", len(auth))
	}
}

// TestBestPoolIsAuthorized 断言成员授权判定（是否在后 15 名内）。
func TestBestPoolIsAuthorized(t *testing.T) {
	p := NewBestPool()
	for b := byte(1); b <= 20; b++ {
		p.Add(mkCandidate(b))
	}
	if p.IsAuthorized(mkCandidate(1)) {
		t.Fatal("top-5 member 1 must not be authorized")
	}
	if !p.IsAuthorized(mkCandidate(6)) {
		t.Fatal("member 6 must be authorized")
	}
	if p.IsAuthorized(mkCandidate(0xFF)) {
		t.Fatal("non-member must not be authorized")
	}
}
