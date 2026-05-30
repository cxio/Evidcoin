// Package hashtree implements the protocol-wide generic binary hash tree and
// verification path encoding (DEC-0004), plus the ordered-leaf helpers used by
// the specialised trees (block transaction tree, input/output trees, UTXO/UTCO
// middle layers). Node hashes are carried as []byte because a generic tree mixes
// a 48-byte SHA3-384 leaf layer with 32-byte BLAKE3-256 branch layers, and a
// single-leaf root equals the 48-byte leaf hash itself.
package hashtree

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/crypto"
)

// ErrEmptyTree is returned when building a tree from zero leaves. Empty roots
// are defined per-structure (DEC-0004), not by the generic tree.
var ErrEmptyTree = errors.New("hashtree: cannot build tree from zero leaves")

// ErrLeafIndexRange is returned for an out-of-range proof leaf index.
var ErrLeafIndexRange = errors.New("hashtree: leaf index out of range")

// Tree is an immutable generic binary hash tree. levels[0] holds the leaf
// hashes; each higher level is derived by branch hashing adjacent pairs, with
// an odd trailing node promoted directly (not duplicated).
type Tree struct {
	levels [][][]byte
}

// LeafHash computes a generic tree leaf hash for an already-assembled payload
// (SHA3-384 + tree.leaf domain tag). Any sequence prefix must already be the
// front of payload (see OrderedLeaf2 / OrderedLeaf3).
func LeafHash(payload []byte) []byte {
	return crypto.HashTreeLeaf(payload).Bytes()
}

// branchHash computes a branch node hash: BLAKE3-256(tree.branch || left || right).
func branchHash(left, right []byte) []byte {
	pre := make([]byte, 0, len(left)+len(right))
	pre = append(pre, left...)
	pre = append(pre, right...)
	h := crypto.HashTreeBranch(pre)
	return h.Bytes()
}

// BuildTree builds a generic binary hash tree from precomputed leaf hashes.
// It returns ErrEmptyTree when leafHashes is empty. A single leaf yields a tree
// whose root equals that leaf hash (no extra branch layer).
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

// BuildFromPayloads is a convenience that leaf-hashes each payload then builds
// the tree.
func BuildFromPayloads(payloads [][]byte) (*Tree, error) {
	leaves := make([][]byte, len(payloads))
	for i, p := range payloads {
		leaves[i] = LeafHash(p)
	}
	return BuildTree(leaves)
}

// Root returns a copy of the tree root hash.
func (t *Tree) Root() []byte {
	top := t.levels[len(t.levels)-1]
	out := make([]byte, len(top[0]))
	copy(out, top[0])
	return out
}

// LeafCount returns the number of leaves in the tree.
func (t *Tree) LeafCount() int {
	return len(t.levels[0])
}
