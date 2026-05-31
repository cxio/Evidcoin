package hashtree

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
)

func TestProofVerifyAllLeaves(t *testing.T) {
	for _, n := range []int{1, 2, 3, 4, 5, 8, 9} {
		tree, err := BuildTree(leaves(n))
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < n; i++ {
			p, err := tree.Proof(i)
			if err != nil {
				t.Fatalf("n=%d Proof(%d): %v", n, i, err)
			}
			if !Verify(p) {
				t.Fatalf("n=%d leaf %d proof failed to verify", n, i)
			}
		}
	}
}

func TestSingleLeafProofHasNoSiblingsAndVerifies(t *testing.T) {
	l := leaves(1)
	tree, err := BuildTree(l)
	if err != nil {
		t.Fatal(err)
	}
	p, err := tree.Proof(0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(p.LeafHash, l[0]) {
		t.Fatal("single leaf proof leaf hash must equal original leaf hash")
	}
	wantRoot := crypto.HashTreeBranch(l[0]).Bytes()
	if !bytes.Equal(p.Root, wantRoot) {
		t.Fatal("single leaf proof root must be normalized to branch hash")
	}
	if len(p.Siblings) != 0 {
		t.Fatalf("single leaf proof siblings = %d, want 0", len(p.Siblings))
	}
	if !Verify(p) {
		t.Fatal("single leaf proof must verify")
	}
}

func TestProofWrongDirectionFails(t *testing.T) {
	tree, _ := BuildTree(leaves(4))
	p, err := tree.Proof(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Siblings) == 0 {
		t.Fatal("expected sibling steps")
	}
	// 翻转第一个方向位；验证应失败。
	if p.Siblings[0].Direction == SiblingLeft {
		p.Siblings[0].Direction = SiblingRight
	} else {
		p.Siblings[0].Direction = SiblingLeft
	}
	if Verify(p) {
		t.Fatal("proof with wrong direction must not verify")
	}
}

func TestProofIndexOutOfRange(t *testing.T) {
	tree, _ := BuildTree(leaves(3))
	if _, err := tree.Proof(3); err != ErrLeafIndexRange {
		t.Fatalf("expected ErrLeafIndexRange, got %v", err)
	}
	if _, err := tree.Proof(-1); err != ErrLeafIndexRange {
		t.Fatalf("expected ErrLeafIndexRange, got %v", err)
	}
}

func TestProofTamperedSiblingFails(t *testing.T) {
	tree, _ := BuildTree(leaves(4))
	p, _ := tree.Proof(2)
	p.Siblings[0].Hash[0] ^= 0xFF
	if Verify(p) {
		t.Fatal("proof with tampered sibling must not verify")
	}
}
