package tx

import (
	"bytes"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 单签与多签 M-of-N 验证（第 08 章 §8-§9，DEC-0102/0103/0104）。
//
// 本层只对「已构造好的签名消息字节序列」执行验证：派生公钥哈希、比对接收者、
// 经 crypto.Verifier 逐一验签。不在此注入运行时 input_index、不执行脚本（脚本引擎
// 在第 10 章调用 FN_CHECKSIG/FN_MCHECKSIG 时提供消息与序位）。
//
// 三套排序严格区分（§9）：
//   - 地址派生：N 个 BaseH 字典序排序后串联、前置 m||n（在 crypto.AddressHashMultiFromBase 内）。
//   - 见证补全集：N-M 个未签公钥初级哈希按字典序升序携带，本层校验升序但不重排。
//   - 验签顺序：签名集/公钥集按见证提供的一一配对顺序逐一验签，不按哈希排序。

const baseHashSize = 32

// VerifySingle 验证单签输入（第 08 章 §8）。
//
// receiver 为输入来源的单签公钥哈希 SHA3-256(BLAKE2b-512(pubKey))；先由 pub 派生公钥哈希
// 并比对 receiver，再经 v 验证 sig 对 message 的合法性。比对失败返回 ErrReceiverMismatch，
// 验签失败返回 ErrSignatureInvalid，算法不一致等底层错误原样返回。
func VerifySingle(v crypto.Verifier, receiver types.AddressHash, pub crypto.PublicKey, sig crypto.Signature, message []byte) error {
	derived := crypto.AddressHashSingle(pub.Bytes())
	if !bytes.Equal(derived.Bytes(), receiver.Bytes()) {
		return ErrReceiverMismatch
	}
	ok, err := v.Verify(pub, message, sig)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSignatureInvalid
	}
	return nil
}

// MultisigWitness 是多签输入的见证三集合视图（第 08 章 §8，DEC-0103 §9）。
type MultisigWitness struct {
	// Signatures 是签名集（M 个签名），按见证提供顺序排列。
	Signatures []crypto.Signature
	// PublicKeys 是公钥集，与 Signatures 一一对应、同序，长度须等于 M。
	PublicKeys []crypto.PublicKey
	// Completion 是补全集：N-M 个未签名公钥初级哈希 BLAKE3-256(pubKey)，
	// 每项 32 字节，须按字典序升序排列（DEC-0103 §9），本层校验但不重排。
	Completion [][]byte
}

// VerifyMultisig 验证多签 M-of-N 输入（第 08 章 §8）。
//
// 由集合规模推出 m=len(Signatures)、n=m+len(Completion)；校验配比与集合一致性后，
// 将公钥集初级哈希与补全集合并派生复合公钥哈希（crypto.AddressHashMultiFromBase，
// 内部字典序排序、前置 m||n），与 receiver 比对，最后按见证顺序逐一验签。
//
// 错误：配比非法返回 ErrMultisigConfig；签名/公钥集规模不符或补全集项长度非法返回
// ErrMultisigSetMismatch；补全集非升序或含重复返回 ErrCompletionNotSorted；接收者
// 不匹配返回 ErrReceiverMismatch；任一签名验证失败返回 ErrSignatureInvalid。
func VerifyMultisig(v crypto.Verifier, receiver types.AddressHash, w MultisigWitness, message []byte) error {
	m := len(w.Signatures)
	if len(w.PublicKeys) != m {
		return ErrMultisigSetMismatch
	}
	n := m + len(w.Completion)
	if m == 0 || n == 0 || m > n || n > 255 {
		return ErrMultisigConfig
	}

	// 补全集：每项 32 字节，严格字典序升序（升序即保证无重复）。
	for i, bh := range w.Completion {
		if len(bh) != baseHashSize {
			return ErrMultisigSetMismatch
		}
		if i > 0 && bytes.Compare(w.Completion[i-1], bh) >= 0 {
			return ErrCompletionNotSorted
		}
	}

	// 合并 BaseH 集合：已签名公钥取初级哈希 + 补全集初级哈希。
	baseHashes := make([][]byte, 0, n)
	for _, pub := range w.PublicKeys {
		bh := crypto.PubKeyBaseHash(pub.Bytes())
		baseHashes = append(baseHashes, bh[:])
	}
	baseHashes = append(baseHashes, w.Completion...)

	derived, err := crypto.AddressHashMultiFromBase(uint8(m), uint8(n), baseHashes)
	if err != nil {
		// 配比/重复等派生期错误归一为多签配置错误。
		return ErrMultisigConfig
	}
	if !bytes.Equal(derived.Bytes(), receiver.Bytes()) {
		return ErrReceiverMismatch
	}

	// 逐一验签：按见证内顺序，签名与对应公钥配对。
	for i := range w.Signatures {
		ok, verr := v.Verify(w.PublicKeys[i], message, w.Signatures[i])
		if verr != nil {
			return verr
		}
		if !ok {
			return ErrSignatureInvalid
		}
	}
	return nil
}
