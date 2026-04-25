package utxo

import (
	"sort"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// FlagOutputs 表示一笔交易的输出花费状态位图。
// 每个 bit 对应一个输出，1=未花费，0=已花费。
type FlagOutputs struct {
	Count     int    // 有效输出数量
	FlagBytes []byte // 状态位（ceil(Count/8) 字节）
}

// appendVarint 本地 varint 编码（避免依赖 internal/tx）。
func appendVarint(buf []byte, v uint64) []byte {
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	return append(buf, byte(v))
}

// appendInt64BE 本地大端 int64 编码。
func appendInt64BE(buf []byte, v int64) []byte {
	uv := uint64(v)
	return append(buf,
		byte(uv>>56),
		byte(uv>>48),
		byte(uv>>40),
		byte(uv>>32),
		byte(uv>>24),
		byte(uv>>16),
		byte(uv>>8),
		byte(uv),
	)
}

// buildFlagOutputs 根据 entries（同一交易的所有输出）构造状态位图。
// entries 必须按 OutIndex 升序排列。
func buildFlagOutputs(entries []*UTXOEntry) FlagOutputs {
	if len(entries) == 0 {
		return FlagOutputs{}
	}
	maxIdx := 0
	for _, e := range entries {
		if e.OutIndex > maxIdx {
			maxIdx = e.OutIndex
		}
	}
	count := maxIdx + 1
	flagBytes := make([]byte, (count+7)/8)
	for _, e := range entries {
		if !e.Spent {
			flagBytes[e.OutIndex/8] |= 1 << (uint(e.OutIndex) % 8)
		}
	}
	return FlagOutputs{Count: count, FlagBytes: flagBytes}
}

// computeDataID 计算某 TxID 下所有输出项数据的哈希摘要。
// 输出项按 OutIndex 升序排列，序列化各项核心字段后整体 SHA3-384。
func computeDataID(entries []*UTXOEntry) types.Hash384 {
	// 按 OutIndex 升序排序，保证确定性
	sorted := make([]*UTXOEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].OutIndex < sorted[j].OutIndex
	})
	var buf []byte
	for _, e := range sorted {
		buf = appendVarint(buf, uint64(e.OutIndex))
		buf = appendInt64BE(buf, e.Amount)
		buf = append(buf, e.Address[:]...)
		buf = append(buf, e.LockScript...)
	}
	return crypto.SHA3_384(buf)
}

// infoHash 计算叶子节点哈希：SHA3-384(TxID ‖ DataID ‖ Count ‖ FlagBytes...)
func infoHash(txID types.Hash384, dataID types.Hash384, flags FlagOutputs) types.Hash384 {
	var buf []byte
	buf = append(buf, txID[:]...)
	buf = append(buf, dataID[:]...)
	buf = append(buf, byte(flags.Count))
	buf = append(buf, flags.FlagBytes...)
	return crypto.SHA3_384(buf)
}

// groupByTxID 将 entries 按 TxID 分组（同一 TxID 的 entry 归为一组）。
func groupByTxID(entries []*UTXOEntry) map[types.Hash384][]*UTXOEntry {
	groups := make(map[types.Hash384][]*UTXOEntry)
	for _, e := range entries {
		groups[e.TxID] = append(groups[e.TxID], e)
	}
	return groups
}

// ComputeFingerprint 对指定年度的 UTXO 集合（含已花费条目）计算指纹根哈希。
// 全量重算，适用于区块完成后触发；增量优化可在生产版本中扩展。
func ComputeFingerprint(store *UTXOStore, year int) types.Hash256 {
	// 收集该年度的所有 entry
	var yearEntries []*UTXOEntry
	for _, group := range store.entries {
		for _, e := range group {
			if e.TxYear == year {
				yearEntries = append(yearEntries, e)
			}
		}
	}
	if len(yearEntries) == 0 {
		return types.Hash256{}
	}
	// 按 TxID 分组，计算各叶子节点的 infoHash
	txGroups := groupByTxID(yearEntries)

	// 按 TxID[8]、TxID[13]、TxID[18] 三级分组
	tx8map := make(map[byte]map[byte]map[byte][]types.Hash384)
	for txID, entries := range txGroups {
		b8 := txID[8]
		b13 := txID[13]
		b18 := txID[18]
		flags := buildFlagOutputs(entries)
		dataID := computeDataID(entries)
		ih := infoHash(txID, dataID, flags)
		if tx8map[b8] == nil {
			tx8map[b8] = make(map[byte]map[byte][]types.Hash384)
		}
		if tx8map[b8][b13] == nil {
			tx8map[b8][b13] = make(map[byte][]types.Hash384)
		}
		tx8map[b8][b13][b18] = append(tx8map[b8][b13][b18], ih)
	}

	// 计算 Year 哈希（各层按索引升序排列，保证确定性）
	tx8Keys := make([]int, 0, len(tx8map))
	for k := range tx8map {
		tx8Keys = append(tx8Keys, int(k))
	}
	sort.Ints(tx8Keys)

	var tx8Hashes []byte
	for _, k8 := range tx8Keys {
		b8 := byte(k8)
		tx13map := tx8map[b8]
		tx13Keys := make([]int, 0, len(tx13map))
		for k := range tx13map {
			tx13Keys = append(tx13Keys, int(k))
		}
		sort.Ints(tx13Keys)

		var tx13Hashes []byte
		for _, k13 := range tx13Keys {
			b13 := byte(k13)
			tx18map := tx13map[b13]
			tx18Keys := make([]int, 0, len(tx18map))
			for k := range tx18map {
				tx18Keys = append(tx18Keys, int(k))
			}
			sort.Ints(tx18Keys)

			var tx18Hashes []byte
			for _, k18 := range tx18Keys {
				b18 := byte(k18)
				infoHashes := tx18map[b18]
				var leafBytes []byte
				for _, ih := range infoHashes {
					leafBytes = append(leafBytes, ih[:]...)
				}
				tx18Hash := crypto.BLAKE3_256(leafBytes)
				tx18Hashes = append(tx18Hashes, tx18Hash[:]...)
			}
			tx13Hash := crypto.BLAKE3_256(tx18Hashes)
			tx13Hashes = append(tx13Hashes, tx13Hash[:]...)
		}
		tx8Hash := crypto.BLAKE3_256(tx13Hashes)
		tx8Hashes = append(tx8Hashes, tx8Hash[:]...)
	}

	yearHash := crypto.BLAKE3_256(tx8Hashes)
	return yearHash
}

// ComputeRootFingerprint 对所有年度计算 UTXO 指纹根（跨年汇总）。
func ComputeRootFingerprint(store *UTXOStore) types.Hash256 {
	// 收集所有年度
	yearSet := make(map[int]struct{})
	for _, group := range store.entries {
		for _, e := range group {
			yearSet[e.TxYear] = struct{}{}
		}
	}
	years := make([]int, 0, len(yearSet))
	for y := range yearSet {
		years = append(years, y)
	}
	sort.Ints(years)

	var allYearHashes []byte
	for _, y := range years {
		yHash := ComputeFingerprint(store, y)
		allYearHashes = append(allYearHashes, yHash[:]...)
	}
	if len(allYearHashes) == 0 {
		return types.Hash256{}
	}
	return crypto.BLAKE3_256(allYearHashes)
}
