package tx

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// maxShortField 是 <256 字节字段（Receiver/Creator/Title/Memo 等）的最大长度。
const maxShortField = 255

// Coin 是币金信元载荷（类型值 1，入 UTXO，第 07 章 §2）。
// 编码顺序固定为 Receiver || Amount || Memo（DEC-0101 冻结，不可调整）。
type Coin struct {
	// Receiver 是接收者公钥哈希（标准地址，<256 字节）；
	// 若脚本自定义验证（不用 SYS_CHKPASS），可为空或任意 <256 字节序列。
	Receiver []byte
	// Amount 是币金数量，最小单位 chx（1 Bi = 10^8 chx，第 01 章 C-8）。
	Amount types.Amount
	// Memo 是附言，可选，最多 255 字节；缺省以 varint(0) 编码并参与前像。
	Memo []byte
}

// Payload 返回 Coin 载荷的规范编码（第 07 章 §2）：
//
//	Receiver(varint(len)||bytes) || Amount(varint) || Memo(varint(len)||bytes)
//
// 当 Receiver 或 Memo 超过 255 字节时返回相应错误。Coin 载荷不含 AttachmentID，
// 锁定脚本属输出公共头（见 output.go），不在此编码。
func (c Coin) Payload() ([]byte, error) {
	if len(c.Receiver) > maxShortField {
		return nil, ErrReceiverTooLong
	}
	if len(c.Memo) > maxShortField {
		return nil, ErrMemoTooLong
	}
	dst := types.AppendBytes(nil, c.Receiver)
	dst = types.AppendVarUint(dst, uint64(c.Amount))
	dst = types.AppendBytes(dst, c.Memo)
	return dst, nil
}

// parseCoin 从规范载荷字节解码为 Coin 结构体（DEC-0101 字段顺序：Receiver||Amount||Memo）。
func parseCoin(payload []byte) (Coin, error) {
	receiver, n, err := types.ReadBytes(payload)
	if err != nil {
		return Coin{}, err
	}
	amount, n2, err := types.ReadVarUint(payload[n:])
	if err != nil {
		return Coin{}, err
	}
	memo, _, err := types.ReadBytes(payload[n+n2:])
	if err != nil {
		return Coin{}, err
	}
	return Coin{Receiver: receiver, Amount: types.Amount(amount), Memo: memo}, nil
}

// payloadLeafPreimage 将 Coin 在输出项叶哈希前像中的载荷部分追加到 dst（DEC-0101/DEC-0002）。
// flags 为 Output.DigestFlags，bit7=账户摘要（Receiver），bit6=内容摘要（Amount+Memo）；
// bit5=脚本摘要由 Output.appendLeafPreimage 负责，不在本方法范围内。
// 前像顺序（编码段位置不变）：Receiver_or_digest || Amount_or_digest || Memo（内容摘要时省略）。
func (c Coin) payloadLeafPreimage(dst []byte, flags uint8) []byte {
	digestAcct := flags&DigestAccount != 0
	digestCont := flags&DigestContent != 0

	// 账户段 = Receiver（新编码第 1 段）。
	if digestAcct {
		h := crypto.HashOutputDigestAccount(c.Receiver)
		dst = append(dst, h.Bytes()...)
	} else {
		dst = types.AppendBytes(dst, c.Receiver)
	}

	// 内容段 = Amount + Memo（新编码第 2、3 段）；DigestContent 时合并为 48B 摘要。
	if digestCont {
		cb := types.AppendVarUint(nil, uint64(c.Amount))
		cb = append(cb, types.AppendBytes(nil, c.Memo)...)
		h := crypto.HashOutputDigestContent(cb)
		dst = append(dst, h.Bytes()...)
	} else {
		dst = types.AppendVarUint(dst, uint64(c.Amount))
		dst = types.AppendBytes(dst, c.Memo)
	}
	return dst
}
