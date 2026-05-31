package tx

import "github.com/cxio/evidcoin/pkg/types"

// CanonicalBody 组装不含见证的规范交易体（第 06 章 §4）：
//
//	Header || Inputs(varint(count)||Input*) || Outputs(varint(count)||Output*)
//
// 解锁脚本计入交易体；签名见证不在此编码（见证定义与剪枝见第 04、08 章）。
// 任一组成部分编码失败（脚本超限、短引用过短、MintPKHash 非法等）时返回相应错误。
func CanonicalBody(h *TxHeader, in Inputs, outputs []Output) ([]byte, error) {
	head, err := h.CanonicalBytes()
	if err != nil {
		return nil, err
	}
	inputs, err := in.canonicalList()
	if err != nil {
		return nil, err
	}
	outs, err := canonicalOutputs(outputs)
	if err != nil {
		return nil, err
	}
	body := make([]byte, 0, len(head)+len(inputs)+len(outs))
	body = append(body, head...)
	body = append(body, inputs...)
	body = append(body, outs...)
	return body, nil
}

// TxSize 返回不含见证的规范交易体字节长度（第 06 章 §7）。
func TxSize(h *TxHeader, in Inputs, outputs []Output) (int, error) {
	body, err := CanonicalBody(h, in, outputs)
	if err != nil {
		return 0, err
	}
	return len(body), nil
}

// CheckTxSize 校验交易体（不含见证）尺寸不超过 MaxTxSize（65535 字节）。
// 超过时返回 ErrTxTooLarge。
func CheckTxSize(h *TxHeader, in Inputs, outputs []Output) error {
	size, err := TxSize(h, in, outputs)
	if err != nil {
		return err
	}
	if size > types.MaxTxSize {
		return ErrTxTooLarge
	}
	return nil
}
