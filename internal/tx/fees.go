package tx

import "github.com/cxio/evidcoin/pkg/types"

// SumCoinOutputs 累加输出集中所有币金输出的 Amount 字段（第 06 章 §7）。
// 币金输出 Payload 编码顺序为 Receiver || Amount || Memo（见 coin.go），通过 parseCoin 解析。
// 非币金输出（凭信/存证/自定义）忽略不计。
// 当某币金输出 Payload 解码失败时返回 ErrCoinAmountDecode；
// 累加溢出 uint64 时返回 types.ErrAmountOverflow。
func SumCoinOutputs(outputs []Output) (types.Amount, error) {
	var sum uint64
	for i := range outputs {
		o := outputs[i]
		if o.Type != TypeCoin {
			continue
		}
		coin, err := parseCoin(o.Payload)
		if err != nil {
			return 0, ErrCoinAmountDecode
		}
		amount := uint64(coin.Amount)
		if sum > ^uint64(0)-amount {
			return 0, types.ErrAmountOverflow
		}
		sum += amount
	}
	return types.Amount(sum), nil
}

// TxFee 计算普通交易费 = 输入币金总额 - 输出币金总额（第 06 章 §7：币金守恒）。
// inputCoinTotal 为首领与其余币金输入的来源金额合计，由验证层从 UTXO 解析后注入
// （本层不查询状态）。当输出币金总额大于输入币金总额时返回 ErrCoinNotConserved。
//
// 输出项数量是否超过输入项数量 2 倍不是协议拒绝规则（百日扩张比例属客户端构造策略，
// 见 proposal 05/14），故本函数不对输入/输出数量做任何约束。
func TxFee(inputCoinTotal types.Amount, outputs []Output) (types.Amount, error) {
	outTotal, err := SumCoinOutputs(outputs)
	if err != nil {
		return 0, err
	}
	if outTotal > inputCoinTotal {
		return 0, ErrCoinNotConserved
	}
	return inputCoinTotal - outTotal, nil
}
