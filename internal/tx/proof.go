package tx

import (
	"github.com/cxio/evidcoin/pkg/types"
)

// Proof 是存证信元载荷（类型值 3，不入集，第 07 章 §2）。
// 编码顺序固定为 Creator || Title || Content || AttachmentID（第 07 章 §51）。
// 存证用于表达存在性，可被引用，但无接收者字段（不可转移）。
type Proof struct {
	// Creator 是创建者或创建者引用（<256 字节，可空）。
	Creator []byte
	// Title 是标题（<256 字节）。
	Title []byte
	// Content 是内容，最多 2KB。
	Content []byte
	// AttachmentID 是可选附件 ID 结构的已编码字节（结构见 attachment.go）；
	// 缺省（nil）以 varint(0) 编码并参与前像。
	AttachmentID []byte
}

// Payload 返回 Proof 载荷的规范编码（第 07 章 §51）。Creator/Title 须 <256 字节，
// Content 须 ≤2KB，否则返回相应错误。Creator 与 AttachmentID 作为可选字段编码。
func (p Proof) Payload() ([]byte, error) {
	if len(p.Creator) > maxShortField {
		return nil, ErrCreatorTooLong
	}
	if len(p.Title) > maxShortField {
		return nil, ErrTitleTooLong
	}
	if len(p.Content) > maxDescription {
		return nil, ErrDescriptionTooLong
	}
	if len(p.AttachmentID) > maxShortField {
		return nil, ErrAttachmentIDTooLong
	}
	dst := types.AppendBytes(nil, p.Creator)
	dst = types.AppendBytes(dst, p.Title)
	dst = types.AppendBytes(dst, p.Content)
	dst = types.AppendBytes(dst, p.AttachmentID)
	return dst, nil
}
