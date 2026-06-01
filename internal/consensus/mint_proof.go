package consensus

import "github.com/cxio/evidcoin/pkg/types"

// MintProof 是择优凭证（第 11 章 §4，DEC-0301 字段顺序冻结）。
// 最终铸造者的 MintProof 存储于区块 Coinbase 交易中并参与 TxID 计算。
// 凭证中的签名是铸造资格证明，不是输入项花费签名，不属于见证剪枝范畴（第 08 章）。
type MintProof struct {
	// TxHeight 是铸凭交易实际所在区块高度。
	TxHeight uint32
	// TxID 是铸凭交易完整 48 字节 TxID。
	TxID types.TxID
	// MintPubKey 是铸造者公钥（验证身份与签名）。
	MintPubKey []byte
	// MintHash 是铸凭哈希值本身（可由前像推导，携带便于检索/预筛选）。
	// 注意：MintHash 置于签名前仅便于检索；签名验证仍以重新计算的铸凭哈希为准。
	MintHash types.MintHash
	// Signature 是铸造者对 MintHash 的签名。
	Signature []byte
}

// CanonicalBytes 按 DEC-0301 冻结的五字段顺序编码 MintProof：
//
//	TxHeight(u32 BE) || TxID[48] || varint(len)||MintPubKey || MintHash[32] || varint(len)||Signature
func (p MintProof) CanonicalBytes() []byte {
	out := make([]byte, 0, 4+48+1+len(p.MintPubKey)+32+1+len(p.Signature))
	out = types.AppendUint32BE(out, p.TxHeight)
	out = append(out, p.TxID.Bytes()...)
	out = types.AppendBytes(out, p.MintPubKey)
	out = append(out, p.MintHash.Bytes()...)
	out = types.AppendBytes(out, p.Signature)
	return out
}

// parseMintProof 从 src 解析一个 MintProof，返回消费的字节数。
// 该函数不检查尾随字节，便于在更大字节流（如 Coinbase Minter 字段）中复用。
func parseMintProof(src []byte) (MintProof, int, error) {
	var p MintProof
	off := 0

	height, n, err := types.ReadUint32BE(src[off:])
	if err != nil {
		return p, 0, ErrMintProofTooShort
	}
	off += n
	p.TxHeight = height

	if len(src[off:]) < 48 {
		return p, 0, ErrMintProofTooShort
	}
	txID, err := types.NewTxID(src[off : off+48])
	if err != nil {
		return p, 0, ErrMintProofTooShort
	}
	off += 48
	p.TxID = txID

	pubKey, n, err := types.ReadBytes(src[off:])
	if err != nil {
		return p, 0, ErrMintProofTooShort
	}
	off += n
	p.MintPubKey = pubKey

	if len(src[off:]) < 32 {
		return p, 0, ErrMintProofTooShort
	}
	mintHash, err := types.NewMintHash(src[off : off+32])
	if err != nil {
		return p, 0, ErrMintProofTooShort
	}
	off += 32
	p.MintHash = mintHash

	sig, n, err := types.ReadBytes(src[off:])
	if err != nil {
		return p, 0, ErrMintProofTooShort
	}
	off += n
	p.Signature = sig

	return p, off, nil
}

// ReadMintProof 严格解析单个 MintProof，要求完整消费 src。
// 存在多余尾随字节时返回 ErrMintProofTrailing。
func ReadMintProof(src []byte) (MintProof, int, error) {
	p, n, err := parseMintProof(src)
	if err != nil {
		return MintProof{}, 0, err
	}
	if n != len(src) {
		return MintProof{}, 0, ErrMintProofTrailing
	}
	return p, n, nil
}
