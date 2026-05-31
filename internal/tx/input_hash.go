package tx

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// Inputs 是一笔普通交易的完整输入集：一个首领币金输入加零或多个其余输入。
// 普通交易必须至少有 Coin 首领输入（第 06 章 §5；Coinbase 无输入，另见 coinbase.go）。
type Inputs struct {
	// Lead 是首领币金输入（输入列表第 0 项）。
	Lead LeadInput
	// Rest 是其余输入项，按创建者给定顺序排列，不自动排序。
	Rest []RestInput
}

// Validate 执行输入集的本地结构验证：其余输入的来源类别必须为
// InputCoin 或 InputCredit，InputProof 一律拒绝（存证不可作输入源）。
// 短引用与脚本长度约束在编码阶段（canonicalList）强制。
func (in Inputs) Validate() error {
	for i := range in.Rest {
		k := in.Rest[i].Kind
		if k != InputCoin && k != InputCredit {
			return ErrInputKindInvalid
		}
	}
	_, err := in.canonicalList()
	return err
}

// canonicalList 返回输入集的规范列表编码（即交易体 Inputs 字段）：
//
//	varint(count) || LeadInput || RestInput*
//
// count 为首领加其余输入的总数。该字节序列同时是 ListHash 的前像。
func (in Inputs) canonicalList() ([]byte, error) {
	count := 1 + len(in.Rest)
	dst := types.AppendVarUint(nil, uint64(count))
	dst, err := in.Lead.appendCanonical(dst)
	if err != nil {
		return nil, err
	}
	for i := range in.Rest {
		dst, err = in.Rest[i].appendCanonical(dst)
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}

// ListHash 计算输入项串联哈希 ListHash = SHA3-384(canonicalList)（第 04 章 §3.3）。
func (in Inputs) ListHash() (types.Hash48, error) {
	b, err := in.canonicalList()
	if err != nil {
		return types.Hash48{}, err
	}
	return crypto.HashInputList(b), nil
}

// HashInputs 计算交易输入根 HashInputs = BLAKE3-256(ListHash || LeadPKHash)
// （第 04 章 §3.3）。leadPKHash 为首领输入来源输出的公钥哈希（验证时从 UTXO 查得），
// 在本层作为参数注入，参与铸造者验证。
func (in Inputs) HashInputs(leadPKHash []byte) (types.Hash32, error) {
	lh, err := in.ListHash()
	if err != nil {
		return types.Hash32{}, err
	}
	return crypto.HashInputRoot(lh.Bytes(), leadPKHash), nil
}
