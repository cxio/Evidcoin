package tx

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 签名消息布局（DEC-0102 §1）。本文件实现普通交易「待签字节序列」的构造，
// 不执行脚本、不注入运行时 input_index（由脚本引擎在执行 FN_CHECKSIG/FN_MCHECKSIG
// 时提供，见第 06 章）。ML-DSA 验证输入即此处产出的签名消息字节序列。

// ChkType 是签名校验类别（SigScope 首字节，DEC-0102 §1）。
type ChkType uint8

const (
	// ChkCoinbase 标记 Coinbase 域（chk_type=0），不走授权种类覆盖路径（见 coinbase_sig.go）。
	ChkCoinbase ChkType = 0
	// ChkCoinSpend 标记币金花费（chk_type=1），与 FN_CHECKSIG/FN_MCHECKSIG 第一实参一致。
	ChkCoinSpend ChkType = 1
	// ChkCreditTransfer 标记凭信转移（chk_type=2）。
	ChkCreditTransfer ChkType = 2
)

// ChainScope 是签名消息的链标识前缀材料（DEC-0102 §1 / 第 03 章 MixData）。
// 它是 internal/blockchain 链身份的字节化视图；为保持 Layer 1 同层不互相依赖，
// 本层以基础类型承载，由上层从 blockchain.ChainIdentity 转换填充。
type ChainScope struct {
	// ProtocolID 区分本链与其它链（主网 "Evidcoin@v1"）。
	ProtocolID string
	// ChainID 是运行态标识（mainnet/testnet/devnet）。
	ChainID string
	// GenesisID 是创世区块哈希（48 字节，SHA3-384）。
	GenesisID types.BlockID
	// BoundID 是可选的主链绑定标识已编码字节（存在时为 Height(4B BE)||BlockPrefix(20B)）；
	// 空时仍以 varint(0) 占位编码，保证主链/分叉链签名消息结构一致（DEC-0102 §1）。
	BoundID []byte
}

// appendCanonical 追加 ChainScope 的规范编码（DEC-0102 §1）：
//
//	ProtoLen || ProtocolID || ChainLen || ChainID || GenesisID(48) || BoundLen || BoundID
//
// ProtocolID/ChainID 以 varint(len)||ASCII bytes 编码；GenesisID 定长 48 字节直接追加；
// BoundID 以 varint(len)||bytes 编码，空时为 varint(0)，不省略。
func (s ChainScope) appendCanonical(dst []byte) []byte {
	dst = types.AppendBytes(dst, []byte(s.ProtocolID))
	dst = types.AppendBytes(dst, []byte(s.ChainID))
	dst = append(dst, s.GenesisID.Bytes()...)
	dst = types.AppendBytes(dst, s.BoundID)
	return dst
}

// SignableOutput 是签名覆盖范围所需的单个输出语义视图（DEC-0102 §3 辅项）。
// 字段已按语义拆分为接收者、内容、锁定脚本三段，供 SCRIPT/CONTENT/RECEIVER/OUTPUT
// 辅项分别嵌入。每段在签名消息中以 varint(len)||bytes 编码。
type SignableOutput struct {
	// Receiver 是接收者字段原始字节（Coin/Credit 的 Receiver，Proof 的 Creator）。
	Receiver []byte
	// Content 是除锁定脚本与接收者外的字段拼接编码（CONTENT 段）。
	Content []byte
	// LockScript 是锁定脚本（SCRIPT 段）。
	LockScript []byte
}

// SignableFromCoin 从币金信元与其锁定脚本构造签名覆盖视图（DEC-0102 §3）。
// 内容段为 Amount(varint)||Memo(varint(len)||bytes)；接收者段为 Receiver。
func SignableFromCoin(c Coin, lockScript []byte) (SignableOutput, error) {
	if len(c.Receiver) > maxShortField {
		return SignableOutput{}, ErrReceiverTooLong
	}
	if len(c.Memo) > maxShortField {
		return SignableOutput{}, ErrMemoTooLong
	}
	content := types.AppendVarUint(nil, uint64(c.Amount))
	content = types.AppendBytes(content, c.Memo)
	return SignableOutput{Receiver: c.Receiver, Content: content, LockScript: lockScript}, nil
}

// SignableFromCredit 从凭信信元与其锁定脚本构造签名覆盖视图（DEC-0102 §3）。
// 内容段为 Creator||Title||Description||AttachmentID（各 varint(len)||bytes）；
// 接收者段为 Receiver。
func SignableFromCredit(c Credit, lockScript []byte) (SignableOutput, error) {
	if len(c.Receiver) > maxShortField {
		return SignableOutput{}, ErrReceiverTooLong
	}
	if len(c.Creator) > maxShortField {
		return SignableOutput{}, ErrCreatorTooLong
	}
	if len(c.Title) > maxShortField {
		return SignableOutput{}, ErrTitleTooLong
	}
	if len(c.Description) > maxDescription {
		return SignableOutput{}, ErrDescriptionTooLong
	}
	if len(c.AttachmentID) > maxShortField {
		return SignableOutput{}, ErrAttachmentIDTooLong
	}
	content := types.AppendBytes(nil, c.Creator)
	content = types.AppendBytes(content, c.Title)
	content = types.AppendBytes(content, c.Description)
	content = types.AppendBytes(content, c.AttachmentID)
	return SignableOutput{Receiver: c.Receiver, Content: content, LockScript: lockScript}, nil
}

// SignableFromProof 从存证信元与其锁定脚本构造签名覆盖视图（DEC-0102 §3）。
// 存证无接收者，Creator 视作接收者段；内容段为 Title||Content||AttachmentID。
func SignableFromProof(p Proof, lockScript []byte) (SignableOutput, error) {
	if len(p.Creator) > maxShortField {
		return SignableOutput{}, ErrCreatorTooLong
	}
	if len(p.Title) > maxShortField {
		return SignableOutput{}, ErrTitleTooLong
	}
	if len(p.Content) > maxDescription {
		return SignableOutput{}, ErrDescriptionTooLong
	}
	if len(p.AttachmentID) > maxShortField {
		return SignableOutput{}, ErrAttachmentIDTooLong
	}
	content := types.AppendBytes(nil, p.Title)
	content = types.AppendBytes(content, p.Content)
	content = types.AppendBytes(content, p.AttachmentID)
	return SignableOutput{Receiver: p.Creator, Content: content, LockScript: lockScript}, nil
}

// SigParams 聚合构造普通交易签名消息所需的全部材料（DEC-0102 §1）。
type SigParams struct {
	// Chain 是链标识前缀材料。
	Chain ChainScope
	// ChkType 是签名校验类别（币金花费/凭信转移）。
	ChkType ChkType
	// AuthFlag 是授权种类 8 位标记。
	AuthFlag AuthFlag
	// InputIndex 是当前被验证输入项序位（运行时注入）。
	InputIndex uint64
	// Version 是交易头版本号。
	Version uint16
	// Timestamp 是交易时间戳（Unix 毫秒）。
	Timestamp int64
	// MintPKHash 是可选铸凭公钥哈希，长度须为 0 或 32。
	MintPKHash []byte
	// Inputs 是交易输入集，用于 CoveredInputs。
	Inputs Inputs
	// Outputs 是交易输出的签名覆盖视图，用于 CoveredOutputs。
	Outputs []SignableOutput
}

// BuildSignatureMessage 构造普通交易签名消息字节序列（DEC-0102 §1）：
//
//	DomainTag("signature.message") || ChainScope || SigScope
//	  || TxHeaderCore || CoveredInputs || CoveredOutputs
//
// 当 auth_flag 非法（辅项互斥/主辅失配）、MintPKHash 长度非法、或
// SIGOUT_SELF/SIGIN_SELF 引用越界时返回相应错误。
func BuildSignatureMessage(p SigParams) ([]byte, error) {
	if err := p.AuthFlag.Validate(); err != nil {
		return nil, err
	}
	if len(p.MintPKHash) != 0 && len(p.MintPKHash) != 32 {
		return nil, ErrMintPKHashLength
	}

	dst := crypto.SignatureMessageTag()
	dst = p.Chain.appendCanonical(dst)

	// SigScope：chk_type(1B) || auth_flag(1B) || input_index(varint)。
	dst = append(dst, byte(p.ChkType), p.AuthFlag.Byte())
	dst = types.AppendVarUint(dst, p.InputIndex)

	// TxHeaderCore：Version(uint16 BE) || Timestamp(int64 BE) || MintPKHash(varint(len)||bytes)。
	dst = types.AppendUint16BE(dst, p.Version)
	dst = types.AppendInt64BE(dst, p.Timestamp)
	dst = types.AppendBytes(dst, p.MintPKHash)

	var err error
	dst, err = appendCoveredInputs(dst, p.AuthFlag, p.InputIndex, p.Inputs)
	if err != nil {
		return nil, err
	}
	dst, err = appendCoveredOutputs(dst, p.AuthFlag, p.InputIndex, p.Outputs)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

// appendCoveredInputs 按授权独项追加 CoveredInputs（DEC-0102 §3）。
// SIGIN_ALL 优先于 SIGIN_SELF；二者皆未设置时为空。
func appendCoveredInputs(dst []byte, flag AuthFlag, inputIndex uint64, in Inputs) ([]byte, error) {
	switch {
	case flag.Has(SigInAll):
		count := 1 + len(in.Rest)
		dst = types.AppendVarUint(dst, uint64(count))
		var err error
		dst, err = in.Lead.appendCanonical(dst)
		if err != nil {
			return nil, err
		}
		for i := range in.Rest {
			dst, err = in.Rest[i].appendCanonical(dst)
			if err != nil {
				return nil, err
			}
		}
		return dst, nil
	case flag.Has(SigInSelf):
		enc, err := in.inputAt(inputIndex)
		if err != nil {
			return nil, err
		}
		dst = types.AppendVarUint(dst, inputIndex)
		return append(dst, enc...), nil
	default:
		return dst, nil
	}
}

// inputAt 返回第 index 个输入项的规范编码（index 0 为首领输入，其余为 Rest）。
func (in Inputs) inputAt(index uint64) ([]byte, error) {
	if index == 0 {
		return in.Lead.appendCanonical(nil)
	}
	ri := index - 1
	if ri >= uint64(len(in.Rest)) {
		return nil, ErrInputIndexRange
	}
	return in.Rest[ri].appendCanonical(nil)
}

// appendCoveredOutputs 按授权主项/辅项追加 CoveredOutputs（DEC-0102 §3）。
// 无主项时为空；SIGOUT_ALL 优先于 SIGOUT_SELF。
func appendCoveredOutputs(dst []byte, flag AuthFlag, inputIndex uint64, outs []SignableOutput) ([]byte, error) {
	switch {
	case flag.Has(SigOutAll):
		dst = types.AppendVarUint(dst, uint64(len(outs)))
		for i := range outs {
			dst = types.AppendVarUint(dst, uint64(i))
			dst = appendOutputFields(dst, flag, outs[i])
		}
		return dst, nil
	case flag.Has(SigOutSelf):
		if inputIndex >= uint64(len(outs)) {
			return nil, ErrSigOutSelfRange
		}
		return appendOutputFields(dst, flag, outs[inputIndex]), nil
	default:
		return dst, nil
	}
}

// appendOutputFields 按辅项追加单个输出的内嵌字段段（DEC-0102 §3）。
// 字段段顺序固定为 SCRIPT, CONTENT, RECEIVER（对应 bit3>bit2>bit1）；OUTPUT 展开为三段并集。
// 每段以 varint(len)||bytes 编码。
func appendOutputFields(dst []byte, flag AuthFlag, o SignableOutput) []byte {
	if flag.Has(AuxOutput) {
		dst = types.AppendBytes(dst, o.LockScript)
		dst = types.AppendBytes(dst, o.Content)
		dst = types.AppendBytes(dst, o.Receiver)
		return dst
	}
	if flag.Has(AuxScript) {
		dst = types.AppendBytes(dst, o.LockScript)
	}
	if flag.Has(AuxContent) {
		dst = types.AppendBytes(dst, o.Content)
	}
	if flag.Has(AuxReceiver) {
		dst = types.AppendBytes(dst, o.Receiver)
	}
	return dst
}
