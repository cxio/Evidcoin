# M6: consensus 模块设计

> **模块路径:** `internal/consensus`
> **依赖:** `pkg/crypto`, `pkg/types`, `internal/tx`, `internal/utxo`
> **预估工时:** 4-5 天

## 概述

共识模块实现历史证明（Proof of Historical, PoH）共识机制。核心思想是利用历史交易ID作为铸造权竞争的基础，通过铸凭哈希计算和择优池管理来决定区块铸造者。本模块还包含铸币奖励计算、分叉处理等功能。

## 功能清单

| 功能 | 文件 | 说明 |
|------|------|------|
| 共识参数 | `params.go` | 共识相关常量和配置 |
| 铸凭哈希 | `mint_hash.go` | 铸凭哈希计算逻辑 |
| 择优凭证 | `credential.go` | 铸造候选者凭证 |
| 择优池 | `pool.go` | 择优池管理（预选/同步） |
| 铸币奖励 | `reward.go` | 区块奖励和交易费计算 |
| 分叉处理 | `fork.go` | 分叉检测与竞争 |
| 共识引擎 | `engine.go` | 共识核心逻辑 |

---

## 详细设计

### 1. params.go - 共识参数

```go
package consensus

import "time"

// 共识参数常量
const (
	// 评参区块偏移
	// 铸凭交易评选时，以链末端 -9 号区块为评参区块
	EvalBlockOffset = 9

	// UTXO指纹区块偏移
	// 取评参区块之前的 -24 号区块的 UTXO 指纹
	UTXOFingerprintOffset = 24

	// 铸凭交易区块范围
	// 有效范围: [-80000, -25]（约11个月）
	MintTxMinOffset = 25    // 最小偏移（最近）
	MintTxMaxOffset = 80000 // 最大偏移（最远，约11个月）

	// 择优池参数
	PoolSize        = 20 // 择优池容量
	PoolSyncAuth    = 15 // 有权同步的后段成员数（后15名）
	PoolSyncPeriod  = 2  // 同步期区块数
	PoolCollectTime = 6  // 广播收集期区块数

	// 分叉竞争
	ForkCompeteBlocks = 25 // 分叉竞争区块数

	// 区块时间
	BlockInterval = 6 * time.Minute // 出块间隔
	BlocksPerYear = 87661           // 每年区块数（恒星年）
	BlocksPerDay  = 240             // 每天区块数

	// 兑奖参数
	RewardConfirmBlocks = 48 // 兑奖评估区块数
	MinRewardConfirms   = 2  // 最低确认数
	EarlyClaimBlocks    = 25 // 提前兑奖区块数（满足确认后）

	// 初段规则
	InitialPhaseBlocks = 9 // 初段区块数（无评参区块）
)

// 铸币规则阶段
type MintPhase int

const (
	MintPhasePrelaunch MintPhase = iota // 预发布期（1-3年）
	MintPhaseNormal                     // 正式发行期
	MintPhaseLongTerm                   // 长期低通胀期
)

// MintSchedule 铸币时间表
type MintSchedule struct {
	StartHeight uint64    // 起始高度
	EndHeight   uint64    // 结束高度（0表示无限）
	AmountPerBlock uint64 // 每块铸币量（聪）
	Phase       MintPhase // 阶段
}

// 铸币时间表
// 预发布期：1-3年，分别 10/20/30 币/块
// 正式期：40币/块起，每2年递减20%，直到3币/块
// 长期：持续3币/块
var MintSchedules = []MintSchedule{
	// 预发布期
	{0, 87661, 10_0000_0000, MintPhasePrelaunch},           // 第1年：10币/块
	{87661, 175322, 20_0000_0000, MintPhasePrelaunch},      // 第2年：20币/块
	{175322, 262983, 30_0000_0000, MintPhasePrelaunch},     // 第3年：30币/块
	// 正式发行期（从第4年开始）
	{262983, 438305, 40_0000_0000, MintPhaseNormal},        // 年4-5：40币
	{438305, 613627, 32_0000_0000, MintPhaseNormal},        // 年6-7：32币
	{613627, 788949, 25_0000_0000, MintPhaseNormal},        // 年8-9：25币
	{788949, 964271, 20_0000_0000, MintPhaseNormal},        // 年10-11：20币
	{964271, 1139593, 16_0000_0000, MintPhaseNormal},       // 年12-13：16币
	{1139593, 1314915, 12_0000_0000, MintPhaseNormal},      // 年14-15：12币
	{1314915, 1490237, 9_0000_0000, MintPhaseNormal},       // 年16-17：9币
	{1490237, 1665559, 7_0000_0000, MintPhaseNormal},       // 年18-19：7币
	{1665559, 1840881, 5_0000_0000, MintPhaseNormal},       // 年20-21：5币
	{1840881, 2016203, 4_0000_0000, MintPhaseNormal},       // 年22-23：4币
	{2016203, 2191525, 3_0000_0000, MintPhaseNormal},       // 年24-25：3币
	// 长期低通胀
	{2191525, 0, 3_0000_0000, MintPhaseLongTerm},           // 26年+：持续3币
}

// 收益分配比例（百分比）
const (
	MinterSharePercent   = 50 // 铸造者总分成
	CheckTeamShare       = 40 // 校验组分成（铸造者的80%）
	MintCredentialShare  = 10 // 铸凭者分成（铸造者的20%）
	DepotsSharePercent   = 20 // depots 分成
	BlockqsSharePercent  = 20 // blockqs 分成
	Stun2pSharePercent   = 10 // stun2p 分成
	TxFeeBurnPercent     = 50 // 交易费销毁比例
)

// GetMintAmount 获取指定高度的铸币量
func GetMintAmount(height uint64) uint64 {
	for _, s := range MintSchedules {
		if height >= s.StartHeight && (s.EndHeight == 0 || height < s.EndHeight) {
			return s.AmountPerBlock
		}
	}
	// 默认返回长期铸币量
	return 3_0000_0000
}

// GetMintPhase 获取指定高度的铸币阶段
func GetMintPhase(height uint64) MintPhase {
	for _, s := range MintSchedules {
		if height >= s.StartHeight && (s.EndHeight == 0 || height < s.EndHeight) {
			return s.Phase
		}
	}
	return MintPhaseLongTerm
}
```

### 2. mint_hash.go - 铸凭哈希计算

```go
package consensus

import (
	"encoding/binary"
	"errors"

	"evidcoin/pkg/crypto"
	"evidcoin/pkg/types"
)

var (
	ErrInvalidMintTx      = errors.New("invalid mint transaction")
	ErrMintTxOutOfRange   = errors.New("mint transaction out of valid range")
	ErrInvalidEvalBlock   = errors.New("invalid evaluation block")
	ErrSignatureMismatch  = errors.New("signature mismatch")
)

// MintHashInput 铸凭哈希计算输入
type MintHashInput struct {
	// 铸凭交易ID
	MintTxID types.Hash512

	// 评参区块信息
	EvalBlockMintHash types.Hash512 // 评参区块的铸凭哈希
	UTXOFingerprint   types.Hash512 // -24号区块的UTXO指纹

	// 当前区块时间戳
	Timestamp int64
}

// CalcHashData 计算待签名的哈希数据
// hashData = Hash( 铸凭交易ID + 评参区块铸凭哈希 + UTXO指纹 + 时间戳 )
func (m *MintHashInput) CalcHashData() types.Hash512 {
	buf := make([]byte, 64+64+64+8)

	copy(buf[0:64], m.MintTxID[:])
	copy(buf[64:128], m.EvalBlockMintHash[:])
	copy(buf[128:192], m.UTXOFingerprint[:])
	binary.BigEndian.PutUint64(buf[192:200], uint64(m.Timestamp))

	return crypto.Hash512(buf)
}

// MintHash 铸凭哈希计算
// 1. hashData = Hash( 铸凭交易ID + 评参区块铸凭哈希 + UTXO指纹 + 时间戳 )
// 2. signData = Sign( hashData )
// 3. mintHash = Hash( signData )
type MintHash struct {
	Input     MintHashInput   // 输入数据
	HashData  types.Hash512   // 待签名哈希
	SignData  []byte          // 签名数据
	MintHash  types.Hash512   // 最终铸凭哈希
}

// CalcMintHash 计算铸凭哈希
func CalcMintHash(input MintHashInput, privateKey []byte) (*MintHash, error) {
	mh := &MintHash{
		Input: input,
	}

	// Step 1: 计算待签名哈希
	mh.HashData = input.CalcHashData()

	// Step 2: 签名
	signData, err := crypto.Sign(privateKey, mh.HashData[:])
	if err != nil {
		return nil, err
	}
	mh.SignData = signData

	// Step 3: 计算铸凭哈希
	mh.MintHash = crypto.Hash512(signData)

	return mh, nil
}

// VerifyMintHash 验证铸凭哈希
func VerifyMintHash(mh *MintHash, publicKey []byte) bool {
	// 验证签名
	if !crypto.Verify(publicKey, mh.HashData[:], mh.SignData) {
		return false
	}

	// 验证铸凭哈希
	expectedMintHash := crypto.Hash512(mh.SignData)
	return mh.MintHash == expectedMintHash
}

// CompareMintHash 比较两个铸凭哈希
// 返回: -1 (a优于b), 0 (相等), 1 (b优于a)
// 值小者优
func CompareMintHash(a, b types.Hash512) int {
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

// IsMintTxValid 检查交易是否为有效的铸凭交易
// 有效范围: [-80000, -25]
func IsMintTxValid(txHeight, currentHeight uint64) bool {
	// 初段规则：前9个区块任何交易都可以
	if currentHeight < InitialPhaseBlocks {
		return true
	}

	if txHeight >= currentHeight {
		return false
	}

	offset := currentHeight - txHeight
	return offset >= MintTxMinOffset && offset <= MintTxMaxOffset
}

// GetEvalBlockHeight 获取评参区块高度
func GetEvalBlockHeight(currentHeight uint64) uint64 {
	// 初段规则：前9个区块返回创始块
	if currentHeight < InitialPhaseBlocks {
		return 0
	}
	return currentHeight - EvalBlockOffset
}

// GetUTXOFingerprintBlockHeight 获取UTXO指纹区块高度
func GetUTXOFingerprintBlockHeight(currentHeight uint64) uint64 {
	evalHeight := GetEvalBlockHeight(currentHeight)
	if evalHeight < UTXOFingerprintOffset-EvalBlockOffset {
		return 0
	}
	return currentHeight - UTXOFingerprintOffset
}
```

### 3. credential.go - 择优凭证

```go
package consensus

import (
	"encoding/binary"
	"errors"

	"evidcoin/pkg/crypto"
	"evidcoin/pkg/types"
)

var (
	ErrInvalidCredential = errors.New("invalid mint credential")
	ErrCredentialExpired = errors.New("mint credential expired")
)

// Credential 择优凭证
// 铸造候选者的证明材料
type Credential struct {
	// 交易定位
	Year     uint16        // 交易年度
	TxID     types.Hash512 // 交易ID

	// 铸造者信息
	MinterPubKey []byte // 铸造者公钥（首笔输入的接收者）

	// 签名数据
	SignData []byte // 铸凭哈希签名（对 hashData 的签名）

	// 计算结果（不序列化，运行时计算）
	mintHash types.Hash512
}

// NewCredential 创建择优凭证
func NewCredential(year uint16, txID types.Hash512, pubKey []byte, signData []byte) *Credential {
	c := &Credential{
		Year:         year,
		TxID:         txID,
		MinterPubKey: pubKey,
		SignData:     signData,
	}
	c.mintHash = crypto.Hash512(signData)
	return c
}

// MintHash 获取铸凭哈希
func (c *Credential) MintHash() types.Hash512 {
	if c.mintHash == (types.Hash512{}) {
		c.mintHash = crypto.Hash512(c.SignData)
	}
	return c.mintHash
}

// Verify 验证凭证
func (c *Credential) Verify(hashData types.Hash512) bool {
	return crypto.Verify(c.MinterPubKey, hashData[:], c.SignData)
}

// Bytes 序列化凭证
func (c *Credential) Bytes() []byte {
	// 2 + 64 + 2 + pubKey + 2 + signData
	pubLen := len(c.MinterPubKey)
	signLen := len(c.SignData)
	buf := make([]byte, 2+64+2+pubLen+2+signLen)

	offset := 0
	
	// Year
	binary.BigEndian.PutUint16(buf[offset:], c.Year)
	offset += 2

	// TxID
	copy(buf[offset:], c.TxID[:])
	offset += 64

	// MinterPubKey
	binary.BigEndian.PutUint16(buf[offset:], uint16(pubLen))
	offset += 2
	copy(buf[offset:], c.MinterPubKey)
	offset += pubLen

	// SignData
	binary.BigEndian.PutUint16(buf[offset:], uint16(signLen))
	offset += 2
	copy(buf[offset:], c.SignData)

	return buf
}

// ParseCredential 解析凭证
func ParseCredential(data []byte) (*Credential, error) {
	if len(data) < 70 { // 最小长度 2+64+2+2
		return nil, ErrInvalidCredential
	}

	c := &Credential{}
	offset := 0

	// Year
	c.Year = binary.BigEndian.Uint16(data[offset:])
	offset += 2

	// TxID
	copy(c.TxID[:], data[offset:offset+64])
	offset += 64

	// MinterPubKey
	pubLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2
	if offset+pubLen > len(data) {
		return nil, ErrInvalidCredential
	}
	c.MinterPubKey = make([]byte, pubLen)
	copy(c.MinterPubKey, data[offset:offset+pubLen])
	offset += pubLen

	// SignData
	if offset+2 > len(data) {
		return nil, ErrInvalidCredential
	}
	signLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2
	if offset+signLen > len(data) {
		return nil, ErrInvalidCredential
	}
	c.SignData = make([]byte, signLen)
	copy(c.SignData, data[offset:offset+signLen])

	// 计算铸凭哈希
	c.mintHash = crypto.Hash512(c.SignData)

	return c, nil
}

// MinterAddress 获取铸造者地址
func (c *Credential) MinterAddress() types.Address {
	return crypto.PubKeyToAddress(c.MinterPubKey)
}
```

### 4. pool.go - 择优池管理

```go
package consensus

import (
	"sort"
	"sync"

	"evidcoin/pkg/types"
)

// PoolEntry 择优池条目
type PoolEntry struct {
	Credential *Credential   // 择优凭证
	MintHash   types.Hash512 // 铸凭哈希（缓存）
}

// Pool 择优池
// 每个评参区块对应一个择优池
type Pool struct {
	mu sync.RWMutex

	// 评参区块高度
	EvalHeight uint64

	// 候选者列表（按铸凭哈希排序，值小者优）
	entries []*PoolEntry

	// 已同步标记
	syncedFrom map[types.Address]bool
}

// NewPool 创建择优池
func NewPool(evalHeight uint64) *Pool {
	return &Pool{
		EvalHeight: evalHeight,
		entries:    make([]*PoolEntry, 0, PoolSize),
		syncedFrom: make(map[types.Address]bool),
	}
}

// Add 尝试添加候选者
// 返回是否成功添加（比池中最差的好则添加）
func (p *Pool) Add(cred *Credential) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	mintHash := cred.MintHash()
	entry := &PoolEntry{
		Credential: cred,
		MintHash:   mintHash,
	}

	// 如果池未满，直接添加
	if len(p.entries) < PoolSize {
		p.entries = append(p.entries, entry)
		p.sort()
		return true
	}

	// 池已满，比较最差的
	worst := p.entries[len(p.entries)-1]
	if CompareMintHash(mintHash, worst.MintHash) >= 0 {
		// 不比最差的好，拒绝
		return false
	}

	// 替换最差的
	p.entries[len(p.entries)-1] = entry
	p.sort()
	return true
}

// sort 内部排序（值小者优）
func (p *Pool) sort() {
	sort.Slice(p.entries, func(i, j int) bool {
		return CompareMintHash(p.entries[i].MintHash, p.entries[j].MintHash) < 0
	})
}

// Best 获取最优候选者
func (p *Pool) Best() *PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.entries) == 0 {
		return nil
	}
	return p.entries[0]
}

// Get 获取指定排名的候选者（0-based）
func (p *Pool) Get(rank int) *PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if rank < 0 || rank >= len(p.entries) {
		return nil
	}
	return p.entries[rank]
}

// Rank 获取凭证的排名（0-based，-1表示不在池中）
func (p *Pool) Rank(cred *Credential) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	mintHash := cred.MintHash()
	for i, e := range p.entries {
		if e.MintHash == mintHash {
			return i
		}
	}
	return -1
}

// Size 获取池大小
func (p *Pool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.entries)
}

// All 获取所有条目
func (p *Pool) All() []*PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*PoolEntry, len(p.entries))
	copy(result, p.entries)
	return result
}

// CanSync 检查地址是否有权同步
// 后15名有权同步
func (p *Pool) CanSync(addr types.Address) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.entries) <= PoolSize-PoolSyncAuth {
		return false
	}

	// 检查是否在后15名中
	startIdx := len(p.entries) - PoolSyncAuth
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(p.entries); i++ {
		if p.entries[i].Credential.MinterAddress() == addr {
			return true
		}
	}
	return false
}

// HasSyncedFrom 检查是否已从该地址同步
func (p *Pool) HasSyncedFrom(addr types.Address) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.syncedFrom[addr]
}

// MarkSynced 标记已同步
func (p *Pool) MarkSynced(addr types.Address) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.syncedFrom[addr] = true
}

// Merge 合并另一个池
func (p *Pool) Merge(other *Pool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, entry := range other.All() {
		// 内部添加逻辑（无锁版本）
		p.addUnlocked(entry)
	}
}

func (p *Pool) addUnlocked(entry *PoolEntry) bool {
	if len(p.entries) < PoolSize {
		p.entries = append(p.entries, entry)
		p.sort()
		return true
	}

	worst := p.entries[len(p.entries)-1]
	if CompareMintHash(entry.MintHash, worst.MintHash) >= 0 {
		return false
	}

	p.entries[len(p.entries)-1] = entry
	p.sort()
	return true
}

// PoolSet 择优池集合
// 管理多个评参区块的择优池
type PoolSet struct {
	mu    sync.RWMutex
	pools map[uint64]*Pool
}

// NewPoolSet 创建择优池集合
func NewPoolSet() *PoolSet {
	return &PoolSet{
		pools: make(map[uint64]*Pool),
	}
}

// GetOrCreate 获取或创建择优池
func (ps *PoolSet) GetOrCreate(evalHeight uint64) *Pool {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if pool, exists := ps.pools[evalHeight]; exists {
		return pool
	}

	pool := NewPool(evalHeight)
	ps.pools[evalHeight] = pool
	return pool
}

// Get 获取择优池
func (ps *PoolSet) Get(evalHeight uint64) *Pool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.pools[evalHeight]
}

// Remove 移除择优池
func (ps *PoolSet) Remove(evalHeight uint64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.pools, evalHeight)
}

// Cleanup 清理过期的择优池
func (ps *PoolSet) Cleanup(currentHeight uint64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// 保留最近的若干个池
	keepFrom := uint64(0)
	if currentHeight > EvalBlockOffset+10 {
		keepFrom = currentHeight - EvalBlockOffset - 10
	}

	for height := range ps.pools {
		if height < keepFrom {
			delete(ps.pools, height)
		}
	}
}
```

### 5. reward.go - 铸币奖励

```go
package consensus

import (
	"evidcoin/pkg/types"
)

// BlockReward 区块奖励
type BlockReward struct {
	// 总收益
	MintAmount    types.Amount // 原始铸币
	TxFeeTotal    types.Amount // 交易费总额
	RetainedReward types.Amount // 兑奖截留（前期未兑奖部分）
	TotalRevenue  types.Amount // 总收益

	// 分配
	TxFeeBurned   types.Amount // 销毁的交易费
	TxFeeRecovered types.Amount // 回收的交易费

	// 收益分配
	CheckTeamReward    types.Amount // 校验组收益
	MintCredReward     types.Amount // 铸凭者收益
	DepotsReward       types.Amount // depots 奖励
	BlockqsReward      types.Amount // blockqs 奖励
	Stun2pReward       types.Amount // stun2p 奖励
}

// CalcBlockReward 计算区块奖励
func CalcBlockReward(height uint64, txFeeTotal, retainedReward types.Amount) *BlockReward {
	r := &BlockReward{
		MintAmount:     types.Amount(GetMintAmount(height)),
		TxFeeTotal:     txFeeTotal,
		RetainedReward: retainedReward,
	}

	// 交易费处理：50%销毁，50%回收
	r.TxFeeBurned = txFeeTotal * TxFeeBurnPercent / 100
	r.TxFeeRecovered = txFeeTotal - r.TxFeeBurned

	// 总收益 = 铸币 + 回收的交易费 + 截留
	r.TotalRevenue = r.MintAmount + r.TxFeeRecovered + retainedReward

	// 分配计算
	// 铸造者：50%（校验组40% + 铸凭者10%）
	minterTotal := r.TotalRevenue * MinterSharePercent / 100
	r.CheckTeamReward = minterTotal * CheckTeamShare / MinterSharePercent
	r.MintCredReward = minterTotal * MintCredentialShare / MinterSharePercent

	// 公共服务：50%
	r.DepotsReward = r.TotalRevenue * DepotsSharePercent / 100
	r.BlockqsReward = r.TotalRevenue * BlockqsSharePercent / 100
	r.Stun2pReward = r.TotalRevenue * Stun2pSharePercent / 100

	return r
}

// RewardConfirmSlot 兑奖确认槽
// 每位铸造者需要对前48个区块的奖励目标进行评估
type RewardConfirmSlot struct {
	// 48个区块，每块3个服务地址，共144位
	// 使用18字节（144 bits）表示
	Confirms [18]byte
}

// SetConfirm 设置确认位
// blockOffset: 0-47, serviceIndex: 0-2
func (s *RewardConfirmSlot) SetConfirm(blockOffset, serviceIndex int) {
	if blockOffset < 0 || blockOffset >= RewardConfirmBlocks {
		return
	}
	if serviceIndex < 0 || serviceIndex >= 3 {
		return
	}

	bitPos := blockOffset*3 + serviceIndex
	bytePos := bitPos / 8
	bitOffset := bitPos % 8
	s.Confirms[bytePos] |= 1 << bitOffset
}

// GetConfirm 获取确认位
func (s *RewardConfirmSlot) GetConfirm(blockOffset, serviceIndex int) bool {
	if blockOffset < 0 || blockOffset >= RewardConfirmBlocks {
		return false
	}
	if serviceIndex < 0 || serviceIndex >= 3 {
		return false
	}

	bitPos := blockOffset*3 + serviceIndex
	bytePos := bitPos / 8
	bitOffset := bitPos % 8
	return (s.Confirms[bytePos] & (1 << bitOffset)) != 0
}

// PendingReward 待兑奖项
type PendingReward struct {
	Height       uint64        // 区块高度
	ServiceType  int           // 服务类型（0:depots, 1:blockqs, 2:stun2p）
	Address      types.Address // 服务地址
	Amount       types.Amount  // 奖励金额
	Confirms     int           // 确认数
	FullyClaimed bool          // 是否已完全兑奖
}

// CanClaim 检查是否可以兑奖
func (pr *PendingReward) CanClaim(currentHeight uint64) bool {
	if pr.FullyClaimed {
		return false
	}

	// 已满足确认要求且过了25个区块
	if pr.Confirms >= MinRewardConfirms {
		return currentHeight >= pr.Height+EarlyClaimBlocks
	}

	// 48个区块后可以部分兑奖
	return currentHeight >= pr.Height+RewardConfirmBlocks
}

// ClaimAmount 计算可兑奖金额
func (pr *PendingReward) ClaimAmount() types.Amount {
	if pr.FullyClaimed {
		return 0
	}

	switch pr.Confirms {
	case 0:
		return 0
	case 1:
		return pr.Amount * 50 / 100
	default:
		return pr.Amount
	}
}

// RetainedAmount 计算截留金额
func (pr *PendingReward) RetainedAmount() types.Amount {
	return pr.Amount - pr.ClaimAmount()
}
```

### 6. fork.go - 分叉处理

```go
package consensus

import (
	"sync"

	"evidcoin/pkg/types"
)

// ForkState 分叉状态
type ForkState int

const (
	ForkStateNone       ForkState = iota // 无分叉
	ForkStateCompeting                   // 竞争中
	ForkStateResolved                    // 已解决
)

// ForkBranch 分叉分支
type ForkBranch struct {
	// 分叉起点
	ForkHeight uint64        // 分叉高度
	ForkBlock  types.Hash512 // 分叉前的共同区块

	// 分支信息
	TipHeight  uint64        // 当前顶端高度
	TipBlock   types.Hash512 // 顶端区块ID
	TotalStake uint64        // 累计币权销毁

	// 分支区块列表（从分叉点开始）
	BlockIDs []types.Hash512
}

// Fork 分叉信息
type Fork struct {
	mu sync.RWMutex

	State      ForkState    // 分叉状态
	ForkHeight uint64       // 分叉高度
	Main       *ForkBranch  // 主分支
	Candidate  *ForkBranch  // 候选分支
}

// NewFork 创建分叉
func NewFork(forkHeight uint64, mainBlock, candidateBlock types.Hash512) *Fork {
	return &Fork{
		State:      ForkStateCompeting,
		ForkHeight: forkHeight,
		Main: &ForkBranch{
			ForkHeight: forkHeight,
			TipHeight:  forkHeight,
			TipBlock:   mainBlock,
			BlockIDs:   []types.Hash512{mainBlock},
		},
		Candidate: &ForkBranch{
			ForkHeight: forkHeight,
			TipHeight:  forkHeight,
			TipBlock:   candidateBlock,
			BlockIDs:   []types.Hash512{candidateBlock},
		},
	}
}

// AddMainBlock 添加主分支区块
func (f *Fork) AddMainBlock(blockID types.Hash512, stake uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.Main.TipHeight++
	f.Main.TipBlock = blockID
	f.Main.TotalStake += stake
	f.Main.BlockIDs = append(f.Main.BlockIDs, blockID)
}

// AddCandidateBlock 添加候选分支区块
func (f *Fork) AddCandidateBlock(blockID types.Hash512, stake uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.Candidate.TipHeight++
	f.Candidate.TipBlock = blockID
	f.Candidate.TotalStake += stake
	f.Candidate.BlockIDs = append(f.Candidate.BlockIDs, blockID)
}

// ShouldSwitch 检查是否应该切换到候选分支
// 规则：候选区块的币权销毁量3倍于主区块则胜出
func (f *Fork) ShouldSwitch() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// 候选分支币权是主分支的3倍以上
	return f.Candidate.TotalStake >= f.Main.TotalStake*3
}

// IsCompetitionOver 检查竞争是否结束
func (f *Fork) IsCompetitionOver(currentHeight uint64) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// 超过25个区块竞争结束
	return currentHeight >= f.ForkHeight+ForkCompeteBlocks
}

// Winner 获取胜出分支
func (f *Fork) Winner() *ForkBranch {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.ShouldSwitch() {
		return f.Candidate
	}
	return f.Main
}

// ForkManager 分叉管理器
type ForkManager struct {
	mu    sync.RWMutex
	forks map[uint64]*Fork // key: 分叉高度
}

// NewForkManager 创建分叉管理器
func NewForkManager() *ForkManager {
	return &ForkManager{
		forks: make(map[uint64]*Fork),
	}
}

// AddFork 添加分叉
func (fm *ForkManager) AddFork(fork *Fork) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.forks[fork.ForkHeight] = fork
}

// GetFork 获取分叉
func (fm *ForkManager) GetFork(height uint64) *Fork {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.forks[height]
}

// HasActiveFork 检查是否有活跃分叉
func (fm *ForkManager) HasActiveFork() bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	for _, fork := range fm.forks {
		if fork.State == ForkStateCompeting {
			return true
		}
	}
	return false
}

// ResolveCompletedForks 解决已完成的分叉
func (fm *ForkManager) ResolveCompletedForks(currentHeight uint64) []*Fork {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	var resolved []*Fork
	for height, fork := range fm.forks {
		if fork.State == ForkStateCompeting && fork.IsCompetitionOver(currentHeight) {
			fork.State = ForkStateResolved
			resolved = append(resolved, fork)
		}

		// 清理过老的分叉记录
		if currentHeight > height+ForkCompeteBlocks*2 {
			delete(fm.forks, height)
		}
	}
	return resolved
}

// DetectFork 检测分叉
// 当收到与当前链不同的区块时调用
func DetectFork(currentTip, newBlock types.Hash512, height uint64) *Fork {
	if currentTip == newBlock {
		return nil
	}
	return NewFork(height, currentTip, newBlock)
}
```

### 7. engine.go - 共识引擎

```go
package consensus

import (
	"context"
	"errors"
	"sync"
	"time"

	"evidcoin/pkg/types"
)

var (
	ErrNotMinter        = errors.New("not a valid minter")
	ErrPoolNotReady     = errors.New("preference pool not ready")
	ErrBlockNotReady    = errors.New("block not ready for minting")
)

// BlockInfo 区块信息接口
type BlockInfo interface {
	Height() uint64
	ID() types.Hash512
	Timestamp() int64
	MintHash() types.Hash512
	UTXOFingerprint() types.Hash512
	Stakes() uint64
}

// Engine 共识引擎
type Engine struct {
	mu sync.RWMutex

	// 择优池集合
	poolSet *PoolSet

	// 分叉管理
	forkMgr *ForkManager

	// 当前链顶信息
	tipHeight uint64
	tipBlock  types.Hash512

	// 回调接口
	blockProvider BlockProvider
}

// BlockProvider 区块提供者接口
type BlockProvider interface {
	GetBlockByHeight(height uint64) (BlockInfo, error)
	GetCurrentHeight() uint64
	GetCurrentTip() types.Hash512
}

// NewEngine 创建共识引擎
func NewEngine(provider BlockProvider) *Engine {
	return &Engine{
		poolSet:       NewPoolSet(),
		forkMgr:       NewForkManager(),
		blockProvider: provider,
	}
}

// UpdateTip 更新链顶信息
func (e *Engine) UpdateTip(height uint64, blockID types.Hash512) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tipHeight = height
	e.tipBlock = blockID
}

// SubmitCredential 提交择优凭证
func (e *Engine) SubmitCredential(cred *Credential) (bool, error) {
	e.mu.RLock()
	currentHeight := e.tipHeight
	e.mu.RUnlock()

	// 计算评参区块高度
	evalHeight := GetEvalBlockHeight(currentHeight + 1)

	// 获取或创建择优池
	pool := e.poolSet.GetOrCreate(evalHeight)

	// 添加到择优池
	added := pool.Add(cred)
	return added, nil
}

// GetBestMinter 获取最优铸造者
func (e *Engine) GetBestMinter(targetHeight uint64) (*Credential, error) {
	evalHeight := GetEvalBlockHeight(targetHeight)
	pool := e.poolSet.Get(evalHeight)
	if pool == nil {
		return nil, ErrPoolNotReady
	}

	best := pool.Best()
	if best == nil {
		return nil, ErrNotMinter
	}

	return best.Credential, nil
}

// GetMinterRank 获取凭证的铸造排名
func (e *Engine) GetMinterRank(cred *Credential, targetHeight uint64) int {
	evalHeight := GetEvalBlockHeight(targetHeight)
	pool := e.poolSet.Get(evalHeight)
	if pool == nil {
		return -1
	}
	return pool.Rank(cred)
}

// CanMint 检查是否可以铸造
func (e *Engine) CanMint(cred *Credential, targetHeight uint64) (bool, time.Duration) {
	rank := e.GetMinterRank(cred, targetHeight)
	if rank < 0 {
		return false, 0
	}

	// 计算等待时间
	// 排名0：等待30秒
	// 排名n：等待 30 + n*15 秒
	waitTime := MintDelayFirst + time.Duration(rank)*MintDelayStep

	// 检查当前时间是否已到
	targetTime := e.getBlockTargetTime(targetHeight)
	now := time.Now()

	readyTime := targetTime.Add(waitTime)
	if now.Before(readyTime) {
		return false, readyTime.Sub(now)
	}

	return true, 0
}

func (e *Engine) getBlockTargetTime(height uint64) time.Time {
	// 基于创世时间和区块高度计算目标时间
	// 这里简化处理，实际需要从创世块获取基准时间
	genesisTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return genesisTime.Add(time.Duration(height) * BlockInterval)
}

// CalcMintHashInput 计算铸凭哈希输入
func (e *Engine) CalcMintHashInput(mintTxID types.Hash512, targetHeight uint64) (*MintHashInput, error) {
	evalHeight := GetEvalBlockHeight(targetHeight)
	utxoHeight := GetUTXOFingerprintBlockHeight(targetHeight)

	// 获取评参区块
	evalBlock, err := e.blockProvider.GetBlockByHeight(evalHeight)
	if err != nil {
		return nil, err
	}

	// 获取UTXO指纹区块
	utxoBlock, err := e.blockProvider.GetBlockByHeight(utxoHeight)
	if err != nil {
		return nil, err
	}

	// 计算目标时间戳
	targetTime := e.getBlockTargetTime(targetHeight)

	return &MintHashInput{
		MintTxID:          mintTxID,
		EvalBlockMintHash: evalBlock.MintHash(),
		UTXOFingerprint:   utxoBlock.UTXOFingerprint(),
		Timestamp:         targetTime.UnixNano(),
	}, nil
}

// HandleNewBlock 处理新区块
func (e *Engine) HandleNewBlock(block BlockInfo) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	height := block.Height()
	blockID := block.ID()

	// 检测分叉
	if height == e.tipHeight+1 && blockID != e.tipBlock {
		// 可能是分叉
		fork := DetectFork(e.tipBlock, blockID, height)
		if fork != nil {
			e.forkMgr.AddFork(fork)
		}
	}

	// 更新链顶
	e.tipHeight = height
	e.tipBlock = blockID

	// 清理过期的择优池
	e.poolSet.Cleanup(height)

	// 解决已完成的分叉
	e.forkMgr.ResolveCompletedForks(height)

	return nil
}

// SyncPool 同步择优池
func (e *Engine) SyncPool(evalHeight uint64, remotePool *Pool, signer types.Address) error {
	localPool := e.poolSet.Get(evalHeight)
	if localPool == nil {
		return ErrPoolNotReady
	}

	// 检查签名者是否有权同步
	if !localPool.CanSync(signer) {
		return ErrNotMinter
	}

	// 检查是否已同步
	if localPool.HasSyncedFrom(signer) {
		return nil // 已同步，忽略
	}

	// 合并池
	localPool.Merge(remotePool)
	localPool.MarkSynced(signer)

	return nil
}

// Run 运行共识引擎
func (e *Engine) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// 周期性清理和维护
			e.mu.Lock()
			e.poolSet.Cleanup(e.tipHeight)
			e.forkMgr.ResolveCompletedForks(e.tipHeight)
			e.mu.Unlock()
		}
	}
}
```

---

## 测试用例

### consensus_test.go

```go
package consensus

import (
	"testing"

	"evidcoin/pkg/crypto"
	"evidcoin/pkg/types"
)

func TestGetMintAmount(t *testing.T) {
	tests := []struct {
		height   uint64
		expected uint64
	}{
		{0, 10_0000_0000},      // 第1年
		{50000, 10_0000_0000},  // 第1年
		{87661, 20_0000_0000},  // 第2年开始
		{175322, 30_0000_0000}, // 第3年开始
		{262983, 40_0000_0000}, // 第4年开始
		{3000000, 3_0000_0000}, // 长期
	}

	for _, tt := range tests {
		got := GetMintAmount(tt.height)
		if got != tt.expected {
			t.Errorf("GetMintAmount(%d) = %d, want %d", tt.height, got, tt.expected)
		}
	}
}

func TestMintHashInput(t *testing.T) {
	input := MintHashInput{
		MintTxID:          crypto.Hash512([]byte("test_tx")),
		EvalBlockMintHash: crypto.Hash512([]byte("eval_mint")),
		UTXOFingerprint:   crypto.Hash512([]byte("utxo_fp")),
		Timestamp:         1700000000000,
	}

	hashData := input.CalcHashData()
	if hashData == (types.Hash512{}) {
		t.Error("CalcHashData should not return empty hash")
	}
}

func TestCompareMintHash(t *testing.T) {
	a := types.Hash512{}
	b := types.Hash512{}
	a[0] = 1
	b[0] = 2

	if CompareMintHash(a, b) != -1 {
		t.Error("a should be less than b")
	}
	if CompareMintHash(b, a) != 1 {
		t.Error("b should be greater than a")
	}
	if CompareMintHash(a, a) != 0 {
		t.Error("a should equal a")
	}
}

func TestIsMintTxValid(t *testing.T) {
	tests := []struct {
		txHeight      uint64
		currentHeight uint64
		expected      bool
	}{
		{0, 5, true},        // 初段
		{0, 100, true},      // 有效范围内
		{50, 100, true},     // 有效范围内
		{80, 100, false},    // 太近（offset=20 < 25）
		{0, 80100, false},   // 太远（offset=80100 > 80000）
	}

	for _, tt := range tests {
		got := IsMintTxValid(tt.txHeight, tt.currentHeight)
		if got != tt.expected {
			t.Errorf("IsMintTxValid(%d, %d) = %v, want %v",
				tt.txHeight, tt.currentHeight, got, tt.expected)
		}
	}
}

func TestPool(t *testing.T) {
	pool := NewPool(100)

	// 创建凭证
	for i := 0; i < 25; i++ {
		pubKey := make([]byte, 32)
		pubKey[0] = byte(i)
		signData := make([]byte, 64)
		signData[0] = byte(255 - i) // 值越小越优

		cred := NewCredential(2024, crypto.Hash512(pubKey), pubKey, signData)
		pool.Add(cred)
	}

	// 池应该只保留20个
	if pool.Size() != PoolSize {
		t.Errorf("pool size = %d, want %d", pool.Size(), PoolSize)
	}

	// 最优应该是signData[0]=255-24=231的那个
	best := pool.Best()
	if best == nil {
		t.Fatal("best should not be nil")
	}
}

func TestBlockReward(t *testing.T) {
	// 测试区块奖励计算
	reward := CalcBlockReward(1000, 8_0000_0000, 0)

	// 铸币量应该是10币（第1年）
	if reward.MintAmount != 10_0000_0000 {
		t.Errorf("MintAmount = %d, want %d", reward.MintAmount, 10_0000_0000)
	}

	// 交易费50%销毁
	if reward.TxFeeBurned != 4_0000_0000 {
		t.Errorf("TxFeeBurned = %d, want %d", reward.TxFeeBurned, 4_0000_0000)
	}

	// 总收益 = 铸币 + 回收交易费
	expectedTotal := types.Amount(10_0000_0000 + 4_0000_0000)
	if reward.TotalRevenue != expectedTotal {
		t.Errorf("TotalRevenue = %d, want %d", reward.TotalRevenue, expectedTotal)
	}
}

func TestRewardConfirmSlot(t *testing.T) {
	slot := RewardConfirmSlot{}

	// 设置确认
	slot.SetConfirm(0, 0)
	slot.SetConfirm(10, 1)
	slot.SetConfirm(47, 2)

	// 检查确认
	if !slot.GetConfirm(0, 0) {
		t.Error("should be confirmed at (0, 0)")
	}
	if !slot.GetConfirm(10, 1) {
		t.Error("should be confirmed at (10, 1)")
	}
	if !slot.GetConfirm(47, 2) {
		t.Error("should be confirmed at (47, 2)")
	}
	if slot.GetConfirm(1, 0) {
		t.Error("should not be confirmed at (1, 0)")
	}
}

func TestFork(t *testing.T) {
	mainBlock := crypto.Hash512([]byte("main"))
	candBlock := crypto.Hash512([]byte("candidate"))

	fork := NewFork(100, mainBlock, candBlock)

	if fork.State != ForkStateCompeting {
		t.Error("fork should be competing")
	}

	// 添加区块
	fork.AddMainBlock(crypto.Hash512([]byte("main2")), 1000)
	fork.AddCandidateBlock(crypto.Hash512([]byte("cand2")), 3001)

	// 候选分支币权3倍于主分支，应该切换
	if !fork.ShouldSwitch() {
		t.Error("should switch to candidate")
	}

	// 竞争未结束
	if fork.IsCompetitionOver(110) {
		t.Error("competition should not be over at height 110")
	}

	// 竞争结束
	if !fork.IsCompetitionOver(126) {
		t.Error("competition should be over at height 126")
	}
}
```

---

## 实现步骤

### Step 1: 创建包结构

```bash
mkdir -p internal/consensus
touch internal/consensus/params.go
touch internal/consensus/mint_hash.go
touch internal/consensus/credential.go
touch internal/consensus/pool.go
touch internal/consensus/reward.go
touch internal/consensus/fork.go
touch internal/consensus/engine.go
touch internal/consensus/consensus_test.go
```

### Step 2: 按顺序实现

1. `params.go` - 共识参数和铸币时间表
2. `mint_hash.go` - 铸凭哈希计算
3. `credential.go` - 择优凭证
4. `pool.go` - 择优池管理
5. `reward.go` - 奖励计算
6. `fork.go` - 分叉处理
7. `engine.go` - 共识引擎

### Step 3: 测试验证

```bash
go test -v ./internal/consensus/...
```

---

## 注意事项

1. **初段规则**: 前9个区块没有评参区块，需要特殊处理
2. **铸凭哈希签名**: 攻击者无法预知签名结果，因此无法判断哪个铸凭交易更优
3. **择优池同步**: 后15名有权同步，每个授权节点只有一次同步权力
4. **交易费销毁**: 50%销毁形成通缩，平衡长期通胀
5. **分叉竞争**: 候选区块币权3倍于主区块才能胜出
6. **兑奖截留**: 未满足确认的奖励会被截留到第49号区块回收

## 关键算法

### 铸凭哈希计算流程

```
1. hashData = Hash( mintTxID + evalBlockMintHash + utxoFingerprint + timestamp )
2. signData = Sign( privateKey, hashData )
3. mintHash = Hash( signData )
```

### 择优池同步流程

```
1. 新区块创建后，6个区块时段收集广播
2. 成为-7号区块后，进入同步期（2个区块时段）
3. 成为-9号区块后，择优池确定，开始铸造竞争
```

### 分叉竞争规则

```
1. 25个区块内持续竞争
2. 候选区块币权销毁量3倍于主区块则胜出
3. 25个区块后按币权销毁量决定胜负
```
