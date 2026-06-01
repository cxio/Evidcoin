package consensus

import (
	"bytes"
	"testing"
)

// poolOf 构造一个含给定首字节候选者的择优池。
func poolOf(bs ...byte) *BestPool {
	p := NewBestPool()
	for _, b := range bs {
		p.Add(mkCandidate(b))
	}
	return p
}

// syncerID 由候选者首字节生成一个稳定的节点标识。
func syncerID(b byte) NodeID {
	var id NodeID
	copy(id[:], bytes.Repeat([]byte{b}, len(id)))
	return id
}

// TestMergePoolsAscendingTruncated 断言合并后按 MintHash 升序并截断到 20。
func TestMergePoolsAscendingTruncated(t *testing.T) {
	local := NewBestPool()
	for b := byte(1); b <= 15; b++ {
		local.Add(mkCandidate(b))
	}
	target := NewBestPool()
	for b := byte(10); b <= 30; b++ {
		target.Add(mkCandidate(b))
	}
	merged := MergePools(local, target)
	if merged.Len() != BestPoolCapacity {
		t.Fatalf("merged len = %d, want %d", merged.Len(), BestPoolCapacity)
	}
	got := merged.Candidates()
	for i := 1; i < len(got); i++ {
		if CompareMintCandidates(got[i-1], got[i]) >= 0 {
			t.Fatalf("merged not ascending at %d", i)
		}
	}
	// 合并并集前 20 个最小者：首字节 1..20。
	if got[0].MintHash[0] != 1 || got[19].MintHash[0] != 20 {
		t.Fatalf("merged range wrong: first=%x last=%x", got[0].MintHash[:1], got[19].MintHash[:1])
	}
}

// TestMergePoolsEmptyLocal 断言新上线节点本地池为空不影响合并（裁判池创建）。
func TestMergePoolsEmptyLocal(t *testing.T) {
	local := NewBestPool() // 空
	target := poolOf(3, 1, 2)
	merged := MergePools(local, target)
	if merged.Len() != 3 {
		t.Fatalf("merged len = %d, want 3", merged.Len())
	}
	if merged.Candidates()[0].MintHash[0] != 1 {
		t.Fatal("empty local must not corrupt merge order")
	}
}

// TestRefereeAcceptsRearMember 断言裁判池：对端落在合并池后 15 位才被接受为授权发起方。
func TestRefereeAcceptsRearMember(t *testing.T) {
	local := NewBestPool()
	for b := byte(1); b <= 20; b++ {
		local.Add(mkCandidate(b))
	}
	target := poolOf(6, 25) // 对端目标池
	s := NewSyncSession(local)

	// 对端候选者为首字节 6（合并后处于后 15 位）→ 接受。
	if !s.Referee(target, mkCandidate(6)) {
		t.Fatal("rear-15 peer must be accepted as authorized syncer")
	}
	// 对端候选者为首字节 1（前 5 名）→ 拒绝。
	if s.Referee(target, mkCandidate(1)) {
		t.Fatal("top-5 peer must not be authorized")
	}
}

// TestSyncReplayRejected 断言同一授权节点对同一目标池仅一次同步权。
func TestSyncReplayRejected(t *testing.T) {
	local := poolOf(1, 2, 3, 4, 5, 6, 7)
	s := NewSyncSession(local)
	target := poolOf(6, 7, 8)
	id := syncerID(6)

	if err := s.AcceptSync(id, target); err != nil {
		t.Fatalf("first sync should succeed: %v", err)
	}
	if err := s.AcceptSync(id, target); err != ErrSyncReplay {
		t.Fatalf("replay must be rejected, got %v", err)
	}
	// 不同节点仍可同步。
	if err := s.AcceptSync(syncerID(7), target); err != nil {
		t.Fatalf("different node sync should succeed: %v", err)
	}
}

// TestPreMergeAndFinalMerge 断言预合并取后 15 位授权集、终合并产出最终择优池。
func TestPreMergeAndFinalMerge(t *testing.T) {
	local := NewBestPool()
	for b := byte(1); b <= 20; b++ {
		local.Add(mkCandidate(b))
	}
	s := NewSyncSession(local)

	// 收集到的多个目标池（暂存）。
	targets := []*BestPool{
		poolOf(6, 7, 21),
		poolOf(8, 9, 22),
		poolOf(10, 23),
	}
	// 预合并：合并本地池与所有暂存目标池，取后 15 位为授权节点集。
	authorized := s.PreMerge(targets)
	if len(authorized) != 15 {
		t.Fatalf("authorized set = %d, want 15", len(authorized))
	}
	// 授权集不得含前 5 名（首字节 1..5）。
	for _, a := range authorized {
		if a.MintHash[0] <= 5 {
			t.Fatalf("top-5 member %x must not be in authorized set", a.MintHash[:1])
		}
	}

	// 终合并：合并授权集成员目标池得最终择优池。
	final := s.FinalMerge(targets)
	if final.Len() != BestPoolCapacity {
		t.Fatalf("final pool len = %d, want %d", final.Len(), BestPoolCapacity)
	}
	got := final.Candidates()
	for i := 1; i < len(got); i++ {
		if CompareMintCandidates(got[i-1], got[i]) >= 0 {
			t.Fatalf("final pool not ascending at %d", i)
		}
	}
}

// TestNodeIDZeroValue 仅确保 NodeID 为定长可比较类型（用于防重放映射键）。
func TestNodeIDZeroValue(t *testing.T) {
	var a, b NodeID
	if a != b {
		t.Fatal("zero NodeID must compare equal")
	}
}
