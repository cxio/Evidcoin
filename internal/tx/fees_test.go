package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// coinOutput 构造一个指定序位与金额的币金输出（用于交易费测试）。
func coinOutput(t *testing.T, serial uint32, amount uint64) Output {
	t.Helper()
	pl, err := Coin{Amount: types.Amount(amount)}.Payload()
	if err != nil {
		t.Fatalf("Coin.Payload: %v", err)
	}
	return Output{Serial: serial, Type: TypeCoin, Payload: pl}
}

// TestSumCoinOutputs 校验仅对币金输出求和，忽略非币金输出。
func TestSumCoinOutputs(t *testing.T) {
	proofPayload, err := Proof{Title: []byte("t")}.Payload()
	if err != nil {
		t.Fatalf("Proof.Payload: %v", err)
	}
	outs := []Output{
		coinOutput(t, 0, 100),
		{Serial: 1, Type: TypeProof, Payload: proofPayload}, // 非币金，不计入
		coinOutput(t, 2, 250),
	}
	sum, err := SumCoinOutputs(outs)
	if err != nil {
		t.Fatalf("SumCoinOutputs: %v", err)
	}
	if sum != types.Amount(350) {
		t.Fatalf("币金输出总额应为 350，got=%d", sum)
	}
}

// TestTxFeeConservation 校验交易费 = 输入币金总额 - 输出币金总额；输出超过输入则拒绝。
func TestTxFeeConservation(t *testing.T) {
	outs := []Output{coinOutput(t, 0, 300)}
	fee, err := TxFee(types.Amount(500), outs)
	if err != nil {
		t.Fatalf("TxFee: %v", err)
	}
	if fee != types.Amount(200) {
		t.Fatalf("交易费应为 200，got=%d", fee)
	}
	if _, err := TxFee(types.Amount(100), outs); err == nil {
		t.Fatal("输出总额大于输入总额应被拒绝")
	}
}

// TestTxFeeManyOutputsAllowed 校验不因输出项数量超过输入项数量 2 倍而拒绝普通交易
// （百日扩张比例属客户端构造策略，非协议拒绝规则）。
func TestTxFeeManyOutputsAllowed(t *testing.T) {
	outs := make([]Output, 10)
	for i := range outs {
		outs[i] = coinOutput(t, uint32(i), 10)
	}
	// 10 个输出对 1 个输入（远超 2 倍），总额 100 ≤ 输入 100，应正常返回。
	fee, err := TxFee(types.Amount(100), outs)
	if err != nil {
		t.Fatalf("大量输出不应被拒绝: %v", err)
	}
	if fee != types.Amount(0) {
		t.Fatalf("交易费应为 0，got=%d", fee)
	}
}
