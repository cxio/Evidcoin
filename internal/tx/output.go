package tx

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 输出项配置字节（Config byte）高 4 位摘要标记（第 06 章 §6，DEC-0101）。
// 摘要标记只影响输出项叶子哈希前像的选择：被标记的片段先对实际字段字节计算摘要，
// 再以该摘要参与输出项哈希；不改变 payload 编码、输出类型、状态归属或签名授权。
const (
	// DigestAccount 指示接收者使用哈希摘要参与计算输出项哈希（bit7，通常无用）。
	DigestAccount uint8 = 0x80
	// DigestContent 指示内容部分使用哈希摘要参与计算输出项哈希（bit6，大负载时有用）。
	DigestContent uint8 = 0x40
	// DigestScript 指示脚本部分使用哈希摘要参与计算输出项哈希（bit5，长脚本时有用）。
	DigestScript uint8 = 0x20
	// digestMask 是有效摘要标记的掩码（bit7|bit6|bit5）。
	digestMask uint8 = DigestAccount | DigestContent | DigestScript
)

// OutputType 是标准输出的低 4 位类型值（第 06 章 §6）。
type OutputType uint8

const (
	// TypeReserved 是预留类型值 0，无意义，结构验证拒绝。
	TypeReserved OutputType = 0
	// TypeCoin 是币金输出（可作输入源，入 UTXO）。
	TypeCoin OutputType = 1
	// TypeCredit 是凭信输出（可作输入源，入 UTCO）。
	TypeCredit OutputType = 2
	// TypeProof 是存证输出（不可作输入源，不入集）。
	TypeProof OutputType = 3
)

// Output 是交易输出 envelope（第 06 章 §6，DEC-0101）：公共头 Config + 信元载荷 Payload
// + 锁定脚本 LockScript。Config 字节高 4 位为摘要标记，低 4 位为类型值；无销毁位，
// 普通交易不可销毁币金（销毁仅由 Coinbase BurnCoin 表达）。
type Output struct {
	// Serial 是输出序位（从 0 开始，第 04 章 §3.4 / 附.交易.md）。它由输出在交易
	// 输出集中的位置决定，用于输入项 OutIndex 引用与脚本 SIGOUT_SELF 同序位定位。
	// Serial 不进入输出 envelope 的规范编码，仅在计算输出树时校验须等于其位置下标。
	Serial uint32
	// DigestFlags 是摘要标记位：高 4 位中 bit7=账户摘要、bit6=内容摘要、bit5=脚本摘要、
	// bit4 未用（必须为 0）。低 4 位不使用（由 Type 单独表达）。
	// 摘要标记只影响输出项叶子哈希前像，不改变 payload 字段顺序或状态归属。
	DigestFlags uint8
	// Type 是输出类型值。
	Type OutputType
	// Payload 是信元载荷字节（三类编码见 coin.go/credit.go/proof.go）。
	Payload []byte
	// LockScript 是锁定脚本，长度受 MaxLockScript 限制。
	LockScript []byte
}

// Config 计算输出公共头配置字节（第 06 章 §6，DEC-0101）。
// 高 4 位为摘要标记（DigestFlags & digestMask），低 4 位为类型值。
// bit4 必须保持未置位；类型值须为币金/凭信/存证之一（预留 0 及未知值被拒绝）。
func (o Output) Config() (byte, error) {
	if o.DigestFlags&^digestMask != 0 {
		// bit4 或低 4 位内有非法位被置位。
		return 0, ErrOutputDigestFlags
	}
	switch o.Type {
	case TypeCoin, TypeCredit, TypeProof:
	default:
		// 预留类型值 0 与未知类型值（非法位置）一律拒绝。
		return 0, ErrOutputType
	}
	return (o.DigestFlags & digestMask) | byte(o.Type), nil
}

// InState 报告该输出是否进入 UTXO/UTCO 状态集并可作为后续输入源。
// 存证不入集；币金入 UTXO，凭信入 UTCO。摘要标记不影响状态归属。
func (o Output) InState() bool {
	return o.Type == TypeCoin || o.Type == TypeCredit
}

// appendCanonical 追加输出 envelope 的规范编码：
//
//	Config(byte) || Payload || LockScript(varint(len)||bytes)
//
// Payload 直接追加（三类载荷各字段自带长度前缀，整体自界定）。
// 当 LockScript 超过 MaxLockScript 时返回 ErrLockScriptTooLong。
func (o Output) appendCanonical(dst []byte) ([]byte, error) {
	if len(o.LockScript) > types.MaxLockScript {
		return nil, ErrLockScriptTooLong
	}
	cfg, err := o.Config()
	if err != nil {
		return nil, err
	}
	dst = append(dst, cfg)
	dst = append(dst, o.Payload...)
	dst = types.AppendBytes(dst, o.LockScript)
	return dst, nil
}

// appendLeafPreimage 追加输出项叶哈希前像（第 04 章 §3.4 / DEC-0101 / DEC-0002）。
// 与 appendCanonical 不同，当摘要标记（DigestAccount/Content/Script）置位时，
// 相应片段由 SHA3-384 摘要（48B）替代原始字节，且替代发生在片段的规范编码位置。
// 未设置任何标记时，前像字节与规范编码等同。
//
// 前像结构（各类型，无标记时等于规范编码）：
//
//	Coin  : Config || Receiver(encoded) || Amount(varint) || Memo(encoded) || Script(encoded)
//	Credit: Config || Receiver(encoded) || Content(encoded fields...) || Script(encoded)
//	Proof : Config || Content(encoded fields...) || Script(encoded)
//
// 摘要域标签（DEC-0002）：
//
//	DigestAccount → SHA3-384("output.digest.account"  || receiver_raw)     替代 Receiver 编码字段
//	DigestContent → SHA3-384("output.digest.content"  || content_bytes)    替代全部内容字段
//	DigestScript  → SHA3-384("output.digest.script"   || lockscript_raw)   替代 LockScript 编码字段
//
// TypeCoin/Credit 通过 parseCoin/parseCredit + payloadLeafPreimage 计算载荷部分；
// TypeProof 直接使用 o.Payload 原始字节（空载荷合法，不经 round-trip 解码）。
// DigestAccount 对 Proof 无效（无接收者字段）。
func (o Output) appendLeafPreimage(dst []byte) ([]byte, error) {
	if len(o.LockScript) > types.MaxLockScript {
		return nil, ErrLockScriptTooLong
	}
	cfg, err := o.Config()
	if err != nil {
		return nil, err
	}
	dst = append(dst, cfg)

	switch o.Type {
	case TypeCoin:
		c, err := parseCoin(o.Payload)
		if err != nil {
			return nil, err
		}
		dst = c.payloadLeafPreimage(dst, o.DigestFlags)
	case TypeCredit:
		c, err := parseCredit(o.Payload)
		if err != nil {
			return nil, err
		}
		dst = c.payloadLeafPreimage(dst, o.DigestFlags)
	case TypeProof:
		// Proof 无接收者/内容拆分；直接以 Payload 字节作为内容段（空载荷合法）。
		if o.DigestFlags&DigestContent != 0 {
			h := crypto.HashOutputDigestContent(o.Payload)
			dst = append(dst, h.Bytes()...)
		} else {
			dst = append(dst, o.Payload...)
		}
	default:
		return nil, ErrOutputType
	}

	// 脚本段（LockScript）。
	if o.DigestFlags&DigestScript != 0 {
		h := crypto.HashOutputDigestScript(o.LockScript)
		dst = append(dst, h.Bytes()...)
	} else {
		dst = types.AppendBytes(dst, o.LockScript)
	}

	return dst, nil
}
