package tx

import "github.com/cxio/evidcoin/pkg/types"

// AttachmentID 是附件 ID 结构（第 07 章 §4，权威见 5.信用结构.md#附件id的结构）。
// 附件本身由外部数据网络（Depots）存储，交易仅携带此 ID 结构（整体 <256 字节）。
//
// 规范编码（Encode）顺序：
//
//	Type(2B) || Fingerprint(64B) || PieceCount(2B,BE) ||
//	[GroupHash(32B) if PieceCount>=1] || Size(varint)
//
// 编码长度由外层 varint(length) 表达（第 07 章 §4），编码字节数须 <256。
type AttachmentID struct {
	// Type 是附件类型：前字节大类、后字节小类（参考 HTML:MIME 分类）。
	Type [2]byte
	// Fingerprint 是附件指纹（SHA3-512，对数据本身完整哈希，域标签 attachment.fingerprint）。
	Fingerprint types.AttachmentHash
	// PieceCount 是分片数量（<65536）。0 表示无分片且省略 GroupHash；
	// 1 表示无分片但 GroupHash 正常计算；>1 表示分片，GroupHash 为含序校验树根。
	PieceCount uint16
	// GroupHash 是片组哈希（BLAKE3-256，无域标签）。仅 PieceCount>=1 时编码。
	GroupHash types.TreeHash
	// Size 是附件大小（字节，varint）。值由用户设置，节点不核查真实性。
	Size uint64
}

// Encode 返回附件 ID 结构的规范编码（第 07 章 §4）。
// PieceCount 为 0 时省略片组哈希字段；>=1 时编码 32 字节片组哈希。
// 当编码字节数超过 255 时返回 ErrAttachmentIDTooLong。
func (a AttachmentID) Encode() ([]byte, error) {
	var body []byte
	body = append(body, a.Type[0], a.Type[1])
	body = append(body, a.Fingerprint.Bytes()...)
	body = types.AppendUint16BE(body, a.PieceCount)
	if a.PieceCount >= 1 {
		body = append(body, a.GroupHash.Bytes()...)
	}
	body = types.AppendVarUint(body, a.Size)
	if len(body) > 255 {
		return nil, ErrAttachmentIDTooLong
	}
	return body, nil
}
