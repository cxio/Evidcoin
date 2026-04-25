package blockchain

import "github.com/cxio/evidcoin/pkg/types"

// ChainIdentity 链标识信息，用于 P2P 握手与交易签名。
type ChainIdentity struct {
	// ProtocolID 协议标识，如 "Evidcoin@V1"
	ProtocolID string
	// ChainID 运行态标识，如 "mainnet"
	ChainID string
	// GenesisID 创世区块 ID（SHA3-384）
	GenesisID types.Hash384
	// BoundID 主链绑定（可选），取 -29 号区块 ID 前 20 字节
	// 为 nil 时表示未绑定
	BoundID []byte
}

// Bytes 将链标识序列化为字节序列，用于参与交易签名前置。
// 格式：ProtocolID_len(1) + ProtocolID + ChainID_len(1) + ChainID + GenesisID(48) + BoundID
func (ci *ChainIdentity) Bytes() []byte {
	pid := []byte(ci.ProtocolID)
	cid := []byte(ci.ChainID)
	size := 1 + len(pid) + 1 + len(cid) + types.Hash384Len
	if len(ci.BoundID) > 0 {
		size += len(ci.BoundID)
	}
	buf := make([]byte, 0, size)
	buf = append(buf, byte(len(pid)))
	buf = append(buf, pid...)
	buf = append(buf, byte(len(cid)))
	buf = append(buf, cid...)
	buf = append(buf, ci.GenesisID[:]...)
	if len(ci.BoundID) > 0 {
		buf = append(buf, ci.BoundID...)
	}
	return buf
}
