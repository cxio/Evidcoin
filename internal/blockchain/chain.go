package blockchain

import "github.com/cxio/evidcoin/pkg/types"

// Chain 管理单条区块头链的 tip、查询与最小入块验证（第 05 章 §4、§6）。
// 不实现分叉选择与长期分叉重组：手动切链是社会性「用脚投票」，非算法逻辑（§7）。
type Chain struct {
	store HeaderStore
}

// NewChain 基于给定 HeaderStore 构造 Chain。
func NewChain(store HeaderStore) *Chain {
	return &Chain{store: store}
}

// AddHeader 校验并入块 h，成功返回其计算出的 BlockID。
// 校验规则见 validateNext：创世初始化、高度连续、PrevBlock 衔接、同高度冲突拒绝。
func (c *Chain) AddHeader(h *BlockHeader) (types.BlockID, error) {
	if err := c.validateNext(h); err != nil {
		return types.BlockID{}, err
	}
	if err := c.store.Put(h); err != nil {
		return types.BlockID{}, err
	}
	return h.ID(), nil
}

// AddHeaderWithID 与 AddHeader 相同，但先校验调用方声明的 claimedID 与重算的
// BlockID 一致，不一致返回 ErrBlockIDMismatch。用于接收携带外部声明 ID 的提交。
func (c *Chain) AddHeaderWithID(h *BlockHeader, claimedID types.BlockID) (types.BlockID, error) {
	if h.ID() != claimedID {
		return types.BlockID{}, ErrBlockIDMismatch
	}
	return c.AddHeader(h)
}

// Tip 返回当前链尾区块头，空链返回 ErrHeaderNotFound。
func (c *Chain) Tip() (*BlockHeader, error) {
	return c.store.Tip()
}

// HeaderByHeight 按高度查询区块头。
func (c *Chain) HeaderByHeight(height uint32) (*BlockHeader, error) {
	return c.store.ByHeight(height)
}

// HeaderByID 按 BlockID 查询区块头。
func (c *Chain) HeaderByID(id types.BlockID) (*BlockHeader, error) {
	return c.store.ByID(id)
}
