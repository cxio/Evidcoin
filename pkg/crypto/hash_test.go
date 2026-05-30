package crypto

import (
	"bytes"
	"testing"

	"golang.org/x/crypto/sha3"
	"lukechampine.com/blake3"
)

func TestDomainTagFormat(t *testing.T) {
	got := DomainTag("block.header")
	want := append([]byte("Evidcoin/v1/block.header"), 0x00)
	if !bytes.Equal(got, want) {
		t.Fatalf("DomainTag = %q, want %q", got, want)
	}
}

func TestDomainTagFullSet(t *testing.T) {
	tags := [][]byte{
		tagBlockHeader, tagTxHeader, tagTreeLeaf, tagTreeBranch, tagCheckRoot,
		tagUTXOLeaf, tagUTCOLeaf, tagMintHash, tagSignatureMsg, tagAttachment,
		tagAddressSingle, tagAddressMulti, tagUTXOEmpty, tagUTCOEmpty,
	}
	if len(tags) != 14 {
		t.Fatalf("expected 14 domain tags, got %d", len(tags))
	}
	seen := make(map[string]bool)
	for _, tag := range tags {
		if !bytes.HasPrefix(tag, []byte(domainPrefix)) {
			t.Errorf("tag %q missing prefix", tag)
		}
		if tag[len(tag)-1] != 0x00 {
			t.Errorf("tag %q missing 0x00 terminator", tag)
		}
		if seen[string(tag)] {
			t.Errorf("duplicate tag %q", tag)
		}
		seen[string(tag)] = true
	}
}

func TestHashOutputLengths(t *testing.T) {
	data := []byte("payload")
	if got := HashBlockHeader(data).Bytes(); len(got) != 48 {
		t.Errorf("block header len = %d, want 48", len(got))
	}
	if got := HashAttachment(data).Bytes(); len(got) != 64 {
		t.Errorf("attachment len = %d, want 64", len(got))
	}
	if got := HashMint(data).Bytes(); len(got) != 32 {
		t.Errorf("mint len = %d, want 32", len(got))
	}
	if got := HashTreeBranch(data).Bytes(); len(got) != 32 {
		t.Errorf("tree branch len = %d, want 32", len(got))
	}
}

func TestHashDomainSeparation(t *testing.T) {
	data := []byte("same-payload")
	// 相同 payload 在不同 SHA3-384 域下的结果必须不同。
	if bytes.Equal(HashBlockHeader(data).Bytes(), HashTxHeader(data).Bytes()) {
		t.Error("block.header and tx.header collide on same payload")
	}
	if bytes.Equal(HashUTXOLeaf(data).Bytes(), HashUTCOLeaf(data).Bytes()) {
		t.Error("utxo.leaf and utco.leaf collide on same payload")
	}
}

func TestHashBlockHeaderVector(t *testing.T) {
	data := []byte("header")
	want := sha3.Sum384(append(append([]byte{}, tagBlockHeader...), data...))
	if !bytes.Equal(HashBlockHeader(data).Bytes(), want[:]) {
		t.Error("HashBlockHeader does not match SHA3-384(tag||data)")
	}
}

func TestEmptyRoots(t *testing.T) {
	wantUTXO := sha3.Sum384(tagUTXOEmpty)
	if !bytes.Equal(EmptyUTXORoot().Bytes(), wantUTXO[:]) {
		t.Error("EmptyUTXORoot mismatch")
	}
	wantUTCO := sha3.Sum384(tagUTCOEmpty)
	if !bytes.Equal(EmptyUTCORoot().Bytes(), wantUTCO[:]) {
		t.Error("EmptyUTCORoot mismatch")
	}
	if bytes.Equal(EmptyUTXORoot().Bytes(), EmptyUTCORoot().Bytes()) {
		t.Error("UTXO and UTCO empty roots must differ")
	}
}

func TestAttachmentPieceTreeNoDomainTag(t *testing.T) {
	piece := []byte("file-piece-data")
	pieceHash := blake3.Sum256(piece)
	var pre [34]byte
	pre[0] = 0x00
	pre[1] = 0x05
	copy(pre[2:], pieceHash[:])
	want := blake3.Sum256(pre[:])

	got := HashAttachmentPieceLeaf(5, piece)
	if !bytes.Equal(got.Bytes(), want[:]) {
		t.Fatalf("piece leaf = %x, want %x (must be domain-tag-free)", got.Bytes(), want)
	}
}
