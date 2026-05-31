package blockchain

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// ErrHeaderNotFound 表示按高度或 BlockID 查询区块头时未命中（含空 store 查询 tip）。
var ErrHeaderNotFound = errors.New("blockchain: header not found")

// HeaderStore 是区块头存储接口（第 05 章 §5、§6）。
// 只定义接口契约，生产级存储（含年块衔接式省略、Blockqs 完整性回填）延后实现。
// 测试用内存实现见 memstore_test.go。
type HeaderStore interface {
	// Put 写入一个区块头，同时建立按 BlockID 与按高度的索引。
	Put(h *BlockHeader) error
	// ByID 按 BlockID 查询区块头，未命中返回 ErrHeaderNotFound。
	ByID(id types.BlockID) (*BlockHeader, error)
	// ByHeight 按高度查询区块头，未命中返回 ErrHeaderNotFound。
	ByHeight(height uint32) (*BlockHeader, error)
	// Tip 返回当前最高高度的区块头，空 store 返回 ErrHeaderNotFound。
	Tip() (*BlockHeader, error)
}
