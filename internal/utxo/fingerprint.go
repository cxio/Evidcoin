package utxo

import (
	"bytes"
	"sort"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// groupByteIndices 是三级中间分层使用的 0-based TxID 字节索引（DEC-0201）：
// 分别取 TxID 第 8、12、16 个字节值分组。三者均落在 DEC-0101 最短输入引用
// （TxIDPart >= 16 字节）覆盖范围内。
var groupByteIndices = [3]int{7, 11, 15}

// terminalLevel 是末端叶子层的层号：年度(0) → [7](1) → [11](2) → [15](3) → 叶(4)。
const terminalLevel = len(groupByteIndices) + 1

// leafRecord 是一个 TxID 分组在状态树中的末端叶：年度、完整 TxID 与叶哈希。
type leafRecord struct {
	year uint64
	txid types.TxID
	hash []byte
}

// 状态指纹末端叶子（第 09 章 §3-§4、DEC-0201）。
//
// 末端叶前像为 `TxID || Count || FlagBytes`，叶哈希 = `SHA3-384(DomainTag("utxo.leaf") || 前像)`。
// 其中：
//   - Count 是该 TxID 的有效（未花费）输出数量，以 ULEB128 变长整数编码（DEC-0001）。
//   - FlagBytes 第 i 位对应输出序位 i，每字节低位优先；1=未花费，0=已花费/无效；
//     尾部未用位为 0。
//   - 输出详情（金额/接收者/脚本）属缓存集，不进入前像。

// flagOutputs 由同一 TxID 的若干 entry 计算状态位集合与有效输出数。
//
// FlagBytes 长度仅由有效（未花费）输出的最大序位决定：UTXO 集语义上只含未花费
// 成员，已花费输出已移出集合，故其序位既不置位也不参与长度，确保逆向推导
// （第 09 章 §7）可在仅有未花费集合时复现同一叶。无有效输出时返回 count=0、
// flagBytes=nil（该 TxID 不产生叶，第 09 章 §6）。
func flagOutputs(entries []Entry) (count uint64, flagBytes []byte) {
	maxSerial := -1
	for _, e := range entries {
		if e.Spent {
			continue
		}
		if int(e.OutIndex) > maxSerial {
			maxSerial = int(e.OutIndex)
		}
	}
	if maxSerial < 0 {
		return 0, nil
	}
	flagBytes = make([]byte, maxSerial/8+1)
	for _, e := range entries {
		if e.Spent {
			continue
		}
		i := e.OutIndex
		flagBytes[i/8] |= 1 << (i % 8)
		count++
	}
	return count, flagBytes
}

// leafPreimage 构造末端叶前像（不含域标签）：`TxID || varuint(Count) || FlagBytes`。
func leafPreimage(txid types.TxID, count uint64, flagBytes []byte) []byte {
	dst := make([]byte, 0, len(txid)+10+len(flagBytes))
	dst = append(dst, txid[:]...)
	dst = types.AppendVarUint(dst, count)
	dst = append(dst, flagBytes...)
	return dst
}

// leafHash 计算 UTXO 末端叶哈希：`SHA3-384(DomainTag("utxo.leaf") || leafPreimage)`。
func leafHash(txid types.TxID, count uint64, flagBytes []byte) types.Hash48 {
	return crypto.HashUTXOLeaf(leafPreimage(txid, count, flagBytes))
}

// Root 计算当前 UTXO 集的五层宽成员状态指纹根（第 09 章 §2、§5、DEC-0201）：
//
//   - 顶层按年度数值升序，其下三级按 TxID 字节 [7]/[11]/[15] 升序分组，末端按
//     完整 TxID 字典序排列；空年度/空分组不编码。
//   - 各中间层与末端组复用 pkg/hashtree 通用二叉树构造（tree.branch 分支域、
//     奇数层提升、单叶根一元归一化），见 proposal 04 §3.5；末端叶为 utxo.leaf
//     域 SHA3-384（见 leafHash）。
//   - 集合为空时返回专用空根 BLAKE3-256(DomainTag("utxo.empty"))，非全零。
//
// 仅未花费输出参与；已花费输出既不计入叶状态位也不影响分层（UTXO 集语义）。
func (s *Store) Root() types.TreeHash {
	recs := s.leafRecords()
	if len(recs) == 0 {
		return crypto.EmptyUTXORoot()
	}
	sortLeafRecords(recs)
	var out types.TreeHash
	copy(out[:], buildWideTree(recs, 0))
	return out
}

// leafRecords 将所有未花费 entry 按 (年度, 完整 TxID) 归并为末端叶记录；
// 无有效输出的分组（count==0）不产生叶。
func (s *Store) leafRecords() []leafRecord {
	type txKey struct {
		year uint64
		txid types.TxID
	}
	groups := make(map[txKey][]Entry)
	for _, e := range s.entries {
		if e.Spent {
			continue
		}
		k := txKey{year: e.Year, txid: e.TxID}
		groups[k] = append(groups[k], e)
	}
	recs := make([]leafRecord, 0, len(groups))
	for k, es := range groups {
		count, flags := flagOutputs(es)
		if count == 0 {
			continue
		}
		recs = append(recs, leafRecord{
			year: k.year,
			txid: k.txid,
			hash: leafHash(k.txid, count, flags).Bytes(),
		})
	}
	return recs
}

// sortLeafRecords 按 (年度升序, TxID[7], TxID[11], TxID[15], 完整 TxID 字典序)
// 对末端叶记录排序，确立宽成员树的分层与组内次序。
func sortLeafRecords(recs []leafRecord) {
	sort.Slice(recs, func(i, j int) bool {
		a, b := recs[i], recs[j]
		if a.year != b.year {
			return a.year < b.year
		}
		for _, idx := range groupByteIndices {
			if a.txid[idx] != b.txid[idx] {
				return a.txid[idx] < b.txid[idx]
			}
		}
		return bytes.Compare(a.txid[:], b.txid[:]) < 0
	})
}

// buildWideTree 自顶向下对已排序的叶记录做分层归约：当前 level 按其分级键将
// 连续等键记录划入同一子组，递归构造子组根，再以通用二叉树合并本层子组。
// recs 必须非空且已排序。
func buildWideTree(recs []leafRecord, level int) []byte {
	if level == terminalLevel {
		leaves := make([][]byte, len(recs))
		for i, r := range recs {
			leaves[i] = r.hash
		}
		t, _ := hashtree.BuildTree(leaves)
		return t.Root()
	}
	var children [][]byte
	for i := 0; i < len(recs); {
		j := i + 1
		for j < len(recs) && levelKeyEqual(recs[i], recs[j], level) {
			j++
		}
		children = append(children, buildWideTree(recs[i:j], level+1))
		i = j
	}
	t, _ := hashtree.BuildTree(children)
	return t.Root()
}

// levelKeyEqual 报告两条叶记录在指定中间层是否同组：level 0 比年度，
// level 1/2/3 分别比 TxID 字节 [7]/[11]/[15]。更高层的同组性由调用方的
// 自顶向下划分保证，故此处只需比较当前层键。
func levelKeyEqual(a, b leafRecord, level int) bool {
	if level == 0 {
		return a.year == b.year
	}
	idx := groupByteIndices[level-1]
	return a.txid[idx] == b.txid[idx]
}
