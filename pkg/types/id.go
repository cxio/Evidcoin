package types

// 协议标识符类型。底层虽与 Hash48/Hash32/Hash64 同宽，但为独立命名类型，
// 编译期禁止语义混用（如 BlockID 不能直接赋值给 TxID）。算法分配见第 02 章。

// BlockID is a block header hash (SHA3-384, 48 bytes).
type BlockID Hash48

// TxID is a transaction header hash (SHA3-384, 48 bytes).
type TxID Hash48

// CheckRoot is the per-block check root hash (SHA3-384, 48 bytes).
type CheckRoot Hash48

// AddressHash is a 32-byte public key hash (SHA3-256(BLAKE2b-512(...))).
type AddressHash Hash32

// TreeHash is a generic tree branch hash (BLAKE3-256, 32 bytes).
type TreeHash Hash32

// AttachmentHash is an attachment full fingerprint (SHA3-512, 64 bytes).
type AttachmentHash Hash64

// MintHash is the mint proof hash (BLAKE3-256, 32 bytes per DEC-0301).
type MintHash Hash32

// NewBlockID constructs a BlockID from b, which must be exactly 48 bytes.
func NewBlockID(b []byte) (BlockID, error) {
	h, err := NewHash48(b)
	return BlockID(h), err
}

// NewTxID constructs a TxID from b, which must be exactly 48 bytes.
func NewTxID(b []byte) (TxID, error) {
	h, err := NewHash48(b)
	return TxID(h), err
}

// NewCheckRoot constructs a CheckRoot from b, which must be exactly 48 bytes.
func NewCheckRoot(b []byte) (CheckRoot, error) {
	h, err := NewHash48(b)
	return CheckRoot(h), err
}

// NewAddressHash constructs an AddressHash from b, which must be exactly 32 bytes.
func NewAddressHash(b []byte) (AddressHash, error) {
	h, err := NewHash32(b)
	return AddressHash(h), err
}

// NewTreeHash constructs a TreeHash from b, which must be exactly 32 bytes.
func NewTreeHash(b []byte) (TreeHash, error) {
	h, err := NewHash32(b)
	return TreeHash(h), err
}

// NewAttachmentHash constructs an AttachmentHash from b, which must be exactly 64 bytes.
func NewAttachmentHash(b []byte) (AttachmentHash, error) {
	h, err := NewHash64(b)
	return AttachmentHash(h), err
}

// NewMintHash constructs a MintHash from b, which must be exactly 32 bytes.
func NewMintHash(b []byte) (MintHash, error) {
	h, err := NewHash32(b)
	return MintHash(h), err
}

// MustBlockID is like NewBlockID but panics on error. Use only in tests and
// static vectors where the input length is known to be valid.
func MustBlockID(b []byte) BlockID {
	id, err := NewBlockID(b)
	if err != nil {
		panic(err)
	}
	return id
}

// MustTxID is like NewTxID but panics on error. Use only in tests and static vectors.
func MustTxID(b []byte) TxID {
	id, err := NewTxID(b)
	if err != nil {
		panic(err)
	}
	return id
}

// Bytes returns a fresh copy of the identifier bytes.
func (id BlockID) Bytes() []byte { return Hash48(id).Bytes() }

// Bytes returns a fresh copy of the identifier bytes.
func (id TxID) Bytes() []byte { return Hash48(id).Bytes() }

// Bytes returns a fresh copy of the identifier bytes.
func (r CheckRoot) Bytes() []byte { return Hash48(r).Bytes() }

// Bytes returns a fresh copy of the identifier bytes.
func (a AddressHash) Bytes() []byte { return Hash32(a).Bytes() }

// Bytes returns a fresh copy of the identifier bytes.
func (t TreeHash) Bytes() []byte { return Hash32(t).Bytes() }

// Bytes returns a fresh copy of the identifier bytes.
func (a AttachmentHash) Bytes() []byte { return Hash64(a).Bytes() }

// Bytes returns a fresh copy of the identifier bytes.
func (m MintHash) Bytes() []byte { return Hash32(m).Bytes() }
