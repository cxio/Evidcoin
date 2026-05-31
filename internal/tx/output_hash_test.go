package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// coinOut 构造一个位于 serial 序位的币金输出，便于输出树测试。
func coinOut(t *testing.T, serial uint32, amount uint64) Output {
	t.Helper()
	pl, err := Coin{Amount: types.Amount(amount)}.Payload()
	if err != nil {
		t.Fatalf("Coin.Payload: %v", err)
	}
	return Output{Serial: serial, Type: TypeCoin, Payload: pl}
}

// TestHashOutputsEmptyRejected 校验普通交易空输出集被拒绝。
func TestHashOutputsEmptyRejected(t *testing.T) {
	if _, err := HashOutputs(nil); err == nil {
		t.Fatal("空输出集应被拒绝")
	}
}

// TestHashOutputsSingleLeafRoot 校验单输出根等于单叶树按 tree.branch 归一化的 32B 根。
func TestHashOutputsSingleLeafRoot(t *testing.T) {
	o := coinOut(t, 0, 100)
	got, err := HashOutputs([]Output{o})
	if err != nil {
		t.Fatalf("HashOutputs: %v", err)
	}
	canon, err := o.appendCanonical(nil)
	if err != nil {
		t.Fatalf("appendCanonical: %v", err)
	}
	tree, err := hashtree.BuildFromPayloads([][]byte{canon})
	if err != nil {
		t.Fatalf("BuildFromPayloads: %v", err)
	}
	if got.Bytes() == nil || string(got.Bytes()) != string(tree.Root()) {
		t.Fatalf("单输出根不匹配\n got=%x\nwant=%x", got.Bytes(), tree.Root())
	}
}

// TestHashOutputsOrderMatters 校验输出顺序变化导致根变化。
func TestHashOutputsOrderMatters(t *testing.T) {
	a := coinOut(t, 0, 100)
	b := coinOut(t, 1, 200)
	r1, err := HashOutputs([]Output{a, b})
	if err != nil {
		t.Fatalf("HashOutputs(ab): %v", err)
	}
	// 交换顺序后须重排 Serial 以满足位置约束，根应不同。
	a2 := coinOut(t, 1, 100)
	b2 := coinOut(t, 0, 200)
	r2, err := HashOutputs([]Output{b2, a2})
	if err != nil {
		t.Fatalf("HashOutputs(ba): %v", err)
	}
	if r1 == r2 {
		t.Fatal("不同顺序的输出集根不应相同")
	}
}

// TestHashOutputsMultiPath 校验多输出（含奇数）可计算且与 hashtree 一致。
func TestHashOutputsMultiPath(t *testing.T) {
	for _, n := range []int{2, 3, 5} {
		outs := make([]Output, n)
		payloads := make([][]byte, n)
		for i := 0; i < n; i++ {
			outs[i] = coinOut(t, uint32(i), uint64(10*i+1))
			canon, err := outs[i].appendCanonical(nil)
			if err != nil {
				t.Fatalf("appendCanonical: %v", err)
			}
			payloads[i] = canon
		}
		got, err := HashOutputs(outs)
		if err != nil {
			t.Fatalf("HashOutputs(n=%d): %v", n, err)
		}
		tree, err := hashtree.BuildFromPayloads(payloads)
		if err != nil {
			t.Fatalf("BuildFromPayloads(n=%d): %v", n, err)
		}
		if string(got.Bytes()) != string(tree.Root()) {
			t.Fatalf("多输出根不匹配(n=%d)\n got=%x\nwant=%x", n, got.Bytes(), tree.Root())
		}
	}
}

// TestHashOutputsSerialMismatch 校验 Serial 与位置不符时拒绝。
func TestHashOutputsSerialMismatch(t *testing.T) {
	a := coinOut(t, 0, 100)
	b := coinOut(t, 5, 200) // 序位应为 1，却标 5
	if _, err := HashOutputs([]Output{a, b}); err == nil {
		t.Fatal("Serial 与位置不符应被拒绝")
	}
}
