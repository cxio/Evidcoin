package blockchain

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// ErrBlockqsUnavailable Blockqs 服务不可用。
var ErrBlockqsUnavailable = errors.New("blockqs service unavailable")

// BlockqsClient 区块查询服务客户端接口。
// 本阶段提供 stub 实现，真实实现在阶段 8（服务接口）完成。
type BlockqsClient interface {
	// FetchHeader 从远程服务获取指定高度的区块头。
	FetchHeader(height int32) (*BlockHeader, error)

	// FetchHeaders 批量获取区块头（用于同步）。
	FetchHeaders(from, to int32) ([]*BlockHeader, error)

	// FetchHeaderByHash 按哈希获取区块头。
	FetchHeaderByHash(hash types.Hash384) (*BlockHeader, error)
}

// noopBlockqsClient 空实现，用于不需要远程获取的场景（测试或离线模式）。
type noopBlockqsClient struct{}

func (n *noopBlockqsClient) FetchHeader(height int32) (*BlockHeader, error) {
	return nil, ErrBlockqsUnavailable
}

func (n *noopBlockqsClient) FetchHeaders(from, to int32) ([]*BlockHeader, error) {
	return nil, ErrBlockqsUnavailable
}

func (n *noopBlockqsClient) FetchHeaderByHash(hash types.Hash384) (*BlockHeader, error) {
	return nil, ErrBlockqsUnavailable
}

// NoopBlockqsClient 返回一个不执行任何操作的 stub 客户端。
func NoopBlockqsClient() BlockqsClient {
	return &noopBlockqsClient{}
}
