package tx

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// AttachmentID 附件标识结构（总长 < 256 字节）。
type AttachmentID struct {
	// TotalLen ID 总字节长
	TotalLen byte
	// MajorType 附件大类（参考 MIME）
	MajorType byte
	// MinorType 附件小类
	MinorType byte
	// Fingerprint SHA3-512 哈希指纹（固定 64 字节，对完整附件数据）
	Fingerprint types.Hash512
	// ShardCount 分片数量（0 = 无分片字段，1 = 单体有哈希，>1 = 分片树）
	ShardCount uint16
	// ShardTreeRoot 片组哈希树根（BLAKE3-256，32 字节）——条件存在
	ShardTreeRoot *types.Hash256
	// DataSize 附件大小（字节）
	DataSize int64
}

// Validate 验证 AttachmentID 的结构合法性。
func (a *AttachmentID) Validate() error {
	if a.ShardCount == 0 && a.ShardTreeRoot != nil {
		return errors.New("shard tree root must be absent when shard count is 0")
	}
	if a.ShardCount > 0 && a.ShardTreeRoot == nil {
		return errors.New("shard tree root required when shard count > 0")
	}
	return nil
}

// Encode 将 AttachmentID 序列化为字节序列。
func (a *AttachmentID) Encode() []byte {
	buf := make([]byte, 0, 16)
	buf = append(buf, a.TotalLen)
	buf = append(buf, a.MajorType)
	buf = append(buf, a.MinorType)
	buf = append(buf, a.Fingerprint[:]...)
	sc := make([]byte, 2)
	sc[0] = byte(a.ShardCount >> 8)
	sc[1] = byte(a.ShardCount)
	buf = append(buf, sc...)
	if a.ShardTreeRoot != nil {
		buf = append(buf, a.ShardTreeRoot[:]...)
	}
	buf = append(buf, AppendVarint(nil, uint64(a.DataSize))...)
	return buf
}
