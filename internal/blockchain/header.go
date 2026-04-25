// Package blockchain 实现 Evidcoin 区块头链的管理与维护。
package blockchain

import (
	"encoding/binary"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// BlockHeader 区块头结构（112 字节常规，年块额外 +48 字节）。
type BlockHeader struct {
	Version   int32        // 协议版本号（4 字节）
	Height    int32        // 区块高度，从 0 开始（4 字节）
	PrevBlock types.Hash384 // 前一区块 SHA3-384 哈希（48 字节）
	CheckRoot types.Hash384 // 校验根（48 字节）
	Stakes    uint64       // 币权销毁量，单位：聪时（8 字节）
	YearBlock types.Hash384 // 前一年块哈希（仅 Height % 87661 == 0 时存在）
}

// IsYearBlock 返回该区块是否为年块（Height 是 BlocksPerYear 的整数倍且不为 0）。
func (h *BlockHeader) IsYearBlock() bool {
	return h.Height > 0 && int(h.Height)%types.BlocksPerYear == 0
}

// Hash 计算区块头的 SHA3-384 哈希，即区块 ID。
func (h *BlockHeader) Hash() types.Hash384 {
	return crypto.SHA3_384(h.Bytes())
}

// Bytes 将区块头序列化为字节序列（用于哈希计算）。
// 格式：Version(4) + Height(4) + PrevBlock(48) + CheckRoot(48) + Stakes(8) [+ YearBlock(48)]
func (h *BlockHeader) Bytes() []byte {
	size := 112
	if h.IsYearBlock() {
		size += 48
	}
	buf := make([]byte, size)
	binary.BigEndian.PutUint32(buf[0:4], uint32(h.Version))
	binary.BigEndian.PutUint32(buf[4:8], uint32(h.Height))
	copy(buf[8:56], h.PrevBlock[:])
	copy(buf[56:104], h.CheckRoot[:])
	binary.BigEndian.PutUint64(buf[104:112], h.Stakes)
	if h.IsYearBlock() {
		copy(buf[112:160], h.YearBlock[:])
	}
	return buf
}

// BlockTime 按高度计算区块时间戳（Unix 毫秒）。
// 时间戳不存储于区块头，通过公式确定性计算。
func BlockTime(genesisTimestamp int64, height int32) int64 {
	return genesisTimestamp + int64(height)*int64(types.BlockInterval.Milliseconds())
}
