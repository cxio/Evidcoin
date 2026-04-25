package blockchain

import "github.com/cxio/evidcoin/pkg/types"

// ForkInfo 分叉信息。
type ForkInfo struct {
	// ForkHeight 分叉发生的高度（两链首次出现分歧的高度）
	ForkHeight int32
	// LocalTip 本地链顶区块头
	LocalTip *BlockHeader
	// RemoteTip 远程链顶区块头
	RemoteTip *BlockHeader
	// RemoteLength 远程链自分叉点后的区块数
	RemoteLength int
}

// DetectFork 检测与指定节点之间的分叉。
// peers 为目标节点的地址列表（本阶段为 stub，真实实现在阶段 7/8）。
func (bc *Blockchain) DetectFork(peers []string) (*ForkInfo, error) {
	// 本阶段返回 nil（stub），共识层实现真实逻辑
	return nil, nil
}

// SwitchChain 切换到替代链。
// 这是一个需要用户明确确认的危险操作。
// forkHeight 为分叉高度，headers 为分叉点之后的替代链区块头序列。
func (bc *Blockchain) SwitchChain(forkHeight int32, headers []*BlockHeader) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 验证替代链的区块头连续性
	for i := 1; i < len(headers); i++ {
		prevHash := headers[i-1].Hash()
		if headers[i].PrevBlock != prevHash {
			return ErrInvalidPrevBlock
		}
		if headers[i].Height != headers[i-1].Height+1 {
			return ErrInvalidHeight
		}
	}
	// 写入替代链区块头
	for _, h := range headers {
		if err := bc.store.Put(h); err != nil {
			return err
		}
	}
	return nil
}

// BootstrapResult 初始验证结果。
type BootstrapResult struct {
	// ChainTip 验证通过的链顶区块头
	ChainTip *BlockHeader
	// Height 当前链高度
	Height int32
	// Confidence 置信度（基于多源一致性，0.0–1.0）
	Confidence float64
	// Sources 参与验证的数据源数量
	Sources int
}

// BootstrapVerify 初始主链验证（stub 实现）。
// 从多个数据源获取主链信息并交叉验证，确认目标主链的合法性。
func (bc *Blockchain) BootstrapVerify(sources []string) (*BootstrapResult, error) {
	// 本阶段返回空结果，真实实现在服务接口阶段完成
	tip, err := bc.store.Tip()
	if err != nil {
		return nil, err
	}
	return &BootstrapResult{
		ChainTip:   tip,
		Height:     tip.Height,
		Confidence: 1.0,
		Sources:    0,
	}, nil
}

// BoundIDFromHeader 从指定区块头计算 BoundID（取区块 ID 前 20 字节）。
// 通常取 -29 号区块（当前高度减 29）的区块 ID 前 20 字节。
func BoundIDFromHeader(h *BlockHeader) []byte {
	hash := h.Hash()
	bound := make([]byte, 20)
	copy(bound, hash[:20])
	return bound
}

// UpdateBoundID 更新链标识中的 BoundID（分叉确定后调用）。
func (bc *Blockchain) UpdateBoundID(height int32) error {
	h, err := bc.HeaderByHeight(height)
	if err != nil {
		return err
	}
	if bc.identity != nil {
		bc.identity.BoundID = BoundIDFromHeader(h)
	}
	return nil
}

// GenesisID 返回创世区块的区块 ID。
func (bc *Blockchain) GenesisID() (types.Hash384, error) {
	genesis, err := bc.HeaderByHeight(0)
	if err != nil {
		return types.Hash384{}, err
	}
	return genesis.Hash(), nil
}
