package tx

import (
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// HashOutputs 计算交易输出根 HashOutputs = Hash256:Tree<Outputs>（第 04 章 §3.4）。
// 各输出按其规范 envelope 编码（Config||Payload||LockScript）作为叶 payload，按出现
// 顺序进入通用二叉哈希树（叶 tree.leaf profile，分支/单叶根 tree.branch profile）。
//
// 普通交易输出集不得为空（第 06 章 §7：币金输出数量须 >0），空集返回 ErrNoOutputs。
// 每个输出的 Serial 必须等于其位置下标，否则返回 ErrOutputSerialMismatch（序位是输出在
// 集合中位置的派生量，不进叶前像，但须自洽以保证 OutIndex 引用与 SIGOUT_SELF 语义正确）。
func HashOutputs(outputs []Output) (types.Hash32, error) {
	if len(outputs) == 0 {
		return types.Hash32{}, ErrNoOutputs
	}
	payloads := make([][]byte, len(outputs))
	for i := range outputs {
		if uint64(outputs[i].Serial) != uint64(i) {
			return types.Hash32{}, ErrOutputSerialMismatch
		}
		canon, err := outputs[i].appendCanonical(nil)
		if err != nil {
			return types.Hash32{}, err
		}
		payloads[i] = canon
	}
	tree, err := hashtree.BuildFromPayloads(payloads)
	if err != nil {
		return types.Hash32{}, err
	}
	return types.NewHash32(tree.Root())
}

// canonicalOutputs 返回输出集的交易体规范编码（第 06 章 §4）：
//
//	varint(count) || Output*
//
// 各输出按创建者给定顺序编码（Config||Payload||LockScript），不自动排序。
// 注意：此处的 count 前缀属交易体序列化，与输出哈希树（无 count 前缀）不同。
func canonicalOutputs(outputs []Output) ([]byte, error) {
	dst := types.AppendVarUint(nil, uint64(len(outputs)))
	var err error
	for i := range outputs {
		dst, err = outputs[i].appendCanonical(dst)
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}
