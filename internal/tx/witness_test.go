package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func sampleWitness() Witness {
	return Witness{Items: []WitnessItem{
		{Type: WitCategory, Data: []byte{0x01}},
		{Type: WitAuthFlag, Data: []byte{byte(SigOutSelf | AuxScript)}},
		{Type: WitSignature, Data: []byte("sig-bytes")},
		{Type: WitPublicKey, Data: []byte("pub-bytes")},
	}}
}

func TestWitnessEncodeDecodeRoundTrip(t *testing.T) {
	w := sampleWitness()
	enc, err := w.Encode()
	if err != nil {
		t.Fatal(err)
	}
	got, n, err := DecodeWitness(enc)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(enc) {
		t.Fatalf("consumed %d bytes, want %d", n, len(enc))
	}
	if len(got.Items) != len(w.Items) {
		t.Fatalf("item count = %d, want %d", len(got.Items), len(w.Items))
	}
	for i := range w.Items {
		if got.Items[i].Type != w.Items[i].Type ||
			string(got.Items[i].Data) != string(w.Items[i].Data) {
			t.Fatalf("item %d mismatch: %+v vs %+v", i, got.Items[i], w.Items[i])
		}
	}
}

func TestWitnessEmptyValid(t *testing.T) {
	var w Witness
	enc, err := w.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) != 1 || enc[0] != 0x00 {
		t.Fatalf("empty witness encoding = %x, want 00", enc)
	}
	got, n, err := DecodeWitness(enc)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || len(got.Items) != 0 {
		t.Fatal("empty witness must decode to zero items")
	}
}

func TestWitnessInvalidItemType(t *testing.T) {
	w := Witness{Items: []WitnessItem{{Type: WitnessItemType(0x07), Data: nil}}}
	if _, err := w.Encode(); err != ErrWitnessItemType {
		t.Fatalf("expected ErrWitnessItemType, got %v", err)
	}
	// 解码侧：count=1 || type=0x09 || ...
	bad := types.AppendVarUint(nil, 1)
	bad = append(bad, 0x09)
	bad = types.AppendBytes(bad, nil)
	if _, _, err := DecodeWitness(bad); err != ErrWitnessItemType {
		t.Fatalf("decode expected ErrWitnessItemType, got %v", err)
	}
}

func TestWitnessTruncated(t *testing.T) {
	// 声明 2 个 item 但只给 1 个。
	enc := types.AppendVarUint(nil, 2)
	enc = append(enc, byte(WitSignature))
	enc = types.AppendBytes(enc, []byte("only-one"))
	if _, _, err := DecodeWitness(enc); err == nil {
		t.Fatal("truncated witness must fail to decode")
	}
}

// TestWitnessNotInTxID 校验见证不计入 TxID：相同交易头在不同见证下 TxID 不变，
// 但完整编码（交易体 || 见证）随见证变化。
func TestWitnessNotInTxID(t *testing.T) {
	h := &TxHeader{
		Version:     1,
		HashInputs:  types.Hash32{},
		HashOutputs: types.Hash32{},
		Timestamp:   123,
	}
	id1, err := h.TxID()
	if err != nil {
		t.Fatal(err)
	}

	w1 := sampleWitness()
	w2 := Witness{Items: []WitnessItem{{Type: WitSignature, Data: []byte("different")}}}
	enc1, _ := w1.Encode()
	enc2, _ := w2.Encode()

	// TxID 仅由交易头决定，与见证无关。
	id2, _ := h.TxID()
	if string(id1.Bytes()) != string(id2.Bytes()) {
		t.Fatal("TxID must not depend on witness")
	}
	if string(enc1) == string(enc2) {
		t.Fatal("different witnesses must produce different encodings")
	}
}

func TestWitnessPruneEmpties(t *testing.T) {
	w := sampleWitness()
	if len(w.Prune().Items) != 0 {
		t.Fatal("pruning a standard input witness must remove all items")
	}
}
