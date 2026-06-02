package services

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestSummary 验证区块概要 TxID profile 与碰撞回退（第 15 章 §4，DEC-0602）。
func TestSummary(t *testing.T) {
	t.Run("txid_prefix_len_constant", func(t *testing.T) {
		// TxIDPrefixLen 必须精确为 16 字节（DEC-0602）。
		if TxIDPrefixLen != 16 {
			t.Errorf("TxIDPrefixLen = %d, want 16", TxIDPrefixLen)
		}
	})

	t.Run("new_txid_prefix_extracts_first_16_bytes", func(t *testing.T) {
		// NewTxIDPrefix 提取 TxID 的前 16 字节，与后 32 字节无关。
		raw := make([]byte, 48)
		for i := range raw {
			raw[i] = byte(i + 1) // 1..48
		}
		txid := types.MustTxID(raw)
		prefix := NewTxIDPrefix(txid)

		// 前 16 字节应为 1..16。
		for i := 0; i < TxIDPrefixLen; i++ {
			if prefix[i] != byte(i+1) {
				t.Errorf("prefix[%d] = %d, want %d", i, prefix[i], i+1)
			}
		}
	})

	t.Run("new_txid_prefix_same_prefix_bytes_equal", func(t *testing.T) {
		// 前 16 字节相同的两个 TxID，前缀必须相等。
		rawA := make([]byte, 48)
		rawB := make([]byte, 48)
		for i := 0; i < 16; i++ {
			rawA[i] = byte(i)
			rawB[i] = byte(i)
		}
		// 后 32 字节不同。
		rawA[16] = 0xAA
		rawB[16] = 0xBB
		txA := types.MustTxID(rawA)
		txB := types.MustTxID(rawB)
		if NewTxIDPrefix(txA) != NewTxIDPrefix(txB) {
			t.Error("prefixes with same first 16 bytes must be equal")
		}
	})

	t.Run("new_txid_prefix_different_prefix_bytes_unequal", func(t *testing.T) {
		// 前 16 字节不同的两个 TxID，前缀必须不等。
		rawA := make([]byte, 48)
		rawB := make([]byte, 48)
		rawA[0] = 0x01
		rawB[0] = 0x02
		txA := types.MustTxID(rawA)
		txB := types.MustTxID(rawB)
		if NewTxIDPrefix(txA) == NewTxIDPrefix(txB) {
			t.Error("prefixes with different first bytes must not be equal")
		}
	})

	t.Run("new_block_summary_prefix_count", func(t *testing.T) {
		// NewBlockSummary 应为每个 TxID 创建一个前缀，TxCount 与前缀数一致。
		blockID := types.MustBlockID(make([]byte, 48))
		txIDs := []types.TxID{
			types.MustTxID(make([]byte, 48)),
			types.MustTxID(append(make([]byte, 47), 0x01)),
			types.MustTxID(append(make([]byte, 47), 0x02)),
		}
		summary := NewBlockSummary(blockID, txIDs)

		if summary.TxCount != 3 {
			t.Errorf("TxCount = %d, want 3", summary.TxCount)
		}
		if len(summary.TxIDPrefixes) != 3 {
			t.Errorf("TxIDPrefixes len = %d, want 3", len(summary.TxIDPrefixes))
		}
		if summary.BlockID != blockID {
			t.Error("BlockID mismatch")
		}
	})

	t.Run("new_block_summary_prefix_content", func(t *testing.T) {
		// 每个前缀内容必须与对应 TxID 的前 16 字节一致。
		blockID := types.MustBlockID(make([]byte, 48))
		rawTx := make([]byte, 48)
		for i := range rawTx {
			rawTx[i] = byte(i + 0x10)
		}
		txid := types.MustTxID(rawTx)
		summary := NewBlockSummary(blockID, []types.TxID{txid})

		want := NewTxIDPrefix(txid)
		if summary.TxIDPrefixes[0] != want {
			t.Error("TxIDPrefix content mismatch")
		}
	})

	t.Run("new_block_summary_empty_txids", func(t *testing.T) {
		// 空 TxID 序列：TxCount=0，TxIDPrefixes 为空。
		blockID := types.MustBlockID(make([]byte, 48))
		summary := NewBlockSummary(blockID, nil)
		if summary.TxCount != 0 {
			t.Errorf("TxCount = %d, want 0", summary.TxCount)
		}
		if len(summary.TxIDPrefixes) != 0 {
			t.Errorf("TxIDPrefixes len = %d, want 0", len(summary.TxIDPrefixes))
		}
	})

	t.Run("encode_format_block_id_first_48_bytes", func(t *testing.T) {
		// Encode() 前 48 字节必须等于 BlockID。
		raw := make([]byte, 48)
		for i := range raw {
			raw[i] = byte(i + 1)
		}
		blockID := types.MustBlockID(raw)
		summary := NewBlockSummary(blockID, nil)
		enc := summary.Encode()

		if len(enc) < 48 {
			t.Fatalf("encoded length = %d, want >= 48", len(enc))
		}
		if !bytes.Equal(enc[:48], raw) {
			t.Error("first 48 bytes of Encode() must equal BlockID")
		}
	})

	t.Run("encode_format_tx_count_as_varint", func(t *testing.T) {
		// Encode() 字节 48 开始是 varint(TxCount)。TxCount=1 → varint=0x01（单字节）。
		blockID := types.MustBlockID(make([]byte, 48))
		txid := types.MustTxID(make([]byte, 48))
		summary := NewBlockSummary(blockID, []types.TxID{txid})
		enc := summary.Encode()

		// 偏移 48：应为 0x01（varint(1)）。
		if len(enc) < 49 {
			t.Fatalf("encoded length = %d, want >= 49", len(enc))
		}
		if enc[48] != 0x01 {
			t.Errorf("enc[48] = 0x%02x, want 0x01 (varint(1))", enc[48])
		}
	})

	t.Run("encode_format_prefixes_follow_txcount", func(t *testing.T) {
		// varint(TxCount) 之后紧跟各前缀（16 字节 each，无额外长度前缀）。
		blockID := types.MustBlockID(make([]byte, 48))
		rawTx := make([]byte, 48)
		for i := range rawTx {
			rawTx[i] = byte(i + 1)
		}
		txid := types.MustTxID(rawTx)
		summary := NewBlockSummary(blockID, []types.TxID{txid})
		enc := summary.Encode()

		// 总长度应为 48（BlockID）+ 1（varint(1)）+ 16（前缀）= 65。
		want := 48 + 1 + TxIDPrefixLen
		if len(enc) != want {
			t.Errorf("encoded length = %d, want %d", len(enc), want)
		}
		// 偏移 49 开始的 16 字节应等于 TxID 的前 16 字节。
		prefix := enc[49:]
		for i, b := range prefix {
			if b != byte(i+1) {
				t.Errorf("prefix[%d] = 0x%02x, want 0x%02x", i, b, byte(i+1))
			}
		}
	})

	t.Run("encode_deterministic", func(t *testing.T) {
		// 同一区块概要多次编码结果必须一致。
		blockID := types.MustBlockID(make([]byte, 48))
		txids := make([]types.TxID, 3)
		for i := range txids {
			raw := make([]byte, 48)
			raw[0] = byte(i + 1)
			txids[i] = types.MustTxID(raw)
		}
		summary := NewBlockSummary(blockID, txids)
		enc1 := summary.Encode()
		enc2 := summary.Encode()
		if !bytes.Equal(enc1, enc2) {
			t.Error("Encode is not deterministic")
		}
	})

	t.Run("encode_multiple_txids_length", func(t *testing.T) {
		// 多个 TxID 时：总长 = 48 + varint(n) + n*16。
		blockID := types.MustBlockID(make([]byte, 48))
		cases := []struct {
			n    int
			want int // 假定 n < 128，varint(n) 为 1 字节
		}{
			{0, 48 + 1 + 0*TxIDPrefixLen},
			{1, 48 + 1 + 1*TxIDPrefixLen},
			{5, 48 + 1 + 5*TxIDPrefixLen},
			{10, 48 + 1 + 10*TxIDPrefixLen},
		}
		for _, tc := range cases {
			txIDs := make([]types.TxID, tc.n)
			for i := range txIDs {
				raw := make([]byte, 48)
				raw[0] = byte(i + 1)
				txIDs[i] = types.MustTxID(raw)
			}
			summary := NewBlockSummary(blockID, txIDs)
			enc := summary.Encode()
			if len(enc) != tc.want {
				t.Errorf("n=%d: encoded length = %d, want %d", tc.n, len(enc), tc.want)
			}
		}
	})

	t.Run("collision_fallback_carries_full_txid", func(t *testing.T) {
		// CollisionFallback 携带完整 48 字节 TxID（DEC-0602）。
		blockID := types.MustBlockID(make([]byte, 48))
		rawFull := make([]byte, 48)
		for i := range rawFull {
			rawFull[i] = byte(i + 0x80)
		}
		fullTxID := types.MustTxID(rawFull)

		fb := CollisionFallback{
			BlockID:  blockID,
			TxIndex:  7,
			FullTxID: fullTxID,
		}
		if fb.TxIndex != 7 {
			t.Errorf("TxIndex = %d, want 7", fb.TxIndex)
		}
		if fb.FullTxID != fullTxID {
			t.Error("FullTxID mismatch")
		}
		// 验证 FullTxID 为完整 48 字节（非截断前缀）。
		if len(fb.FullTxID.Bytes()) != 48 {
			t.Errorf("FullTxID length = %d, want 48", len(fb.FullTxID.Bytes()))
		}
	})

	t.Run("txid_prefix_shorter_than_full_txid", func(t *testing.T) {
		// TxIDPrefix（16 字节）严格短于完整 TxID（48 字节）。
		// 通过实际读取前缀字节来同时验证长度与内容。
		raw := make([]byte, 48)
		for i := range raw {
			raw[i] = byte(i + 1) // 1..48
		}
		txid := types.MustTxID(raw)
		prefix := NewTxIDPrefix(txid)

		// TxIDPrefixLen（16）必须严格小于完整 TxID 字节长度（48）。
		fullLen := len(txid.Bytes())
		if TxIDPrefixLen >= fullLen {
			t.Errorf("TxIDPrefixLen %d should be < full TxID len %d", TxIDPrefixLen, fullLen)
		}
		// 读取前缀各字节确认其内容为 TxID 的前 16 字节（使用 prefix 的值）。
		for i := 0; i < TxIDPrefixLen; i++ {
			if prefix[i] != raw[i] {
				t.Errorf("prefix[%d] = 0x%02x, want 0x%02x", i, prefix[i], raw[i])
			}
		}
	})
}
