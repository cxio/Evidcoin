package tx_test

import (
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// ---- varint 测试 ----

// TestVarintRoundtrip 测试可变长整数编解码往返一致性。
func TestVarintRoundtrip(t *testing.T) {
	cases := []uint64{0, 1, 127, 128, 16383, 16384, 1<<32 - 1, 1<<63 - 1}
	for _, v := range cases {
		buf := tx.AppendVarint(nil, v)
		got, n, err := tx.ReadVarint(buf)
		if err != nil {
			t.Errorf("ReadVarint(%d) error = %v", v, err)
			continue
		}
		if got != v {
			t.Errorf("ReadVarint(%d) = %d, want %d", v, got, v)
		}
		if n != len(buf) {
			t.Errorf("ReadVarint(%d) consumed %d bytes, expected %d", v, n, len(buf))
		}
	}
}

// ---- 哈希计算测试 ----

// TestComputeTxIDDeterministic 测试 TxID 计算确定性。
func TestComputeTxIDDeterministic(t *testing.T) {
	header := &tx.TxHeader{
		Version:     1,
		Timestamp:   1700000000000,
		HashInputs:  types.Hash256{1},
		HashOutputs: types.Hash256{2},
	}
	id1 := tx.ComputeTxID(header)
	id2 := tx.ComputeTxID(header)
	if id1 != id2 {
		t.Error("ComputeTxID is not deterministic")
	}
}

// TestComputeTxIDDifferentTimestamp 测试不同时间戳产生不同 TxID。
func TestComputeTxIDDifferentTimestamp(t *testing.T) {
	h1 := &tx.TxHeader{Version: 1, Timestamp: 100}
	h2 := &tx.TxHeader{Version: 1, Timestamp: 200}
	if tx.ComputeTxID(h1) == tx.ComputeTxID(h2) {
		t.Error("different timestamps should produce different TxIDs")
	}
}

// TestComputeLeadHashDeterministic 测试首领输入哈希确定性。
func TestComputeLeadHashDeterministic(t *testing.T) {
	lead := &tx.LeadInput{Year: 2024, TxID: types.Hash384{0xaa}, OutIndex: 0}
	h1 := tx.ComputeLeadHash(lead)
	h2 := tx.ComputeLeadHash(lead)
	if h1 != h2 {
		t.Error("ComputeLeadHash is not deterministic")
	}
}

// TestComputeOutputHashEmpty 测试空输出集哈希为零。
func TestComputeOutputHashEmpty(t *testing.T) {
	h := tx.ComputeOutputHash(nil)
	if !h.IsZero() {
		t.Error("empty output hash should be zero")
	}
}

// TestComputeOutputHashDeterministic 测试输出哈希确定性。
func TestComputeOutputHashDeterministic(t *testing.T) {
	outputs := []tx.Output{
		{Serial: 0, Config: types.OutTypeCoin, Amount: 1000, Address: types.PubKeyHash{1}},
		{Serial: 1, Config: types.OutTypeCoin, Amount: 500, Address: types.PubKeyHash{2}},
	}
	h1 := tx.ComputeOutputHash(outputs)
	h2 := tx.ComputeOutputHash(outputs)
	if h1 != h2 {
		t.Error("ComputeOutputHash is not deterministic")
	}
}

// TestComputeCheckRoot 测试 CheckRoot 合并哈希。
func TestComputeCheckRoot(t *testing.T) {
	txRoot := types.Hash256{0x01}
	utxoFP := types.Hash256{0x02}
	utcoFP := types.Hash256{0x03}
	r1 := tx.ComputeCheckRoot(txRoot, utxoFP, utcoFP)
	r2 := tx.ComputeCheckRoot(txRoot, utxoFP, utcoFP)
	if r1 != r2 {
		t.Error("ComputeCheckRoot is not deterministic")
	}
	// 改变任一输入应产生不同结果
	r3 := tx.ComputeCheckRoot(types.Hash256{0x99}, utxoFP, utcoFP)
	if r1 == r3 {
		t.Error("ComputeCheckRoot collision on different inputs")
	}
}

// ---- 输出类型测试 ----

// TestOutputType 测试输出类型字段提取。
func TestOutputType(t *testing.T) {
	cases := []struct {
		config byte
		want   byte
	}{
		{types.OutTypeCoin, types.OutTypeCoin},
		{types.OutTypeCredit, types.OutTypeCredit},
		{types.OutTypeProof, types.OutTypeProof},
		{tx.OutFlagDestroy | types.OutTypeCoin, types.OutTypeCoin},
	}
	for _, tc := range cases {
		o := &tx.Output{Config: tc.config}
		if got := o.Type(); got != tc.want {
			t.Errorf("Config 0x%02x Type() = %d, want %d", tc.config, got, tc.want)
		}
	}
}

// TestOutputCanBeUTXO 测试 UTXO 资格判断。
func TestOutputCanBeUTXO(t *testing.T) {
	cases := []struct {
		name   string
		config byte
		want   bool
	}{
		{"coin normal", types.OutTypeCoin, true},
		{"coin destroy", tx.OutFlagDestroy | types.OutTypeCoin, false},
		{"credit normal", types.OutTypeCredit, false},
		{"proof normal", types.OutTypeProof, false},
		{"custom class", tx.OutFlagCustomClass | types.OutTypeCoin, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := &tx.Output{Config: tc.config}
			if got := o.CanBeUTXO(); got != tc.want {
				t.Errorf("CanBeUTXO() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ---- 附件 ID 测试 ----

// TestAttachmentIDValidate 测试附件 ID 验证逻辑。
func TestAttachmentIDValidate(t *testing.T) {
	fp := types.Hash512{}
	root := types.Hash256{0xcc}
	valid := &tx.AttachmentID{
		TotalLen:      20,
		Fingerprint:   fp,
		ShardCount:    1,
		ShardTreeRoot: &root,
		DataSize:      1024,
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}

	// ShardCount=0 但有 ShardTreeRoot：应报错
	invalid := &tx.AttachmentID{
		Fingerprint:   fp,
		ShardCount:    0,
		ShardTreeRoot: &root,
	}
	if err := invalid.Validate(); err == nil {
		t.Error("expected error when ShardCount=0 but ShardTreeRoot set")
	}

	// ShardCount>0 但无 ShardTreeRoot：应报错
	invalid2 := &tx.AttachmentID{
		Fingerprint:   fp,
		ShardCount:    2,
		ShardTreeRoot: nil,
	}
	if err := invalid2.Validate(); err == nil {
		t.Error("expected error when ShardCount>0 but ShardTreeRoot nil")
	}
}

// TestCoinFee 测试手续费计算。
func TestCoinFee(t *testing.T) {
	outputs := []tx.Output{
		{Config: types.OutTypeCoin, Amount: 300},
		{Config: types.OutTypeCoin, Amount: 200},
	}
	fee, err := tx.CoinFee(600, outputs)
	if err != nil {
		t.Fatalf("CoinFee() error = %v", err)
	}
	if fee != 100 {
		t.Errorf("CoinFee() = %d, want 100", fee)
	}
}

// TestCoinFeeNegative 测试输出超过输入时返回错误。
func TestCoinFeeNegative(t *testing.T) {
	outputs := []tx.Output{
		{Config: types.OutTypeCoin, Amount: 1000},
	}
	_, err := tx.CoinFee(500, outputs)
	if err == nil {
		t.Error("expected error for negative fee, got nil")
	}
}

// TestMultiSigUnlockMN 测试多签 M/N 计算。
func TestMultiSigUnlockMN(t *testing.T) {
	m := &tx.MultiSigUnlock{
		Signatures: [][]byte{{1}, {2}},
		PublicKeys: [][]byte{{3}, {4}},
		Complement: [][32]byte{{5}, {6}, {7}},
	}
	if m.M() != 2 {
		t.Errorf("M() = %d, want 2", m.M())
	}
	if m.N() != 5 {
		t.Errorf("N() = %d, want 5", m.N())
	}
}

// TestValidateSize 测试交易大小验证。
func TestValidateSize(t *testing.T) {
	if err := tx.ValidateSize(1000); err != nil {
		t.Errorf("ValidateSize(1000) unexpected error: %v", err)
	}
	if err := tx.ValidateSize(65536); err == nil {
		t.Error("ValidateSize(65536) should return error")
	}
}

// TestComputeTxTreeRoot 测试交易树根计算。
func TestComputeTxTreeRoot(t *testing.T) {
	// 空列表返回零值
	empty := tx.ComputeTxTreeRoot(nil)
	if !empty.IsZero() {
		t.Error("empty TxTreeRoot should be zero")
	}
	// 单个 TxID 确定性
	ids := []types.Hash384{{0xab}, {0xcd}}
	r1 := tx.ComputeTxTreeRoot(ids)
	r2 := tx.ComputeTxTreeRoot(ids)
	if r1 != r2 {
		t.Error("ComputeTxTreeRoot is not deterministic")
	}
}

// TestComputeInputHash 测试输入集合哈希计算。
func TestComputeInputHash(t *testing.T) {
	lead := &tx.LeadInput{Year: 2025, TxID: types.Hash384{0x01}, OutIndex: 2}
	rests := []tx.RestInput{
		{Year: 2024, TxIDPart: [20]byte{0x02}, OutIndex: 1, TransferIndex: -1},
		{Year: 2024, TxIDPart: [20]byte{0x03}, OutIndex: 0, TransferIndex: 3},
	}
	h1 := tx.ComputeInputHash(lead, rests)
	h2 := tx.ComputeInputHash(lead, rests)
	if h1 != h2 {
		t.Error("ComputeInputHash is not deterministic")
	}
}

// TestOutputMethods 测试 Output 的方法。
func TestOutputMethods(t *testing.T) {
	o := &tx.Output{Config: tx.OutFlagHasAttach | types.OutTypeCredit}
	if !o.HasAttachment() {
		t.Error("HasAttachment() should be true")
	}
	if !o.CanBeUTCO() {
		t.Error("CanBeUTCO() should be true for OutTypeCredit")
	}
	o2 := &tx.Output{Config: tx.OutFlagDestroy | types.OutTypeCredit}
	if o2.CanBeUTCO() {
		t.Error("CanBeUTCO() should be false for destroyed output")
	}
}

// TestTransactionID 测试 Transaction.ID 方法。
func TestTransactionID(t *testing.T) {
	txn := &tx.Transaction{
		Header: tx.TxHeader{Version: 1, Timestamp: 9999},
	}
	id1 := txn.ID()
	id2 := txn.ID()
	if id1 != id2 {
		t.Error("Transaction.ID() is not deterministic")
	}
	if txn.InputCount() != 1 {
		t.Errorf("InputCount() = %d, want 1", txn.InputCount())
	}
}

// TestCoinbaseTxID 测试 CoinbaseTx.ID 方法。
func TestCoinbaseTxID(t *testing.T) {
	cb := &tx.CoinbaseTx{
		Header:      tx.TxHeader{Version: 1, Timestamp: 12345},
		BlockHeight: 100,
		TotalReward: 5000000,
	}
	id1 := cb.ID()
	id2 := cb.ID()
	if id1 != id2 {
		t.Error("CoinbaseTx.ID() is not deterministic")
	}
}

// TestAttachmentIDEncode 测试附件 ID 序列化。
func TestAttachmentIDEncode(t *testing.T) {
	fp := types.Hash512{0xaa}
	root := types.Hash256{0xbb}
	a := &tx.AttachmentID{
		TotalLen:      80,
		MajorType:     1,
		MinorType:     2,
		Fingerprint:   fp,
		ShardCount:    1,
		ShardTreeRoot: &root,
		DataSize:      2048,
	}
	encoded := a.Encode()
	if len(encoded) == 0 {
		t.Error("Encode() should return non-empty bytes")
	}
	// 无分片树根编码
	a2 := &tx.AttachmentID{
		TotalLen:      70,
		Fingerprint:   fp,
		ShardCount:    0,
		ShardTreeRoot: nil,
		DataSize:      512,
	}
	encoded2 := a2.Encode()
	if len(encoded2) == 0 {
		t.Error("Encode() for no-shard should return non-empty bytes")
	}
}

// TestEncodeInt64 测试 int64 编码。
func TestEncodeInt64(t *testing.T) {
	b := tx.EncodeInt64(0x0102030405060708)
	expected := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	for i, v := range expected {
		if b[i] != v {
			t.Errorf("EncodeInt64 byte[%d] = %02x, want %02x", i, b[i], v)
		}
	}
}
