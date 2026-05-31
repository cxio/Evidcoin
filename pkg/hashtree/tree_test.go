package hashtree

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
)

func leaves(n int) [][]byte {
	out := make([][]byte, n)
	for i := 0; i < n; i++ {
		out[i] = LeafHash(OrderedLeaf3(uint32(i), []byte{byte(i)}))
	}
	return out
}

func TestSingleLeafRootNormalizedToBranchHash(t *testing.T) {
	l := leaves(1)
	tree, err := BuildTree(l)
	if err != nil {
		t.Fatal(err)
	}
	want := crypto.HashTreeBranch(l[0]).Bytes()
	if !bytes.Equal(tree.Root(), want) {
		t.Fatal("single leaf root must be normalized to branch hash")
	}
	if bytes.Equal(tree.Root(), l[0]) {
		t.Fatal("single leaf root must not equal the leaf hash")
	}
	if len(tree.Root()) != 32 {
		t.Fatalf("single leaf root len = %d, want 32", len(tree.Root()))
	}
}

func TestBranchOutputs32Bytes(t *testing.T) {
	tree, err := BuildTree(leaves(2))
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Root()) != 32 {
		t.Fatalf("branch root len = %d, want 32", len(tree.Root()))
	}
}

func TestLeftRightSwapChangesRoot(t *testing.T) {
	a := LeafHash([]byte("a"))
	b := LeafHash([]byte("b"))
	t1, _ := BuildTree([][]byte{a, b})
	t2, _ := BuildTree([][]byte{b, a})
	if bytes.Equal(t1.Root(), t2.Root()) {
		t.Fatal("swapping left and right must change the root")
	}
}

func TestOrderedLeafSequencePrefixMatters(t *testing.T) {
	body := []byte("same-body")
	if bytes.Equal(LeafHash(OrderedLeaf3(0, body)), LeafHash(OrderedLeaf3(1, body))) {
		t.Fatal("different 3-byte seq prefix must produce different leaf hash")
	}
	if bytes.Equal(LeafHash(OrderedLeaf2(0, body)), LeafHash(OrderedLeaf2(1, body))) {
		t.Fatal("different 2-byte seq prefix must produce different leaf hash")
	}
}

func TestOddLevelPromotionNotDuplicated(t *testing.T) {
	// 3 个叶子：level0=[a,b,c]；期望根为 branch(branch(a,b), c)，
	// 即 c 直接提升，而不是复制为 branch(c,c)。
	l := leaves(3)
	tree, _ := BuildTree(l)

	ab := branchHash(l[0], l[1])
	wantPromote := branchHash(ab, l[2])
	wantDuplicate := branchHash(ab, branchHash(l[2], l[2]))

	if !bytes.Equal(tree.Root(), wantPromote) {
		t.Fatal("odd node must be promoted directly")
	}
	if bytes.Equal(tree.Root(), wantDuplicate) {
		t.Fatal("odd node must NOT be duplicated")
	}
}

func TestEmptyTreeRejected(t *testing.T) {
	if _, err := BuildTree(nil); err != ErrEmptyTree {
		t.Fatalf("expected ErrEmptyTree, got %v", err)
	}
}
