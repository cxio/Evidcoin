# M5: utxo 模块设计

> **模块路径:** `internal/utxo`
> **依赖:** `pkg/crypto`, `pkg/types`, `internal/tx`
> **预估工时:** 3-4 天

## 概述

UTXO（Unspent Transaction Output）模块管理所有未花费的交易输出。UTXO 集是区块链状态的核心，用于验证交易输入的合法性。本模块还实现 UTXO 指纹计算，用于快速验证 UTXO 集的完整性。

## 功能清单

| 功能 | 文件 | 说明 |
|------|------|------|
| UTXO 条目 | `entry.go` | 单个 UTXO 条目定义 |
| UTXO 集合 | `set.go` | UTXO 集合管理 |
| UTXO 指纹 | `fingerprint.go` | 4层哈希树指纹计算 |
| 索引管理 | `index.go` | 年度/交易/输出索引 |
| 存储接口 | `store.go` | 持久化存储抽象 |

---

## 详细设计

### 1. entry.go - UTXO 条目

```go
package utxo

import (
	"encoding/binary"
	"errors"

	"evidcoin/internal/tx"
	"evidcoin/pkg/crypto"
	"evidcoin/pkg/types"
)

var (
	ErrEntryNotFound = errors.New("UTXO entry not found")
	ErrEntrySpent    = errors.New("UTXO entry already spent")
	ErrInvalidEntry  = errors.New("invalid UTXO entry")
)

// EntryType UTXO 条目类型
type EntryType byte

const (
	EntryTypeCoin       EntryType = 1 // 币金
	EntryTypeCredential EntryType = 2 // 凭信
)

// Entry UTXO 条目
type Entry struct {
	// 定位信息
	Year     uint16        // 交易年度
	TxID     types.Hash512 // 交易ID
	OutIndex uint16        // 输出序位

	// 输出信息
	Type       EntryType     // 条目类型
	Amount     types.Amount  // 金额（币金）或 0（凭信）
	Address    types.Address // 接收地址
	Script     []byte        // 锁定脚本
	
	// 元数据
	Height     uint64 // 所在区块高度
	Timestamp  int64  // 交易时间戳
	CoinDays   uint64 // 币权（币天数）
}

// Key 生成 UTXO 条目的唯一键
func (e *Entry) Key() EntryKey {
	return NewEntryKey(e.Year, e.TxID, e.OutIndex)
}

// Hash 计算条目哈希
func (e *Entry) Hash() types.Hash512 {
	return crypto.Hash512(e.Bytes())
}

// Bytes 序列化条目
func (e *Entry) Bytes() []byte {
	buf := make([]byte, 0, 256)

	// Year
	yearBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(yearBuf, e.Year)
	buf = append(buf, yearBuf...)

	// TxID
	buf = append(buf, e.TxID[:]...)

	// OutIndex
	outBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(outBuf, e.OutIndex)
	buf = append(buf, outBuf...)

	// Type
	buf = append(buf, byte(e.Type))

	// Amount
	buf = append(buf, types.EncodeVarint(int64(e.Amount))...)

	// Address
	buf = append(buf, e.Address[:]...)

	// Script
	scriptLen := make([]byte, 2)
	binary.BigEndian.PutUint16(scriptLen, uint16(len(e.Script)))
	buf = append(buf, scriptLen...)
	buf = append(buf, e.Script...)

	// Height
	heightBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBuf, e.Height)
	buf = append(buf, heightBuf...)

	// Timestamp
	tsBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBuf, uint64(e.Timestamp))
	buf = append(buf, tsBuf...)

	return buf
}

// CalcCoinDays 计算币权（币天数）
// coinDays = amount * days
func (e *Entry) CalcCoinDays(currentHeight uint64) uint64 {
	if e.Type != EntryTypeCoin {
		return 0
	}
	// 每天约 240 个区块（6分钟/块）
	blocksPerDay := uint64(240)
	days := (currentHeight - e.Height) / blocksPerDay
	return uint64(e.Amount) * days
}

// EntryKey UTXO 条目键
type EntryKey struct {
	Year     uint16
	TxID     types.Hash512
	OutIndex uint16
}

// NewEntryKey 创建条目键
func NewEntryKey(year uint16, txID types.Hash512, outIndex uint16) EntryKey {
	return EntryKey{
		Year:     year,
		TxID:     txID,
		OutIndex: outIndex,
	}
}

// Bytes 序列化键
func (k EntryKey) Bytes() []byte {
	buf := make([]byte, 2+64+2)
	binary.BigEndian.PutUint16(buf[0:2], k.Year)
	copy(buf[2:66], k.TxID[:])
	binary.BigEndian.PutUint16(buf[66:68], k.OutIndex)
	return buf
}

// ShortKey 短键（用于非首领输入匹配）
type ShortKey struct {
	Year      uint16
	TxIDShort types.ShortHash
	OutIndex  uint16
}

// NewShortKey 创建短键
func NewShortKey(year uint16, txIDShort types.ShortHash, outIndex uint16) ShortKey {
	return ShortKey{
		Year:      year,
		TxIDShort: txIDShort,
		OutIndex:  outIndex,
	}
}

// FromOutput 从交易输出创建 UTXO 条目
func FromOutput(txHeader *tx.TxHeader, output tx.Output, outIndex uint16, height uint64) *Entry {
	entry := &Entry{
		Year:      uint16(txHeader.Year()),
		TxID:      txHeader.ID(),
		OutIndex:  outIndex,
		Address:   output.Receiver(),
		Script:    output.LockScript(),
		Height:    height,
		Timestamp: txHeader.Timestamp,
	}

	switch o := output.(type) {
	case *tx.CoinOutput:
		entry.Type = EntryTypeCoin
		entry.Amount = o.Amount
	case *tx.CredentialOutput:
		entry.Type = EntryTypeCredential
		entry.Amount = 0
	}

	return entry
}
```

### 2. set.go - UTXO 集合

```go
package utxo

import (
	"sync"

	"evidcoin/pkg/types"
)

// Set UTXO 集合
type Set struct {
	mu      sync.RWMutex
	entries map[EntryKey]*Entry
	
	// 按年度索引
	byYear map[uint16]map[EntryKey]*Entry
	
	// 按地址索引
	byAddress map[types.Address]map[EntryKey]*Entry
	
	// 短键索引（用于快速查找）
	shortIndex map[ShortKey][]EntryKey
}

// NewSet 创建 UTXO 集合
func NewSet() *Set {
	return &Set{
		entries:    make(map[EntryKey]*Entry),
		byYear:     make(map[uint16]map[EntryKey]*Entry),
		byAddress:  make(map[types.Address]map[EntryKey]*Entry),
		shortIndex: make(map[ShortKey][]EntryKey),
	}
}

// Add 添加 UTXO 条目
func (s *Set) Add(entry *Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := entry.Key()
	s.entries[key] = entry

	// 更新年度索引
	if s.byYear[entry.Year] == nil {
		s.byYear[entry.Year] = make(map[EntryKey]*Entry)
	}
	s.byYear[entry.Year][key] = entry

	// 更新地址索引
	if s.byAddress[entry.Address] == nil {
		s.byAddress[entry.Address] = make(map[EntryKey]*Entry)
	}
	s.byAddress[entry.Address][key] = entry

	// 更新短键索引
	shortKey := NewShortKey(entry.Year, entry.TxID.Short(), entry.OutIndex)
	s.shortIndex[shortKey] = append(s.shortIndex[shortKey], key)
}

// Remove 移除 UTXO 条目
func (s *Set) Remove(key EntryKey) (*Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.entries[key]
	if !exists {
		return nil, false
	}

	delete(s.entries, key)

	// 更新年度索引
	if yearMap := s.byYear[entry.Year]; yearMap != nil {
		delete(yearMap, key)
		if len(yearMap) == 0 {
			delete(s.byYear, entry.Year)
		}
	}

	// 更新地址索引
	if addrMap := s.byAddress[entry.Address]; addrMap != nil {
		delete(addrMap, key)
		if len(addrMap) == 0 {
			delete(s.byAddress, entry.Address)
		}
	}

	// 更新短键索引
	shortKey := NewShortKey(entry.Year, entry.TxID.Short(), entry.OutIndex)
	if keys := s.shortIndex[shortKey]; keys != nil {
		for i, k := range keys {
			if k == key {
				s.shortIndex[shortKey] = append(keys[:i], keys[i+1:]...)
				break
			}
		}
		if len(s.shortIndex[shortKey]) == 0 {
			delete(s.shortIndex, shortKey)
		}
	}

	return entry, true
}

// Get 获取 UTXO 条目
func (s *Set) Get(key EntryKey) (*Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.entries[key]
	return entry, exists
}

// GetByShortKey 通过短键查找 UTXO
// 可能返回多个匹配（理论上极少）
func (s *Set) GetByShortKey(shortKey ShortKey) []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := s.shortIndex[shortKey]
	entries := make([]*Entry, 0, len(keys))
	for _, key := range keys {
		if entry, exists := s.entries[key]; exists {
			entries = append(entries, entry)
		}
	}
	return entries
}

// GetByAddress 获取地址的所有 UTXO
func (s *Set) GetByAddress(addr types.Address) []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	addrMap := s.byAddress[addr]
	entries := make([]*Entry, 0, len(addrMap))
	for _, entry := range addrMap {
		entries = append(entries, entry)
	}
	return entries
}

// GetByYear 获取指定年度的所有 UTXO
func (s *Set) GetByYear(year uint16) []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	yearMap := s.byYear[year]
	entries := make([]*Entry, 0, len(yearMap))
	for _, entry := range yearMap {
		entries = append(entries, entry)
	}
	return entries
}

// Years 获取所有有 UTXO 的年度
func (s *Set) Years() []uint16 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	years := make([]uint16, 0, len(s.byYear))
	for year := range s.byYear {
		years = append(years, year)
	}
	return years
}

// Size 获取 UTXO 总数
func (s *Set) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// TotalAmount 计算总金额
func (s *Set) TotalAmount() types.Amount {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var total types.Amount
	for _, entry := range s.entries {
		if entry.Type == EntryTypeCoin {
			total += entry.Amount
		}
	}
	return total
}

// Balance 获取地址余额
func (s *Set) Balance(addr types.Address) types.Amount {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var balance types.Amount
	for _, entry := range s.byAddress[addr] {
		if entry.Type == EntryTypeCoin {
			balance += entry.Amount
		}
	}
	return balance
}

// Clone 克隆 UTXO 集合
func (s *Set) Clone() *Set {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := NewSet()
	for _, entry := range s.entries {
		// 深拷贝条目
		entryCopy := *entry
		entryCopy.Script = make([]byte, len(entry.Script))
		copy(entryCopy.Script, entry.Script)
		clone.Add(&entryCopy)
	}
	return clone
}

// All 返回所有条目的迭代器
func (s *Set) All() []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]*Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		entries = append(entries, entry)
	}
	return entries
}
```

### 3. fingerprint.go - UTXO 指纹

```go
package utxo

import (
	"encoding/binary"
	"sort"

	"evidcoin/pkg/crypto"
	"evidcoin/pkg/types"
)

// Fingerprint UTXO 指纹
// 4层哈希树结构：年度 -> 交易 -> 输出 -> 条目
type Fingerprint struct {
	Root      types.Hash512            // 指纹根
	YearRoots map[uint16]types.Hash512 // 年度哈希根
}

// CalcFingerprint 计算 UTXO 集合的指纹
func CalcFingerprint(set *Set) *Fingerprint {
	fp := &Fingerprint{
		YearRoots: make(map[uint16]types.Hash512),
	}

	years := set.Years()
	if len(years) == 0 {
		return fp
	}

	// 排序年度
	sort.Slice(years, func(i, j int) bool { return years[i] < years[j] })

	// 计算每个年度的哈希根
	yearHashes := make([]types.Hash512, len(years))
	for i, year := range years {
		yearHash := calcYearHash(set, year)
		fp.YearRoots[year] = yearHash
		yearHashes[i] = yearHash
	}

	// 计算总根
	fp.Root = calcTreeRoot(yearHashes)
	return fp
}

// calcYearHash 计算单个年度的哈希
func calcYearHash(set *Set, year uint16) types.Hash512 {
	entries := set.GetByYear(year)
	if len(entries) == 0 {
		return types.Hash512{}
	}

	// 按交易ID分组
	txGroups := make(map[types.Hash512][]*Entry)
	for _, entry := range entries {
		txGroups[entry.TxID] = append(txGroups[entry.TxID], entry)
	}

	// 获取排序的交易ID列表
	txIDs := make([]types.Hash512, 0, len(txGroups))
	for txID := range txGroups {
		txIDs = append(txIDs, txID)
	}
	sort.Slice(txIDs, func(i, j int) bool {
		return compareHash512(txIDs[i], txIDs[j]) < 0
	})

	// 计算每个交易的哈希
	txHashes := make([]types.Hash512, len(txIDs))
	for i, txID := range txIDs {
		txHashes[i] = calcTxHash(txGroups[txID])
	}

	// 前置年度标识
	yearBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(yearBuf, year)

	// 计算年度树根
	treeRoot := calcTreeRoot(txHashes)
	combined := append(yearBuf, treeRoot[:]...)
	return crypto.Hash512(combined)
}

// calcTxHash 计算单个交易的 UTXO 哈希
func calcTxHash(entries []*Entry) types.Hash512 {
	if len(entries) == 0 {
		return types.Hash512{}
	}

	// 按输出序位排序
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].OutIndex < entries[j].OutIndex
	})

	// 计算每个输出的哈希
	outHashes := make([]types.Hash512, len(entries))
	for i, entry := range entries {
		outHashes[i] = entry.Hash()
	}

	return calcTreeRoot(outHashes)
}

// calcTreeRoot 计算哈希树根（类默克尔树）
func calcTreeRoot(hashes []types.Hash512) types.Hash512 {
	if len(hashes) == 0 {
		return types.Hash512{}
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	// 复制一份以避免修改原始切片
	level := make([]types.Hash512, len(hashes))
	copy(level, hashes)

	for len(level) > 1 {
		// 如果是奇数，复制最后一个
		if len(level)%2 != 0 {
			level = append(level, level[len(level)-1])
		}

		// 计算上一层
		parents := make([]types.Hash512, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			combined := make([]byte, 128)
			copy(combined[:64], level[i][:])
			copy(combined[64:], level[i+1][:])
			parents[i/2] = crypto.Hash512(combined)
		}
		level = parents
	}

	return level[0]
}

// compareHash512 比较两个哈希值
func compareHash512(a, b types.Hash512) int {
	for i := 0; i < 64; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// Verify 验证 UTXO 集合是否匹配指纹
func (fp *Fingerprint) Verify(set *Set) bool {
	computed := CalcFingerprint(set)
	return fp.Root == computed.Root
}

// VerifyYear 验证单个年度
func (fp *Fingerprint) VerifyYear(set *Set, year uint16) bool {
	expectedRoot, exists := fp.YearRoots[year]
	if !exists {
		// 如果指纹中没有该年度，集合中也不应该有
		entries := set.GetByYear(year)
		return len(entries) == 0
	}

	computedRoot := calcYearHash(set, year)
	return expectedRoot == computedRoot
}

// Bytes 序列化指纹
func (fp *Fingerprint) Bytes() []byte {
	buf := make([]byte, 0, 64+len(fp.YearRoots)*66)

	// Root
	buf = append(buf, fp.Root[:]...)

	// YearRoots count
	countBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(countBuf, uint16(len(fp.YearRoots)))
	buf = append(buf, countBuf...)

	// YearRoots (sorted)
	years := make([]uint16, 0, len(fp.YearRoots))
	for year := range fp.YearRoots {
		years = append(years, year)
	}
	sort.Slice(years, func(i, j int) bool { return years[i] < years[j] })

	for _, year := range years {
		yearBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(yearBuf, year)
		buf = append(buf, yearBuf...)
		buf = append(buf, fp.YearRoots[year][:]...)
	}

	return buf
}

// ParseFingerprint 解析指纹
func ParseFingerprint(data []byte) (*Fingerprint, error) {
	if len(data) < 66 {
		return nil, ErrInvalidEntry
	}

	fp := &Fingerprint{
		YearRoots: make(map[uint16]types.Hash512),
	}

	// Root
	copy(fp.Root[:], data[:64])

	// YearRoots count
	count := binary.BigEndian.Uint16(data[64:66])

	offset := 66
	for i := uint16(0); i < count; i++ {
		if offset+66 > len(data) {
			return nil, ErrInvalidEntry
		}
		year := binary.BigEndian.Uint16(data[offset : offset+2])
		var hash types.Hash512
		copy(hash[:], data[offset+2:offset+66])
		fp.YearRoots[year] = hash
		offset += 66
	}

	return fp, nil
}
```

### 4. index.go - 索引管理

```go
package utxo

import (
	"sync"

	"evidcoin/pkg/types"
)

// Index UTXO 索引管理器
type Index struct {
	mu sync.RWMutex

	// 交易ID到条目键的映射
	txIndex map[types.Hash512][]EntryKey

	// 短交易ID到完整交易ID的映射
	shortTxIndex map[types.ShortHash][]types.Hash512

	// 区块高度到条目键的映射（用于回滚）
	heightIndex map[uint64][]EntryKey
}

// NewIndex 创建索引管理器
func NewIndex() *Index {
	return &Index{
		txIndex:      make(map[types.Hash512][]EntryKey),
		shortTxIndex: make(map[types.ShortHash][]types.Hash512),
		heightIndex:  make(map[uint64][]EntryKey),
	}
}

// AddEntry 添加条目到索引
func (idx *Index) AddEntry(entry *Entry) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	key := entry.Key()

	// 交易索引
	idx.txIndex[entry.TxID] = append(idx.txIndex[entry.TxID], key)

	// 短交易ID索引
	shortID := entry.TxID.Short()
	found := false
	for _, txID := range idx.shortTxIndex[shortID] {
		if txID == entry.TxID {
			found = true
			break
		}
	}
	if !found {
		idx.shortTxIndex[shortID] = append(idx.shortTxIndex[shortID], entry.TxID)
	}

	// 高度索引
	idx.heightIndex[entry.Height] = append(idx.heightIndex[entry.Height], key)
}

// RemoveEntry 从索引移除条目
func (idx *Index) RemoveEntry(entry *Entry) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	key := entry.Key()

	// 交易索引
	if keys := idx.txIndex[entry.TxID]; keys != nil {
		for i, k := range keys {
			if k == key {
				idx.txIndex[entry.TxID] = append(keys[:i], keys[i+1:]...)
				break
			}
		}
		if len(idx.txIndex[entry.TxID]) == 0 {
			delete(idx.txIndex, entry.TxID)

			// 同时清理短交易ID索引
			shortID := entry.TxID.Short()
			if txIDs := idx.shortTxIndex[shortID]; txIDs != nil {
				for i, txID := range txIDs {
					if txID == entry.TxID {
						idx.shortTxIndex[shortID] = append(txIDs[:i], txIDs[i+1:]...)
						break
					}
				}
				if len(idx.shortTxIndex[shortID]) == 0 {
					delete(idx.shortTxIndex, shortID)
				}
			}
		}
	}

	// 高度索引
	if keys := idx.heightIndex[entry.Height]; keys != nil {
		for i, k := range keys {
			if k == key {
				idx.heightIndex[entry.Height] = append(keys[:i], keys[i+1:]...)
				break
			}
		}
		if len(idx.heightIndex[entry.Height]) == 0 {
			delete(idx.heightIndex, entry.Height)
		}
	}
}

// GetByTxID 通过交易ID获取所有条目键
func (idx *Index) GetByTxID(txID types.Hash512) []EntryKey {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	keys := idx.txIndex[txID]
	result := make([]EntryKey, len(keys))
	copy(result, keys)
	return result
}

// ResolvShortTxID 解析短交易ID
// 返回所有匹配的完整交易ID
func (idx *Index) ResolveShortTxID(shortID types.ShortHash) []types.Hash512 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	txIDs := idx.shortTxIndex[shortID]
	result := make([]types.Hash512, len(txIDs))
	copy(result, txIDs)
	return result
}

// GetByHeight 获取指定高度的所有条目键
func (idx *Index) GetByHeight(height uint64) []EntryKey {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	keys := idx.heightIndex[height]
	result := make([]EntryKey, len(keys))
	copy(result, keys)
	return result
}

// GetHeightRange 获取高度范围内的所有条目键
func (idx *Index) GetHeightRange(from, to uint64) []EntryKey {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var result []EntryKey
	for h := from; h <= to; h++ {
		result = append(result, idx.heightIndex[h]...)
	}
	return result
}
```

### 5. store.go - 存储接口

```go
package utxo

import (
	"evidcoin/pkg/types"
)

// Store UTXO 存储接口
type Store interface {
	// 基本操作
	Get(key EntryKey) (*Entry, error)
	Put(entry *Entry) error
	Delete(key EntryKey) error

	// 批量操作
	BatchPut(entries []*Entry) error
	BatchDelete(keys []EntryKey) error

	// 查询
	GetByTxID(txID types.Hash512) ([]*Entry, error)
	GetByAddress(addr types.Address) ([]*Entry, error)
	GetByYear(year uint16) ([]*Entry, error)

	// 指纹
	GetFingerprint() (*Fingerprint, error)
	SaveFingerprint(fp *Fingerprint) error

	// 迭代
	Iterator() EntryIterator

	// 事务
	Begin() (StoreTx, error)

	// 关闭
	Close() error
}

// StoreTx 存储事务
type StoreTx interface {
	Put(entry *Entry) error
	Delete(key EntryKey) error
	Commit() error
	Rollback() error
}

// EntryIterator 条目迭代器
type EntryIterator interface {
	Next() bool
	Entry() *Entry
	Error() error
	Close()
}

// MemoryStore 内存存储实现
type MemoryStore struct {
	set   *Set
	index *Index
	fp    *Fingerprint
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		set:   NewSet(),
		index: NewIndex(),
	}
}

func (s *MemoryStore) Get(key EntryKey) (*Entry, error) {
	entry, exists := s.set.Get(key)
	if !exists {
		return nil, ErrEntryNotFound
	}
	return entry, nil
}

func (s *MemoryStore) Put(entry *Entry) error {
	s.set.Add(entry)
	s.index.AddEntry(entry)
	return nil
}

func (s *MemoryStore) Delete(key EntryKey) error {
	entry, removed := s.set.Remove(key)
	if !removed {
		return ErrEntryNotFound
	}
	s.index.RemoveEntry(entry)
	return nil
}

func (s *MemoryStore) BatchPut(entries []*Entry) error {
	for _, entry := range entries {
		if err := s.Put(entry); err != nil {
			return err
		}
	}
	return nil
}

func (s *MemoryStore) BatchDelete(keys []EntryKey) error {
	for _, key := range keys {
		if err := s.Delete(key); err != nil {
			return err
		}
	}
	return nil
}

func (s *MemoryStore) GetByTxID(txID types.Hash512) ([]*Entry, error) {
	keys := s.index.GetByTxID(txID)
	entries := make([]*Entry, 0, len(keys))
	for _, key := range keys {
		if entry, exists := s.set.Get(key); exists {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (s *MemoryStore) GetByAddress(addr types.Address) ([]*Entry, error) {
	return s.set.GetByAddress(addr), nil
}

func (s *MemoryStore) GetByYear(year uint16) ([]*Entry, error) {
	return s.set.GetByYear(year), nil
}

func (s *MemoryStore) GetFingerprint() (*Fingerprint, error) {
	if s.fp == nil {
		s.fp = CalcFingerprint(s.set)
	}
	return s.fp, nil
}

func (s *MemoryStore) SaveFingerprint(fp *Fingerprint) error {
	s.fp = fp
	return nil
}

func (s *MemoryStore) Iterator() EntryIterator {
	return &memoryIterator{
		entries: s.set.All(),
		index:   -1,
	}
}

func (s *MemoryStore) Begin() (StoreTx, error) {
	return &memoryTx{store: s}, nil
}

func (s *MemoryStore) Close() error {
	return nil
}

type memoryIterator struct {
	entries []*Entry
	index   int
}

func (it *memoryIterator) Next() bool {
	it.index++
	return it.index < len(it.entries)
}

func (it *memoryIterator) Entry() *Entry {
	if it.index >= 0 && it.index < len(it.entries) {
		return it.entries[it.index]
	}
	return nil
}

func (it *memoryIterator) Error() error { return nil }
func (it *memoryIterator) Close()       {}

type memoryTx struct {
	store   *MemoryStore
	puts    []*Entry
	deletes []EntryKey
}

func (tx *memoryTx) Put(entry *Entry) error {
	tx.puts = append(tx.puts, entry)
	return nil
}

func (tx *memoryTx) Delete(key EntryKey) error {
	tx.deletes = append(tx.deletes, key)
	return nil
}

func (tx *memoryTx) Commit() error {
	for _, entry := range tx.puts {
		tx.store.Put(entry)
	}
	for _, key := range tx.deletes {
		tx.store.Delete(key)
	}
	// 指纹需要重新计算
	tx.store.fp = nil
	return nil
}

func (tx *memoryTx) Rollback() error {
	tx.puts = nil
	tx.deletes = nil
	return nil
}
```

---

## 测试用例

### utxo_test.go

```go
package utxo

import (
	"testing"

	"evidcoin/pkg/crypto"
	"evidcoin/pkg/types"
)

func TestEntry(t *testing.T) {
	txID := crypto.Hash512([]byte("test_tx"))
	addr := types.Address{}
	
	entry := &Entry{
		Year:      2024,
		TxID:      txID,
		OutIndex:  0,
		Type:      EntryTypeCoin,
		Amount:    100 * types.Bi,
		Address:   addr,
		Script:    []byte{0x01, 0x02},
		Height:    1000,
		Timestamp: 1700000000000,
	}

	// 测试键
	key := entry.Key()
	if key.Year != 2024 {
		t.Error("year mismatch")
	}
	if key.TxID != txID {
		t.Error("txID mismatch")
	}

	// 测试序列化
	data := entry.Bytes()
	if len(data) == 0 {
		t.Error("serialization failed")
	}

	// 测试哈希
	hash := entry.Hash()
	if hash == (types.Hash512{}) {
		t.Error("hash should not be empty")
	}
}

func TestSet(t *testing.T) {
	set := NewSet()

	txID := crypto.Hash512([]byte("test_tx"))
	addr := types.Address{}
	copy(addr[:], []byte("test_address"))

	entry := &Entry{
		Year:     2024,
		TxID:     txID,
		OutIndex: 0,
		Type:     EntryTypeCoin,
		Amount:   100 * types.Bi,
		Address:  addr,
		Height:   1000,
	}

	// 添加
	set.Add(entry)
	if set.Size() != 1 {
		t.Errorf("expected size 1, got %d", set.Size())
	}

	// 获取
	key := entry.Key()
	got, exists := set.Get(key)
	if !exists {
		t.Error("entry not found")
	}
	if got.Amount != entry.Amount {
		t.Error("amount mismatch")
	}

	// 按地址查询
	byAddr := set.GetByAddress(addr)
	if len(byAddr) != 1 {
		t.Errorf("expected 1 entry by address, got %d", len(byAddr))
	}

	// 余额
	balance := set.Balance(addr)
	if balance != 100*types.Bi {
		t.Errorf("expected balance 100 Bi, got %d", balance)
	}

	// 移除
	removed, ok := set.Remove(key)
	if !ok {
		t.Error("remove failed")
	}
	if removed.TxID != txID {
		t.Error("removed wrong entry")
	}
	if set.Size() != 0 {
		t.Error("set should be empty")
	}
}

func TestFingerprint(t *testing.T) {
	set := NewSet()

	// 添加几个条目
	for i := 0; i < 5; i++ {
		txID := crypto.Hash512([]byte{byte(i)})
		entry := &Entry{
			Year:     2024,
			TxID:     txID,
			OutIndex: uint16(i),
			Type:     EntryTypeCoin,
			Amount:   types.Amount(i+1) * types.Bi,
			Height:   1000,
		}
		set.Add(entry)
	}

	// 计算指纹
	fp := CalcFingerprint(set)
	if fp.Root == (types.Hash512{}) {
		t.Error("fingerprint root should not be empty")
	}

	// 验证
	if !fp.Verify(set) {
		t.Error("fingerprint verification failed")
	}

	// 修改集合后应该不匹配
	txID := crypto.Hash512([]byte{99})
	set.Add(&Entry{
		Year:     2024,
		TxID:     txID,
		OutIndex: 0,
		Type:     EntryTypeCoin,
		Amount:   1 * types.Bi,
		Height:   1001,
	})

	if fp.Verify(set) {
		t.Error("fingerprint should not match modified set")
	}
}

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()

	txID := crypto.Hash512([]byte("test"))
	entry := &Entry{
		Year:     2024,
		TxID:     txID,
		OutIndex: 0,
		Type:     EntryTypeCoin,
		Amount:   50 * types.Bi,
		Height:   500,
	}

	// Put
	if err := store.Put(entry); err != nil {
		t.Fatalf("put failed: %v", err)
	}

	// Get
	got, err := store.Get(entry.Key())
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Amount != entry.Amount {
		t.Error("amount mismatch")
	}

	// GetByTxID
	entries, err := store.GetByTxID(txID)
	if err != nil {
		t.Fatalf("getByTxID failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	// Delete
	if err := store.Delete(entry.Key()); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = store.Get(entry.Key())
	if err != ErrEntryNotFound {
		t.Error("expected ErrEntryNotFound")
	}
}

func TestCoinDays(t *testing.T) {
	entry := &Entry{
		Type:   EntryTypeCoin,
		Amount: 100 * types.Bi,
		Height: 0,
	}

	// 经过 240 个区块（约1天）
	coinDays := entry.CalcCoinDays(240)
	expected := uint64(100 * types.Bi * 1)
	if coinDays != expected {
		t.Errorf("expected %d coin days, got %d", expected, coinDays)
	}

	// 经过 2400 个区块（约10天）
	coinDays10 := entry.CalcCoinDays(2400)
	expected10 := uint64(100 * types.Bi * 10)
	if coinDays10 != expected10 {
		t.Errorf("expected %d coin days, got %d", expected10, coinDays10)
	}
}
```

---

## 实现步骤

### Step 1: 创建包结构

```bash
mkdir -p internal/utxo
touch internal/utxo/entry.go
touch internal/utxo/set.go
touch internal/utxo/fingerprint.go
touch internal/utxo/index.go
touch internal/utxo/store.go
touch internal/utxo/utxo_test.go
```

### Step 2: 按顺序实现

1. `entry.go` - UTXO 条目定义
2. `set.go` - UTXO 集合（内存）
3. `index.go` - 索引管理
4. `fingerprint.go` - 指纹计算
5. `store.go` - 存储抽象和内存实现

### Step 3: 测试验证

```bash
go test -v ./internal/utxo/...
```

---

## 注意事项

1. **线程安全**: Set 使用 RWMutex 保护并发访问
2. **指纹计算**: 4层哈希树结构（年度->交易->输出->条目）
3. **短ID索引**: 支持 20 字节短 ID 查找
4. **币权计算**: 基于区块高度差和金额
5. **存储抽象**: 便于后续替换为 LevelDB/RocksDB 等持久化存储
6. **回滚支持**: 通过高度索引支持区块回滚
