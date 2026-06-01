package utxo

import "github.com/cxio/evidcoin/pkg/types"

// Snapshot 是 UTXO 集在某一区块结算点的不可变快照，承载两类信息：
//
//   - 链身份与位置锚点：ChainID、GenesisID、Height、BlockID、StateRoot，
//     用于将状态绑定到唯一的链与高度，供更高层做完整性核对（第 09 章）。
//   - 拍照时刻的状态成员副本（私有 entries），供 Restore 原子回滚使用。
//
// 快照只在状态层内部用于「应用-失败-回滚」语义，不承担持久化职责；长期落盘与
// 网络同步不属于本层（见仓库分层约定）。
type Snapshot struct {
	// ChainID 是所属链的字符串标识（如主网/测试网名）。
	ChainID string
	// GenesisID 是创世区块头哈希，作为链身份的密码学锚点。
	GenesisID types.BlockID
	// Height 是拍照时已结算的区块高度。
	Height uint32
	// BlockID 是该高度区块头哈希。
	BlockID types.BlockID
	// StateRoot 是拍照时刻的 UTXO 五层宽成员状态指纹根（见 Store.Root）。
	StateRoot types.TreeHash
	// entries 是拍照时刻状态成员的副本，供回滚还原；不导出以防外部篡改。
	entries map[OutPoint]Entry
}

// Snapshot 对当前 UTXO 集拍照：计算状态 root，复制全部成员，并绑定链身份与
// 高度锚点。返回的快照与后续对 Store 的修改完全隔离（成员映射为独立副本，
// Entry 为值类型；其内的缓存字节切片不会被状态转移原地修改，故共享底层数组安全）。
func (s *Store) Snapshot(chainID string, genesisID, blockID types.BlockID, height uint32) Snapshot {
	return Snapshot{
		ChainID:   chainID,
		GenesisID: genesisID,
		Height:    height,
		BlockID:   blockID,
		StateRoot: s.Root(),
		entries:   copyEntries(s.entries),
	}
}

// Restore 用快照成员原子地覆盖当前状态集，回滚拍照之后的全部变更（新增、删除、
// 已花费标记）。覆盖使用快照成员的独立副本，故同一快照可重复用于多次回滚。
func (s *Store) Restore(snap Snapshot) {
	s.entries = copyEntries(snap.entries)
}

// copyEntries 返回成员映射的浅拷贝：键与 Entry 值逐一复制。Entry 内的字节切片
// 共享底层数组——状态转移只整体替换 Entry 值（如置 Spent）而不原地改写切片内容，
// 故对回滚而言此复制深度已足够。
func copyEntries(src map[OutPoint]Entry) map[OutPoint]Entry {
	dst := make(map[OutPoint]Entry, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
