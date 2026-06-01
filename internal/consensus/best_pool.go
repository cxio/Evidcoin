package consensus

import "sort"

// 择优池（第 11 章 §5）。

const (
	// BestPoolCapacity 是择优池最大容量（候选者数量上限）。
	BestPoolCapacity = 20
	// bestPoolTopExempt 是不具同步发起权的前序名次数量（前 5 名）。
	bestPoolTopExempt = 5
)

// BestPool 是按铸凭哈希三级升序排列、去重、容量受限的择优池。
// 值小者优先（排名靠前）；满池时仅当新候选优于当前最差者才进池并挤出最差者。
// 仅池中后 15 名成员具同步发起权（前 5 名因利益相关性被排除）。
//
// 注：预选窗口（评参 -8 号、最多提前 7 个区块时段得知对比目标）是择优流程的
// 时序约束，由调用方在合适时机注入候选者；本结构只维护池的有序去重与授权集。
type BestPool struct {
	// candidates 始终保持三级升序且无重复。
	candidates []MintCandidate
}

// NewBestPool 创建空择优池。
func NewBestPool() *BestPool {
	return &BestPool{candidates: make([]MintCandidate, 0, BestPoolCapacity)}
}

// Len 返回池内候选者数量。
func (p *BestPool) Len() int { return len(p.candidates) }

// Candidates 返回池内候选者的升序副本（值小者在前）。
func (p *BestPool) Candidates() []MintCandidate {
	out := make([]MintCandidate, len(p.candidates))
	copy(out, p.candidates)
	return out
}

// find 用二分查找定位与 c 三级全等的候选者下标；未找到返回应插入位置与 false。
func (p *BestPool) find(c MintCandidate) (idx int, found bool) {
	i := sort.Search(len(p.candidates), func(i int) bool {
		return CompareMintCandidates(p.candidates[i], c) >= 0
	})
	if i < len(p.candidates) && CompareMintCandidates(p.candidates[i], c) == 0 {
		return i, true
	}
	return i, false
}

// Add 尝试将候选者加入池中：
//   - 重复候选（三级全等）去重，返回 false；
//   - 未满时按序插入，返回 true；
//   - 满池时仅当 c 优于当前最差者才插入并挤出最差者，返回 true；否则返回 false。
func (p *BestPool) Add(c MintCandidate) bool {
	pos, found := p.find(c)
	if found {
		return false
	}
	if len(p.candidates) >= BestPoolCapacity {
		// 满池：c 必须优于当前最差者（末位）才可进池。
		if pos >= BestPoolCapacity {
			return false
		}
		// 先在 pos 处插入，再截断到容量上限（挤出末位最差者）。
		p.insertAt(pos, c)
		p.candidates = p.candidates[:BestPoolCapacity]
		return true
	}
	p.insertAt(pos, c)
	return true
}

// insertAt 在下标 pos 处就地插入候选者，保持切片其余顺序不变。
func (p *BestPool) insertAt(pos int, c MintCandidate) {
	p.candidates = append(p.candidates, MintCandidate{})
	copy(p.candidates[pos+1:], p.candidates[pos:])
	p.candidates[pos] = c
}

// AuthorizedSyncers 返回有同步发起权的成员（池中后 15 名，即跳过前 5 名）。
// 成员不足 6 个时返回空集。返回升序副本。
func (p *BestPool) AuthorizedSyncers() []MintCandidate {
	if len(p.candidates) <= bestPoolTopExempt {
		return nil
	}
	tail := p.candidates[bestPoolTopExempt:]
	out := make([]MintCandidate, len(tail))
	copy(out, tail)
	return out
}

// IsAuthorized 判定候选者是否在授权同步集内（池中后 15 名）。
func (p *BestPool) IsAuthorized(c MintCandidate) bool {
	idx, found := p.find(c)
	if !found {
		return false
	}
	return idx >= bestPoolTopExempt
}
