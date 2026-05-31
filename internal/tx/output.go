package tx

import "github.com/cxio/evidcoin/pkg/types"

// maxCustomIDLen 是自定义类私有标识 ID 的最大长度（Config 低 7 位计数上限，第 06 章 §6）。
const maxCustomIDLen = 127

// OutputType 是标准输出的低 4 位类型值（第 06 章 §6）。
type OutputType uint8

const (
	// TypeReserved 是预留类型值 0，无意义，结构验证拒绝。
	TypeReserved OutputType = 0
	// TypeCoin 是币金输出（可作输入源，入 UTXO）。
	TypeCoin OutputType = 1
	// TypeCredit 是凭信输出（可作输入源，入 UTCO）。
	TypeCredit OutputType = 2
	// TypeProof 是存证输出（不可作输入源，不入集；介管脚本亦属此类）。
	TypeProof OutputType = 3
)

// Output 是交易输出 envelope（第 06 章 §6）：公共头 Config + 信元载荷 Payload
// + 锁定脚本 LockScript。Config 字节高 4 位为位标记，低 4 位为类型值；无销毁位，
// 普通交易不可销毁币金（销毁仅由 Coinbase BurnCoin 表达）。
type Output struct {
	// Serial 是输出序位（从 0 开始，第 04 章 §3.4 / 附.交易.md）。它由输出在交易
	// 输出集中的位置决定，用于输入项 OutIndex 引用与脚本 SIGOUT_SELF 同序位定位。
	// Serial 不进入输出 envelope 的规范编码，仅在计算输出树时校验须等于其位置下标。
	Serial uint32
	// IsCustom 标识自定义类输出（Config bit7=1）。自定义类不进 UTXO/UTCO，
	// 不能作为后续输入源，节点仅校验编码合法性。
	IsCustom bool
	// CustomID 是自定义类私有标识 ID（≤127 字节），仅 IsCustom 时有效。
	CustomID []byte
	// Type 是标准类型值（仅非自定义类时有效）。
	Type OutputType
	// HasAttachment 是包含附件标记（Config bit6，仅标准类时有效）。
	HasAttachment bool
	// Payload 是信元载荷字节（三类编码见 coin.go/credit.go/proof.go）。
	Payload []byte
	// LockScript 是锁定脚本，长度受 MaxLockScript 限制。
	LockScript []byte
}

// Config 计算输出公共头配置字节（第 06 章 §6）。
// 自定义类（IsCustom）：bit7=1，低 7 位为 CustomID 长度计数（≤127）。
// 标准类：bit6=HasAttachment，bit[3:0]=类型值（须为币金/凭信/存证之一）。
func (o Output) Config() (byte, error) {
	if o.IsCustom {
		if len(o.CustomID) > maxCustomIDLen {
			return 0, ErrCustomIDTooLong
		}
		return 0x80 | byte(len(o.CustomID)), nil
	}
	switch o.Type {
	case TypeCoin, TypeCredit, TypeProof:
	default:
		// 预留类型值 0 与未知类型值（非法位置）一律拒绝。
		return 0, ErrOutputType
	}
	var cfg byte
	if o.HasAttachment {
		cfg |= 0x40
	}
	cfg |= byte(o.Type)
	return cfg, nil
}

// InState 报告该输出是否进入 UTXO/UTCO 状态集并可作为后续输入源。
// 自定义类与存证不入集；币金入 UTXO，凭信入 UTCO。
func (o Output) InState() bool {
	if o.IsCustom {
		return false
	}
	return o.Type == TypeCoin || o.Type == TypeCredit
}

// appendCanonical 追加输出 envelope 的规范编码：
//
//	Config(byte) || [CustomID if custom] || Payload || LockScript(varint(len)||bytes)
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
	if o.IsCustom {
		dst = append(dst, o.CustomID...)
	}
	dst = append(dst, o.Payload...)
	dst = types.AppendBytes(dst, o.LockScript)
	return dst, nil
}
