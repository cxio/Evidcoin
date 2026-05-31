package tx

import "github.com/cxio/evidcoin/pkg/types"

// maxShortField 是 <256 字节字段（Receiver/Creator/Title/Memo 等）的最大长度。
const maxShortField = 255

// Coin 是币金信元载荷（类型值 1，入 UTXO，第 07 章 §2）。
// 编码顺序固定为 Amount || Receiver || Memo（DEC-0101）。
type Coin struct {
	// Amount 是币金数量，最小单位 chx（1 Bi = 10^8 chx，第 01 章 C-8）。
	Amount types.Amount
	// Receiver 是接收者公钥哈希（标准地址，<256 字节）；
	// 若脚本自定义验证（不用 SYS_CHKPASS），可为空或任意 <256 字节序列。
	Receiver []byte
	// Memo 是附言，可选，最多 255 字节；缺省以 varint(0) 编码并参与前像。
	Memo []byte
}

// Payload 返回 Coin 载荷的规范编码（第 07 章 §2）：
//
//	Amount(varint) || Receiver(varint(len)||bytes) || Memo(varint(len)||bytes)
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
	dst := types.AppendVarUint(nil, uint64(c.Amount))
	dst = types.AppendBytes(dst, c.Receiver)
	dst = types.AppendBytes(dst, c.Memo)
	return dst, nil
}
