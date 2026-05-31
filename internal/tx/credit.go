package tx

import "github.com/cxio/evidcoin/pkg/types"

// maxDescription 是 Credit.Description / Proof.Content 的最大长度（2KB，第 07 章 §2）。
const maxDescription = 2048

// MaxCreditOutputsPerTx 是单笔交易允许的最大凭信输出数量（第 07 章 §5）。
// 第 2 笔触发交易费加倍规则（结算见状态/共识层），第 3 笔协议拒绝。
const MaxCreditOutputsPerTx = 2

// CreditMaxAge 是凭信可被引用花销的最大币龄（区块高度差）：31 × 87661。
// 失效条件为 age > CreditMaxAge；age == CreditMaxAge 时该区块仍可被引用花销（DEC-0101）。
const CreditMaxAge = 31 * types.BlocksPerYear

// Credit 是凭信信元载荷（类型值 2，入 UTCO，第 07 章 §2）。
// 编码顺序固定为 Receiver || Creator || Title || Description || AttachmentID（DEC-0101）。
type Credit struct {
	// Receiver 是接收者（<256 字节）。
	Receiver []byte
	// Creator 是创建者或创建者引用（<256 字节）。
	Creator []byte
	// Title 是标题（<256 字节，通常可读）。
	Title []byte
	// Description 是描述，最多 2KB。
	Description []byte
	// AttachmentID 是可选附件 ID 结构的已编码字节（结构见 attachment.go）；
	// 缺省（nil）以 varint(0) 编码并参与前像。
	AttachmentID []byte
}

// Payload 返回 Credit 载荷的规范编码（第 07 章 §2）。各短字段须 <256 字节，
// Description 须 ≤2KB，否则返回相应错误。AttachmentID 作为可选字段编码。
func (c Credit) Payload() ([]byte, error) {
	if len(c.Receiver) > maxShortField {
		return nil, ErrReceiverTooLong
	}
	if len(c.Creator) > maxShortField {
		return nil, ErrCreatorTooLong
	}
	if len(c.Title) > maxShortField {
		return nil, ErrTitleTooLong
	}
	if len(c.Description) > maxDescription {
		return nil, ErrDescriptionTooLong
	}
	if len(c.AttachmentID) > maxShortField {
		return nil, ErrAttachmentIDTooLong
	}
	dst := types.AppendBytes(nil, c.Receiver)
	dst = types.AppendBytes(dst, c.Creator)
	dst = types.AppendBytes(dst, c.Title)
	dst = types.AppendBytes(dst, c.Description)
	dst = types.AppendBytes(dst, c.AttachmentID)
	return dst, nil
}

// CreditExpired 报告给定币龄（age，区块高度差）的凭信是否已过期失效。
// 失效条件为 age > CreditMaxAge；边界相等仍可被引用花销（DEC-0101）。
func CreditExpired(age uint64) bool {
	return age > CreditMaxAge
}

// ValidateCreditOutputCount 校验输出集中的凭信输出数量不超过 MaxCreditOutputsPerTx。
// 仅统计入状态集的标准凭信输出（自定义类不计）。超过则返回 ErrTooManyCreditOutputs。
func ValidateCreditOutputCount(outputs []Output) error {
	count := 0
	for i := range outputs {
		o := outputs[i]
		if !o.IsCustom && o.Type == TypeCredit {
			count++
		}
	}
	if count > MaxCreditOutputsPerTx {
		return ErrTooManyCreditOutputs
	}
	return nil
}
