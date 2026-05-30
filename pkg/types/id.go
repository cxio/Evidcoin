package types

// 协议标识符类型。底层虽与 Hash48/Hash32/Hash64 同宽，但为独立命名类型，
// 编译期禁止语义混用（如 BlockID 不能直接赋值给 TxID）。算法分配见第 02 章。

// BlockID 是区块头哈希（SHA3-384，48 字节）。
type BlockID Hash48

// TxID 是交易头哈希（SHA3-384，48 字节）。
type TxID Hash48

// CheckRoot 是每个区块的校验根哈希（SHA3-384，48 字节）。
type CheckRoot Hash48

// AddressHash 是 32 字节公钥哈希（SHA3-256(BLAKE2b-512(...))）。
type AddressHash Hash32

// TreeHash 是通用树分支哈希（BLAKE3-256，32 字节）。
type TreeHash Hash32

// AttachmentHash 是附件完整指纹（SHA3-512，64 字节）。
type AttachmentHash Hash64

// MintHash 是铸凭证明哈希（BLAKE3-256，32 字节，见 DEC-0301）。
type MintHash Hash32

// NewBlockID 从 b 构造 BlockID，b 长度必须恰好为 48 字节。
func NewBlockID(b []byte) (BlockID, error) {
	h, err := NewHash48(b)
	return BlockID(h), err
}

// NewTxID 从 b 构造 TxID，b 长度必须恰好为 48 字节。
func NewTxID(b []byte) (TxID, error) {
	h, err := NewHash48(b)
	return TxID(h), err
}

// NewCheckRoot 从 b 构造 CheckRoot，b 长度必须恰好为 48 字节。
func NewCheckRoot(b []byte) (CheckRoot, error) {
	h, err := NewHash48(b)
	return CheckRoot(h), err
}

// NewAddressHash 从 b 构造 AddressHash，b 长度必须恰好为 32 字节。
func NewAddressHash(b []byte) (AddressHash, error) {
	h, err := NewHash32(b)
	return AddressHash(h), err
}

// NewTreeHash 从 b 构造 TreeHash，b 长度必须恰好为 32 字节。
func NewTreeHash(b []byte) (TreeHash, error) {
	h, err := NewHash32(b)
	return TreeHash(h), err
}

// NewAttachmentHash 从 b 构造 AttachmentHash，b 长度必须恰好为 64 字节。
func NewAttachmentHash(b []byte) (AttachmentHash, error) {
	h, err := NewHash64(b)
	return AttachmentHash(h), err
}

// NewMintHash 从 b 构造 MintHash，b 长度必须恰好为 32 字节。
func NewMintHash(b []byte) (MintHash, error) {
	h, err := NewHash32(b)
	return MintHash(h), err
}

// MustBlockID 与 NewBlockID 类似，但出错时会 panic。
// 仅用于测试与长度已知合法的静态向量。
func MustBlockID(b []byte) BlockID {
	id, err := NewBlockID(b)
	if err != nil {
		panic(err)
	}
	return id
}

// MustTxID 与 NewTxID 类似，但出错时会 panic。
// 仅用于测试与静态向量。
func MustTxID(b []byte) TxID {
	id, err := NewTxID(b)
	if err != nil {
		panic(err)
	}
	return id
}

// Bytes 返回标识符字节的新副本。
func (id BlockID) Bytes() []byte { return Hash48(id).Bytes() }

// Bytes 返回标识符字节的新副本。
func (id TxID) Bytes() []byte { return Hash48(id).Bytes() }

// Bytes 返回标识符字节的新副本。
func (r CheckRoot) Bytes() []byte { return Hash48(r).Bytes() }

// Bytes 返回标识符字节的新副本。
func (a AddressHash) Bytes() []byte { return Hash32(a).Bytes() }

// Bytes 返回标识符字节的新副本。
func (t TreeHash) Bytes() []byte { return Hash32(t).Bytes() }

// Bytes 返回标识符字节的新副本。
func (a AttachmentHash) Bytes() []byte { return Hash64(a).Bytes() }

// Bytes 返回标识符字节的新副本。
func (m MintHash) Bytes() []byte { return Hash32(m).Bytes() }
