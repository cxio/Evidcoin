// Package crypto freezes the protocol-wide hash profiles, domain-isolation
// tags, public-key/address encoding and post-quantum signature (ML-DSA-65)
// abstraction. It depends only on pkg/types and third-party crypto libraries.
package crypto

import (
	"hash"

	"github.com/cxio/evidcoin/pkg/types"
	"golang.org/x/crypto/sha3"
	"lukechampine.com/blake3"
)

// domainPrefix is the fixed namespace prefix for all domain tags (DEC-0002).
const domainPrefix = "Evidcoin/v1/"

// domain tag names (14-item full set: DEC-0002 12 items + DEC-0201 two empty roots).
const (
	tagNameBlockHeader   = "block.header"
	tagNameTxHeader      = "tx.header"
	tagNameTreeLeaf      = "tree.leaf"
	tagNameTreeBranch    = "tree.branch"
	tagNameCheckRoot     = "checkroot"
	tagNameUTXOLeaf      = "utxo.leaf"
	tagNameUTCOLeaf      = "utco.leaf"
	tagNameMintHash      = "mint.hash"
	tagNameSignatureMsg  = "signature.message"
	tagNameAttachment    = "attachment.fingerprint"
	tagNameAddressSingle = "address.single"
	tagNameAddressMulti  = "address.multi"
	tagNameUTXOEmpty     = "utxo.empty"
	tagNameUTCOEmpty     = "utco.empty"
)

// Precomputed domain tags (`"Evidcoin/v1/" || name || 0x00`). These are the
// only authoritative protocol tags; callers must not pass arbitrary tags into
// the hash API, isolation is bound to the use-specific functions below.
var (
	tagBlockHeader   = DomainTag(tagNameBlockHeader)
	tagTxHeader      = DomainTag(tagNameTxHeader)
	tagTreeLeaf      = DomainTag(tagNameTreeLeaf)
	tagTreeBranch    = DomainTag(tagNameTreeBranch)
	tagCheckRoot     = DomainTag(tagNameCheckRoot)
	tagUTXOLeaf      = DomainTag(tagNameUTXOLeaf)
	tagUTCOLeaf      = DomainTag(tagNameUTCOLeaf)
	tagMintHash      = DomainTag(tagNameMintHash)
	tagSignatureMsg  = DomainTag(tagNameSignatureMsg)
	tagAttachment    = DomainTag(tagNameAttachment)
	tagAddressSingle = DomainTag(tagNameAddressSingle)
	tagAddressMulti  = DomainTag(tagNameAddressMulti)
	tagUTXOEmpty     = DomainTag(tagNameUTXOEmpty)
	tagUTCOEmpty     = DomainTag(tagNameUTCOEmpty)
)

// DomainTag builds a domain tag from a use name: "Evidcoin/v1/" || name || 0x00
// (DEC-0002). The tag must be the first segment of a hash preimage.
func DomainTag(name string) []byte {
	tag := make([]byte, 0, len(domainPrefix)+len(name)+1)
	tag = append(tag, domainPrefix...)
	tag = append(tag, name...)
	tag = append(tag, 0x00)
	return tag
}

// sum writes each part into h in order and returns the digest.
func sum(h hash.Hash, parts ...[]byte) []byte {
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

func sha3_384(parts ...[]byte) []byte { return sum(sha3.New384(), parts...) }
func sha3_512(parts ...[]byte) []byte { return sum(sha3.New512(), parts...) }

// blake3_256 returns the 32-byte BLAKE3 digest of the concatenated parts. BLAKE3
// is never used in keyed mode (DEC-0002); isolation relies solely on the tag.
func blake3_256(parts ...[]byte) [32]byte {
	h := blake3.New(32, nil)
	for _, p := range parts {
		h.Write(p)
	}
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// HashBlockHeader hashes a block header preimage (SHA3-384 + block.header).
func HashBlockHeader(data []byte) types.BlockID {
	id, _ := types.NewBlockID(sha3_384(tagBlockHeader, data))
	return id
}

// HashTxHeader hashes a transaction header preimage (SHA3-384 + tx.header).
func HashTxHeader(data []byte) types.TxID {
	id, _ := types.NewTxID(sha3_384(tagTxHeader, data))
	return id
}

// HashCheckRoot hashes a check-root preimage (SHA3-384 + checkroot).
func HashCheckRoot(data []byte) types.CheckRoot {
	r, _ := types.NewCheckRoot(sha3_384(tagCheckRoot, data))
	return r
}

// HashTreeLeaf hashes a generic tree leaf payload (SHA3-384 + tree.leaf).
func HashTreeLeaf(data []byte) types.Hash48 {
	h, _ := types.NewHash48(sha3_384(tagTreeLeaf, data))
	return h
}

// HashTreeBranch hashes a generic tree branch preimage (BLAKE3-256 + tree.branch).
// data is the concatenation left || right.
func HashTreeBranch(data []byte) types.TreeHash {
	return types.TreeHash(blake3_256(tagTreeBranch, data))
}

// HashUTXOLeaf hashes a UTXO end leaf payload (SHA3-384 + utxo.leaf).
func HashUTXOLeaf(data []byte) types.Hash48 {
	h, _ := types.NewHash48(sha3_384(tagUTXOLeaf, data))
	return h
}

// HashUTCOLeaf hashes a UTCO end leaf payload (SHA3-384 + utco.leaf).
func HashUTCOLeaf(data []byte) types.Hash48 {
	h, _ := types.NewHash48(sha3_384(tagUTCOLeaf, data))
	return h
}

// HashAttachment hashes an attachment full fingerprint (SHA3-512 + attachment.fingerprint).
func HashAttachment(data []byte) types.AttachmentHash {
	h, _ := types.NewAttachmentHash(sha3_512(tagAttachment, data))
	return h
}

// HashMint hashes a mint proof preimage (BLAKE3-256 + mint.hash, 32 bytes).
func HashMint(data []byte) types.MintHash {
	return types.MintHash(blake3_256(tagMintHash, data))
}

// SignatureMessageTag returns the signature.message domain tag bytes for use by
// the signature message profile (DEC-0102, 第 08 章).
func SignatureMessageTag() []byte {
	out := make([]byte, len(tagSignatureMsg))
	copy(out, tagSignatureMsg)
	return out
}

// EmptyUTXORoot returns the UTXO empty-state tree root: SHA3-384(DomainTag("utxo.empty")).
func EmptyUTXORoot() types.Hash48 {
	h, _ := types.NewHash48(sha3_384(tagUTXOEmpty))
	return h
}

// EmptyUTCORoot returns the UTCO empty-state tree root: SHA3-384(DomainTag("utco.empty")).
func EmptyUTCORoot() types.Hash48 {
	h, _ := types.NewHash48(sha3_384(tagUTCOEmpty))
	return h
}

// HashAttachmentPieceLeaf hashes an attachment piece tree leaf. This is the sole
// domain-tag-free exception (DEC-0002): BLAKE3-256(2-byte seq || BLAKE3-256(piece)),
// a 34-byte preimage with NO domain tag, so external file-sharing tools can reuse it.
func HashAttachmentPieceLeaf(seq uint16, piece []byte) types.TreeHash {
	pieceHash := blake3_256(piece)
	var seqBytes [2]byte
	seqBytes[0] = byte(seq >> 8)
	seqBytes[1] = byte(seq)
	return types.TreeHash(blake3_256(seqBytes[:], pieceHash[:]))
}

// HashAttachmentPieceBranch hashes an attachment piece tree branch:
// BLAKE3-256(left || right), with NO domain tag (DEC-0002 exception).
func HashAttachmentPieceBranch(left, right types.TreeHash) types.TreeHash {
	l := types.Hash32(left)
	r := types.Hash32(right)
	return types.TreeHash(blake3_256(l[:], r[:]))
}
