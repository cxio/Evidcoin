package tx

import "github.com/cxio/evidcoin/pkg/types"

// MinTxIDPartLen 是输入短引用 TxIDPart 的最小字节长度（第 06 章 §5）。
// 短引用禁止固定长度，用户可自由延长；下限 16 字节防短引用碰撞攻击。
const MinTxIDPartLen = 16

// InputKind 标识输入引用的来源信元类别。该值用于本地结构验证，
// 不进入输入项规范编码（编码只含短引用三元组与解锁脚本，来源类别由
// 引用命中的 UTXO/UTCO 集决定）。数值与输出类型值（第 06 章 §6）对齐。
type InputKind uint8

const (
	// InputCoin 表示币金输入，来源从 UTXO 集检索。
	InputCoin InputKind = 1
	// InputCredit 表示凭信输入，来源从 UTCO 集检索。
	InputCredit InputKind = 2
	// InputProof 表示存证；存证不可作为输入源，结构验证一律拒绝。
	InputProof InputKind = 3
)

// OutPoint 是输入对来源输出的短引用三元组（第 06 章 §5）：
// 交易年度、TxID 前段局部引用、来源输出集下标序位。
type OutPoint struct {
	// Year 是来源交易的真实年度（按时间戳计），UTC 自然年。
	Year uint64
	// TxIDPart 是 TxID 前段局部短引用，长度必须 >= MinTxIDPartLen。
	TxIDPart []byte
	// OutIndex 是来源输出集下标序位。
	OutIndex uint64
}

// appendCanonical 将短引用三元组按规范编码追加到 dst：
// Year(varint) || TxIDPart(varint(len)||bytes) || OutIndex(varint)。
// 当 TxIDPart 短于 MinTxIDPartLen 时返回 ErrTxIDPartTooShort。
func (p OutPoint) appendCanonical(dst []byte) ([]byte, error) {
	if len(p.TxIDPart) < MinTxIDPartLen {
		return nil, ErrTxIDPartTooShort
	}
	dst = types.AppendVarUint(dst, p.Year)
	dst = types.AppendBytes(dst, p.TxIDPart)
	dst = types.AppendVarUint(dst, p.OutIndex)
	return dst, nil
}

// LeadInput 是交易的首领输入项（输入列表第 0 项）。首领输入固定为币金输入，
// 其来源公钥哈希参与输入根计算以便铸造者验证（见第 04 章 §3.3、第 11 章）。
type LeadInput struct {
	// Ref 是来源输出短引用。
	Ref OutPoint
	// UnlockScript 是解锁脚本，计入交易体并参与输入根，不属于可剪枝见证。
	UnlockScript []byte
}

// Kind 返回首领输入的来源信元类别，恒为 InputCoin。
func (in LeadInput) Kind() InputKind { return InputCoin }

// appendCanonical 追加首领输入的规范编码（短引用 || 解锁脚本）。
func (in LeadInput) appendCanonical(dst []byte) ([]byte, error) {
	return appendInputBody(dst, in.Ref, in.UnlockScript)
}

// RestInput 是首领之外的其余输入项，可为币金或凭信输入。
type RestInput struct {
	// Kind 是来源信元类别，必须为 InputCoin 或 InputCredit；InputProof 非法。
	Kind InputKind
	// Ref 是来源输出短引用。
	Ref OutPoint
	// UnlockScript 是解锁脚本，计入交易体并参与输入根。
	UnlockScript []byte
}

// appendCanonical 追加其余输入的规范编码。编码不含 Kind（来源类别由命中集决定）。
func (in RestInput) appendCanonical(dst []byte) ([]byte, error) {
	return appendInputBody(dst, in.Ref, in.UnlockScript)
}

// appendInputBody 追加单个输入项的规范编码：短引用三元组 || UnlockScript。
// 当 UnlockScript 超过 MaxUnlockScript 时返回 ErrUnlockScriptTooLong。
func appendInputBody(dst []byte, ref OutPoint, unlock []byte) ([]byte, error) {
	if len(unlock) > types.MaxUnlockScript {
		return nil, ErrUnlockScriptTooLong
	}
	dst, err := ref.appendCanonical(dst)
	if err != nil {
		return nil, err
	}
	dst = types.AppendBytes(dst, unlock)
	return dst, nil
}
