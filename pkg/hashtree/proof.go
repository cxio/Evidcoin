package hashtree

import "bytes"

// 通用验证路径编码（DEC-0004）。验证路径携带方向位，但不单独携带 leafIndex；
// 序号（若有）已包含在 leaf payload 中，由 leaf 哈希反演即可。

// Direction 表示验证时兄弟哈希相对当前运行节点哈希位于哪一侧。
type Direction uint8

const (
	// SiblingLeft 表示兄弟哈希位于左侧子节点（组合 sibling||cur）。
	SiblingLeft Direction = 0
	// SiblingRight 表示兄弟哈希位于右侧子节点（组合 cur||sibling）。
	SiblingRight Direction = 1
)

// ProofStep 表示验证路径上的一个兄弟节点。
type ProofStep struct {
	Direction Direction
	Hash      []byte
}

// Proof 是通用验证路径。它携带叶哈希、带方向位的兄弟链以及目标根哈希，
// 但不包含 leafIndex 字段。
type Proof struct {
	LeafHash []byte
	Siblings []ProofStep
	Root     []byte
}

// Proof 为索引 idx 的叶子构建验证路径。
func (t *Tree) Proof(idx int) (Proof, error) {
	if idx < 0 || idx >= len(t.levels[0]) {
		return Proof{}, ErrLeafIndexRange
	}
	p := Proof{
		LeafHash: cloneBytes(t.levels[0][idx]),
		Root:     t.Root(),
	}
	cur := idx
	for l := 0; l < len(t.levels)-1; l++ {
		level := t.levels[l]
		// 提升节点（奇数层最后一个）无兄弟，直接上移。
		if cur == len(level)-1 && len(level)%2 == 1 {
			cur /= 2
			continue
		}
		if cur%2 == 0 {
			p.Siblings = append(p.Siblings, ProofStep{
				Direction: SiblingRight,
				Hash:      cloneBytes(level[cur+1]),
			})
		} else {
			p.Siblings = append(p.Siblings, ProofStep{
				Direction: SiblingLeft,
				Hash:      cloneBytes(level[cur-1]),
			})
		}
		cur /= 2
	}
	return p, nil
}

// Verify 基于叶哈希与兄弟链重算根哈希，并与证明中的根进行比较。
func Verify(p Proof) bool {
	cur := cloneBytes(p.LeafHash)
	for _, s := range p.Siblings {
		switch s.Direction {
		case SiblingLeft:
			cur = branchHash(s.Hash, cur)
		case SiblingRight:
			cur = branchHash(cur, s.Hash)
		default:
			return false
		}
	}
	return bytes.Equal(cur, p.Root)
}

func cloneBytes(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
