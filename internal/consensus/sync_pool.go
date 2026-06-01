package consensus

// 择优池三段同步与防重放（第 11 章 §5）。
//
// 同步分段时序（由调用方按区块时段驱动，本层不实现 P2P 线格式与定时器）：
//   - 新区块创建后到 -6 号区块前的 5 个区块时段为「广播收集」（约 30min 裕度）；
//   - 成为 -6 号后进入 2 个区块时段的「同步优化」；
//   - 成为 -8 号即定型为评参区块。
//
// 同步流程三段：
//   1. 裁判池 Referee：本地池副本与对端目标池合并，判断对端是否落在合并池后 15 位；
//   2. 预合并 PreMerge：广播收集截止后合并暂存目标池，取后 15 位为授权节点集；
//   3. 终合并 FinalMerge：合并授权集成员目标池得最终择优池。
//
// 同步为 P2P 概略性约束而非唯一性精确约束；若最终择优池无唯一性导致分叉，
// 由第 12 章分叉竞争收敛。

// NodeID 是同步发起节点的定长标识，用作防重放映射键。
// 具体取值（如节点公钥哈希）由上层 P2P 服务定义；本层只要求可比较。
type NodeID [32]byte

// SyncSession 承载一次择优池同步会话：持有本地池副本与防重放记录。
type SyncSession struct {
	// local 是本地择优池副本（裁判池的基准）。新上线节点本地池可为空。
	local *BestPool
	// seen 记录已对本会话目标池发起过同步的授权节点，用于防重放。
	seen map[NodeID]bool
}

// NewSyncSession 以给定本地池创建同步会话。local 为空（nil 或零候选）不影响裁判池创建。
func NewSyncSession(local *BestPool) *SyncSession {
	if local == nil {
		local = NewBestPool()
	}
	return &SyncSession{
		local: local,
		seen:  make(map[NodeID]bool),
	}
}

// MergePools 合并两个择优池：并集去重，按 MintHash 三级升序排列并截断到容量上限 20。
// 任一入参为空均不影响另一方的顺序。返回新池，不修改入参。
func MergePools(a, b *BestPool) *BestPool {
	out := NewBestPool()
	if a != nil {
		for _, c := range a.candidates {
			out.Add(c)
		}
	}
	if b != nil {
		for _, c := range b.candidates {
			out.Add(c)
		}
	}
	return out
}

// Referee 实现裁判池阶段：将本地池与对端目标池合并，判断发起同步的对端候选者
// peer 是否落在合并池后 15 位（授权区）。返回 true 表示对端有资格作为授权发起方。
func (s *SyncSession) Referee(target *BestPool, peer MintCandidate) bool {
	merged := MergePools(s.local, target)
	return merged.IsAuthorized(peer)
}

// AcceptSync 接受来自授权节点 id 的一次目标池同步，并入本地池。
// 同一节点对同一会话重复发起将被拒绝（防重放，返回 ErrSyncReplay）。
// 合并后本地池保持有序去重并截断到 20。
func (s *SyncSession) AcceptSync(id NodeID, target *BestPool) error {
	if s.seen[id] {
		return ErrSyncReplay
	}
	s.seen[id] = true
	s.local = MergePools(s.local, target)
	return nil
}

// PreMerge 实现预合并阶段：合并本地池与所有暂存目标池，取合并池后 15 位作为
// 授权节点集（升序）。成员不足 6 个时返回空集。该步不修改会话本地池。
func (s *SyncSession) PreMerge(targets []*BestPool) []MintCandidate {
	merged := s.local
	for _, t := range targets {
		merged = MergePools(merged, t)
	}
	return merged.AuthorizedSyncers()
}

// FinalMerge 实现终合并阶段：合并本地池与各暂存目标池，产出最终择优池
// （有序去重、截断到 20）。返回新池，不修改会话本地池。
//
// 同步为概略性约束：本函数只做合并归一，不保证全网唯一性；非唯一时由第 12 章收敛。
func (s *SyncSession) FinalMerge(targets []*BestPool) *BestPool {
	merged := s.local
	for _, t := range targets {
		merged = MergePools(merged, t)
	}
	return merged
}
