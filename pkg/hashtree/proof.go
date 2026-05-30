package hashtree

import "bytes"

// 通用验证路径编码（DEC-0004）。验证路径携带方向位，但不单独携带 leafIndex；
// 序号（若有）已包含在 leaf payload 中，由 leaf 哈希反演即可。

// Direction indicates which side a sibling hash sits on relative to the running
// node hash during verification.
type Direction uint8

const (
	// SiblingLeft means the sibling hash is the left child (combine sibling||cur).
	SiblingLeft Direction = 0
	// SiblingRight means the sibling hash is the right child (combine cur||sibling).
	SiblingRight Direction = 1
)

// ProofStep is one sibling along the verification path.
type ProofStep struct {
	Direction Direction
	Hash      []byte
}

// Proof is a generic verification path. It carries the leaf hash, the sibling
// chain with direction bits, and the expected root, but never a leafIndex field.
type Proof struct {
	LeafHash []byte
	Siblings []ProofStep
	Root     []byte
}

// Proof builds the verification path for the leaf at index idx.
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

// Verify recomputes the root from the leaf hash and sibling chain, comparing it
// against the proof's root.
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
