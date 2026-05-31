// Package hashtree 实现协议范围内的通用二叉哈希树与验证路径编码（DEC-0004），
// 并提供有序叶子辅助函数，供专用树（区块交易树、输出树、UTXO/UTCO 中间层）复用；
// 交易输入根按专用规则计算，不使用本通用树。
// 节点哈希使用 []byte 承载，因为通用树会混合 48 字节 SHA3-384 叶子层与 32 字节 BLAKE3-256 分支层；
// 单叶根按 tree.branch profile 归一化为 32 字节。
package hashtree

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/crypto"
)

// ErrEmptyTree 表示尝试用 0 个叶子构建树。
// 空根由具体结构单独定义（DEC-0004），不由通用树统一给出。
var ErrEmptyTree = errors.New("hashtree: cannot build tree from zero leaves")

// ErrLeafIndexRange 表示证明路径请求的叶子索引越界。
var ErrLeafIndexRange = errors.New("hashtree: leaf index out of range")

// Tree 是不可变的通用二叉哈希树。levels[0] 保存叶子哈希；
// 更高层由相邻节点两两做分支哈希得到；若某层末尾为奇数单节点，
// 则直接提升（不复制自身）。
type Tree struct {
	levels [][][]byte
}

// LeafHash 对已组装的 payload 计算通用树叶哈希
// （SHA3-384 + tree.leaf 域标签）。若包含序号前缀，应已位于 payload
// 开头（见 OrderedLeaf2 / OrderedLeaf3）。
func LeafHash(payload []byte) []byte {
	return crypto.HashTreeLeaf(payload).Bytes()
}

// branchHash 计算分支节点哈希：BLAKE3-256(tree.branch || left || right)。
func branchHash(left, right []byte) []byte {
	pre := make([]byte, 0, len(left)+len(right))
	pre = append(pre, left...)
	pre = append(pre, right...)
	h := crypto.HashTreeBranch(pre)
	return h.Bytes()
}

func singleRootHash(leaf []byte) []byte {
	return crypto.HashTreeBranch(leaf).Bytes()
}

// BuildTree 基于预计算叶哈希构建通用二叉哈希树。
// leafHashes 为空时返回 ErrEmptyTree。只有一个叶子时，树根归一化为分支哈希。
func BuildTree(leafHashes [][]byte) (*Tree, error) {
	if len(leafHashes) == 0 {
		return nil, ErrEmptyTree
	}
	level := make([][]byte, len(leafHashes))
	for i, h := range leafHashes {
		cp := make([]byte, len(h))
		copy(cp, h)
		level[i] = cp
	}
	t := &Tree{levels: [][][]byte{level}}
	if len(level) == 1 {
		t.levels = append(t.levels, [][]byte{singleRootHash(level[0])})
		return t, nil
	}
	for len(level) > 1 {
		next := make([][]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			if i+1 == len(level) {
				// 奇数层最后一个节点直接提升，不复制自身（DEC-0004）。
				next = append(next, level[i])
				continue
			}
			next = append(next, branchHash(level[i], level[i+1]))
		}
		t.levels = append(t.levels, next)
		level = next
	}
	return t, nil
}

// BuildFromPayloads 是便捷函数：先对每个 payload 计算叶哈希，再构建树。
func BuildFromPayloads(payloads [][]byte) (*Tree, error) {
	leaves := make([][]byte, len(payloads))
	for i, p := range payloads {
		leaves[i] = LeafHash(p)
	}
	return BuildTree(leaves)
}

// Root 返回树根哈希的副本。
func (t *Tree) Root() []byte {
	top := t.levels[len(t.levels)-1]
	out := make([]byte, len(top[0]))
	copy(out, top[0])
	return out
}

// LeafCount 返回树中的叶子数量。
func (t *Tree) LeafCount() int {
	return len(t.levels[0])
}
