# M4: tx 模块设计

> **模块路径:** `internal/tx`
> **依赖:** `pkg/crypto`, `pkg/types`
> **预估工时:** 4-5 天

## 概述

交易模块实现 Evidcoin 的交易结构，支持三种基本信元：币金、凭信、存证。交易是信用表达和转移的封装，包含输入和输出两个主要部分。

## 功能清单

| 功能 | 文件 | 说明 |
|------|------|------|
| 交易头 | `header.go` | 版本、时间戳、数据体哈希 |
| 输入项 | `input.go` | UTXO 引用、解锁数据 |
| 输出项 | `output.go` | 币金/凭信/存证输出 |
| 交易体 | `tx.go` | 完整交易结构和验证 |
| Coinbase | `coinbase.go` | 铸币交易 |
| 附件ID | `attachment.go` | 附件标识结构 |
| 签名授权 | `sigflag.go` | 签名消息约束类型 |
| 序列化 | `serialize.go` | 交易编解码 |

---

## 详细设计

### 1. header.go - 交易头

```go
package tx

import (
	"encoding/binary"
	"errors"
	"time"

	"evidcoin/pkg/crypto"
	"evidcoin/pkg/types"
)

var (
	ErrInvalidVersion   = errors.New("invalid transaction version")
	ErrInvalidTimestamp = errors.New("invalid transaction timestamp")
)

// TxHeader 交易头
// 交易头的哈希即为交易ID
type TxHeader struct {
	Version   uint16       // 版本号
	Timestamp int64        // 交易时间戳（Unix 毫秒）
	HashBody  types.Hash512 // 数据体哈希
}

// HeaderSize 交易头固定大小
const HeaderSize = 2 + 8 + 64 // 74 bytes

// ID 计算交易ID
func (h *TxHeader) ID() types.Hash512 {
	return crypto.Hash512(h.Bytes())
}

// ShortID 返回短交易ID（用于非首领输入引用）
func (h *TxHeader) ShortID() types.ShortHash {
	id := h.ID()
	var short types.ShortHash
	copy(short[:], id[:20])
	return short
}

// Year 获取交易年度（按时间戳计算）
func (h *TxHeader) Year() int {
	t := time.UnixMilli(h.Timestamp).UTC()
	return t.Year()
}

// Bytes 序列化交易头
func (h *TxHeader) Bytes() []byte {
	buf := make([]byte, HeaderSize)
	binary.BigEndian.PutUint16(buf[0:2], h.Version)
	binary.BigEndian.PutUint64(buf[2:10], uint64(h.Timestamp))
	copy(buf[10:], h.HashBody[:])
	return buf
}

// ParseHeader 解析交易头
func ParseHeader(data []byte) (*TxHeader, error) {
	if len(data) < HeaderSize {
		return nil, errors.New("insufficient header data")
	}
	h := &TxHeader{
		Version:   binary.BigEndian.Uint16(data[0:2]),
		Timestamp: int64(binary.BigEndian.Uint64(data[2:10])),
	}
	copy(h.HashBody[:], data[10:74])
	return h, nil
}

// Validate 验证交易头
func (h *TxHeader) Validate() error {
	if h.Version == 0 {
		return ErrInvalidVersion
	}
	if h.Timestamp <= 0 {
		return ErrInvalidTimestamp
	}
	return nil
}
```

### 2. input.go - 输入项

```go
package tx

import (
	"encoding/binary"
	"errors"

	"evidcoin/pkg/crypto"
	"evidcoin/pkg/types"
)

var (
	ErrInvalidInputRef = errors.New("invalid input reference")
	ErrInputNotFound   = errors.New("input UTXO not found")
)

// Input 交易输入项
type Input struct {
	Year       uint16        // 来源交易年度
	TxID       types.Hash512 // 来源交易ID（首领输入用全长64字节）
	TxIDShort  types.ShortHash // 来源交易ID（其他输入用20字节）
	OutIndex   uint16        // 来源输出序位
	IsLeader   bool          // 是否为首领输入
	UnlockData []byte        // 解锁数据（脚本）
}

// CredentialInput 凭信输入项（额外字段）
type CredentialInput struct {
	Input
	TransferIndex uint16 // 转出序位（本交易输出集中）
}

// LeaderInputSize 首领输入固定部分大小
const LeaderInputSize = 2 + 64 + 2 // Year + TxID + OutIndex

// RestInputSize 其他输入固定部分大小
const RestInputSize = 2 + 20 + 2 // Year + ShortTxID + OutIndex

// NewLeaderInput 创建首领输入
func NewLeaderInput(year uint16, txID types.Hash512, outIndex uint16) *Input {
	return &Input{
		Year:     year,
		TxID:     txID,
		OutIndex: outIndex,
		IsLeader: true,
	}
}

// NewRestInput 创建其他输入
func NewRestInput(year uint16, txIDShort types.ShortHash, outIndex uint16) *Input {
	return &Input{
		Year:      year,
		TxIDShort: txIDShort,
		OutIndex:  outIndex,
		IsLeader:  false,
	}
}

// Hash 计算输入项哈希
func (in *Input) Hash() types.Hash512 {
	if in.IsLeader {
		return in.leaderHash()
	}
	return in.restHash()
}

func (in *Input) leaderHash() types.Hash512 {
	buf := make([]byte, LeaderInputSize)
	binary.BigEndian.PutUint16(buf[0:2], in.Year)
	copy(buf[2:66], in.TxID[:])
	binary.BigEndian.PutUint16(buf[66:68], in.OutIndex)
	return crypto.Hash512(buf)
}

func (in *Input) restHash() types.Hash512 {
	buf := make([]byte, RestInputSize)
	binary.BigEndian.PutUint16(buf[0:2], in.Year)
	copy(buf[2:22], in.TxIDShort[:])
	binary.BigEndian.PutUint16(buf[22:24], in.OutIndex)
	return crypto.Hash512(buf)
}

// Bytes 序列化输入项（不含解锁数据）
func (in *Input) Bytes() []byte {
	if in.IsLeader {
		buf := make([]byte, LeaderInputSize)
		binary.BigEndian.PutUint16(buf[0:2], in.Year)
		copy(buf[2:66], in.TxID[:])
		binary.BigEndian.PutUint16(buf[66:68], in.OutIndex)
		return buf
	}
	buf := make([]byte, RestInputSize)
	binary.BigEndian.PutUint16(buf[0:2], in.Year)
	copy(buf[2:22], in.TxIDShort[:])
	binary.BigEndian.PutUint16(buf[22:24], in.OutIndex)
	return buf
}

// CalcInputHash 计算输入项集合哈希
// InputHash = Hash512(LeadHash + RestHash)
func CalcInputHash(inputs []*Input) types.Hash512 {
	if len(inputs) == 0 {
		return types.Hash512{}
	}

	// 首领输入哈希
	leadHash := inputs[0].Hash()

	// 其余输入哈希
	var restData []byte
	for i := 1; i < len(inputs); i++ {
		restData = append(restData, inputs[i].Bytes()...)
	}
	restHash := crypto.Hash512(restData)

	// 合并计算
	combined := make([]byte, 128)
	copy(combined[:64], leadHash[:])
	copy(combined[64:], restHash[:])
	return crypto.Hash512(combined)
}
```

### 3. output.go - 输出项

```go
package tx

import (
	"encoding/binary"
	"errors"

	"evidcoin/pkg/types"
)

// OutputType 输出类型
type OutputType byte

const (
	OutputTypeReserved   OutputType = 0 // 预留
	OutputTypeCoin       OutputType = 1 // 币金
	OutputTypeCredential OutputType = 2 // 凭信
	OutputTypeEvidence   OutputType = 3 // 存证
	OutputTypeMediator   OutputType = 4 // 介管脚本
)

// OutputConfig 输出配置（1字节）
type OutputConfig byte

const (
	OutputConfigCustomClass OutputConfig = 1 << 7 // 自定义类
	OutputConfigAttachment  OutputConfig = 1 << 6 // 包含附件
	OutputConfigDestroy     OutputConfig = 1 << 5 // 销毁标记
)

// GetType 获取输出类型
func (c OutputConfig) GetType() OutputType {
	return OutputType(c & 0x0F)
}

// HasAttachment 是否包含附件
func (c OutputConfig) HasAttachment() bool {
	return c&OutputConfigAttachment != 0
}

// IsDestroy 是否销毁
func (c OutputConfig) IsDestroy() bool {
	return c&OutputConfigDestroy != 0
}

// Output 输出项基础接口
type Output interface {
	Type() OutputType
	Config() OutputConfig
	Receiver() types.Address
	LockScript() []byte
	Bytes() []byte
}

// CoinOutput 币金输出
type CoinOutput struct {
	Serial     uint16        // 输出序位
	config     OutputConfig  // 配置
	Amount     types.Amount  // 金额
	Address    types.Address // 接收地址
	Remarks    []byte        // 附言（<256字节）
	Script     []byte        // 锁定脚本
}

// NewCoinOutput 创建币金输出
func NewCoinOutput(serial uint16, amount types.Amount, addr types.Address, script []byte) *CoinOutput {
	return &CoinOutput{
		Serial:  serial,
		config:  OutputConfig(OutputTypeCoin),
		Amount:  amount,
		Address: addr,
		Script:  script,
	}
}

func (o *CoinOutput) Type() OutputType       { return OutputTypeCoin }
func (o *CoinOutput) Config() OutputConfig   { return o.config }
func (o *CoinOutput) Receiver() types.Address { return o.Address }
func (o *CoinOutput) LockScript() []byte     { return o.Script }

func (o *CoinOutput) Bytes() []byte {
	// Serial(2) + Config(1) + Amount(varint) + Address(48) + RemarksLen(1) + Remarks + ScriptLen(2) + Script
	buf := make([]byte, 0, 128)
	
	// Serial
	serialBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(serialBuf, o.Serial)
	buf = append(buf, serialBuf...)
	
	// Config
	buf = append(buf, byte(o.config))
	
	// Amount (varint)
	buf = append(buf, types.EncodeVarint(int64(o.Amount))...)
	
	// Address
	buf = append(buf, o.Address[:]...)
	
	// Remarks
	buf = append(buf, byte(len(o.Remarks)))
	buf = append(buf, o.Remarks...)
	
	// Script
	scriptLen := make([]byte, 2)
	binary.BigEndian.PutUint16(scriptLen, uint16(len(o.Script)))
	buf = append(buf, scriptLen...)
	buf = append(buf, o.Script...)
	
	return buf
}

// CredentialConfig 凭信配置（2字节）
type CredentialConfig uint16

const (
	CredConfigNew        CredentialConfig = 1 << 15 // 新建标记
	CredConfigModifiable CredentialConfig = 1 << 14 // 可否修改
	CredConfigModified   CredentialConfig = 1 << 13 // 是否修改
	CredConfigTransfer   CredentialConfig = 1 << 11 // 转移计次
	CredConfigExpire     CredentialConfig = 1 << 10 // 有效期限
)

// DescriptionLength 获取描述长度
func (c CredentialConfig) DescriptionLength() int {
	return int(c & 0x03FF) // 低10位
}

// CredentialOutput 凭信输出
type CredentialOutput struct {
	Serial        uint16           // 输出序位
	config        OutputConfig     // 基础配置
	CredConfig    CredentialConfig // 凭信配置
	Address       types.Address    // 接收地址
	Creator       []byte           // 创建者（<256字节）
	Title         []byte           // 标题（<256字节）
	Description   []byte           // 描述（<1KB）
	TransferCount uint16           // 转移次数（可选）
	ExpireTime    uint32           // 有效期（秒，可选）
	AttachmentID  []byte           // 附件ID（可选）
	Script        []byte           // 锁定脚本
}

// NewCredentialOutput 创建凭信输出
func NewCredentialOutput(serial uint16, addr types.Address, creator, title, desc []byte) *CredentialOutput {
	credConfig := CredConfigNew | CredentialConfig(len(desc)&0x03FF)
	return &CredentialOutput{
		Serial:      serial,
		config:      OutputConfig(OutputTypeCredential),
		CredConfig:  credConfig,
		Address:     addr,
		Creator:     creator,
		Title:       title,
		Description: desc,
	}
}

func (o *CredentialOutput) Type() OutputType       { return OutputTypeCredential }
func (o *CredentialOutput) Config() OutputConfig   { return o.config }
func (o *CredentialOutput) Receiver() types.Address { return o.Address }
func (o *CredentialOutput) LockScript() []byte     { return o.Script }

// IsNew 是否为新建凭信
func (o *CredentialOutput) IsNew() bool {
	return o.CredConfig&CredConfigNew != 0
}

// IsModifiable 是否可修改
func (o *CredentialOutput) IsModifiable() bool {
	return o.CredConfig&CredConfigModifiable != 0
}

func (o *CredentialOutput) Bytes() []byte {
	// 实现序列化...
	buf := make([]byte, 0, 512)
	
	// Serial
	serialBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(serialBuf, o.Serial)
	buf = append(buf, serialBuf...)
	
	// Config
	buf = append(buf, byte(o.config))
	
	// CredConfig
	credBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(credBuf, uint16(o.CredConfig))
	buf = append(buf, credBuf...)
	
	// Address
	buf = append(buf, o.Address[:]...)
	
	// Creator
	buf = append(buf, byte(len(o.Creator)))
	buf = append(buf, o.Creator...)
	
	// Title
	buf = append(buf, byte(len(o.Title)))
	buf = append(buf, o.Title...)
	
	// Description（长度已在 CredConfig 中）
	buf = append(buf, o.Description...)
	
	// TransferCount（如果有）
	if o.CredConfig&CredConfigTransfer != 0 {
		tcBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(tcBuf, o.TransferCount)
		buf = append(buf, tcBuf...)
	}
	
	// ExpireTime（如果有）
	if o.CredConfig&CredConfigExpire != 0 {
		etBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(etBuf, o.ExpireTime)
		buf = append(buf, etBuf...)
	}
	
	// AttachmentID（如果有）
	if o.config.HasAttachment() {
		buf = append(buf, byte(len(o.AttachmentID)))
		buf = append(buf, o.AttachmentID...)
	}
	
	// Script
	scriptLen := make([]byte, 2)
	binary.BigEndian.PutUint16(scriptLen, uint16(len(o.Script)))
	buf = append(buf, scriptLen...)
	buf = append(buf, o.Script...)
	
	return buf
}

// EvidenceConfig 存证配置（2字节）
type EvidenceConfig uint16

// ContentLength 获取内容长度
func (c EvidenceConfig) ContentLength() int {
	return int(c & 0x07FF) // 低11位，最大2KB
}

// EvidenceOutput 存证输出
type EvidenceOutput struct {
	Serial       uint16         // 输出序位
	config       OutputConfig   // 基础配置
	EvidConfig   EvidenceConfig // 存证配置
	Creator      []byte         // 创建者（<256字节）
	Title        []byte         // 标题（<256字节）
	Content      []byte         // 内容（<2KB）
	AttachmentID []byte         // 附件ID（可选）
	Script       []byte         // 识别脚本
}

// NewEvidenceOutput 创建存证输出
func NewEvidenceOutput(serial uint16, creator, title, content []byte) *EvidenceOutput {
	evidConfig := EvidenceConfig(len(content) & 0x07FF)
	return &EvidenceOutput{
		Serial:     serial,
		config:     OutputConfig(OutputTypeEvidence),
		EvidConfig: evidConfig,
		Creator:    creator,
		Title:      title,
		Content:    content,
	}
}

func (o *EvidenceOutput) Type() OutputType       { return OutputTypeEvidence }
func (o *EvidenceOutput) Config() OutputConfig   { return o.config }
func (o *EvidenceOutput) Receiver() types.Address { return types.Address{} } // 存证无接收者
func (o *EvidenceOutput) LockScript() []byte     { return o.Script }

func (o *EvidenceOutput) Bytes() []byte {
	buf := make([]byte, 0, 512)
	
	// Serial
	serialBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(serialBuf, o.Serial)
	buf = append(buf, serialBuf...)
	
	// Config
	buf = append(buf, byte(o.config))
	
	// EvidConfig
	evidBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(evidBuf, uint16(o.EvidConfig))
	buf = append(buf, evidBuf...)
	
	// Creator
	buf = append(buf, byte(len(o.Creator)))
	buf = append(buf, o.Creator...)
	
	// Title
	buf = append(buf, byte(len(o.Title)))
	buf = append(buf, o.Title...)
	
	// Content（长度已在 EvidConfig 中）
	buf = append(buf, o.Content...)
	
	// AttachmentID
	if o.config.HasAttachment() {
		buf = append(buf, byte(len(o.AttachmentID)))
		buf = append(buf, o.AttachmentID...)
	}
	
	// Script
	scriptLen := make([]byte, 2)
	binary.BigEndian.PutUint16(scriptLen, uint16(len(o.Script)))
	buf = append(buf, scriptLen...)
	buf = append(buf, o.Script...)
	
	return buf
}
```

### 4. tx.go - 完整交易

```go
package tx

import (
	"errors"

	"evidcoin/pkg/crypto"
	"evidcoin/pkg/types"
)

var (
	ErrEmptyInputs     = errors.New("transaction has no inputs")
	ErrEmptyOutputs    = errors.New("transaction has no outputs")
	ErrTxTooLarge      = errors.New("transaction exceeds size limit")
	ErrInvalidBodyHash = errors.New("body hash mismatch")
)

// MaxTxSize 单笔交易最大尺寸（不含解锁数据）
const MaxTxSize = 8192

// Transaction 完整交易
type Transaction struct {
	Header  *TxHeader
	Inputs  []*Input
	Outputs []Output
}

// NewTransaction 创建交易
func NewTransaction(version uint16, timestamp int64) *Transaction {
	return &Transaction{
		Header: &TxHeader{
			Version:   version,
			Timestamp: timestamp,
		},
		Inputs:  make([]*Input, 0),
		Outputs: make([]Output, 0),
	}
}

// AddInput 添加输入项
func (tx *Transaction) AddInput(input *Input) {
	if len(tx.Inputs) == 0 {
		input.IsLeader = true
	}
	tx.Inputs = append(tx.Inputs, input)
}

// AddOutput 添加输出项
func (tx *Transaction) AddOutput(output Output) {
	tx.Outputs = append(tx.Outputs, output)
}

// ID 获取交易ID
func (tx *Transaction) ID() types.Hash512 {
	return tx.Header.ID()
}

// Year 获取交易年度
func (tx *Transaction) Year() int {
	return tx.Header.Year()
}

// CalcBodyHash 计算数据体哈希
// HashBody = Hash512(InputHash + OutputHash)
func (tx *Transaction) CalcBodyHash() types.Hash512 {
	inputHash := CalcInputHash(tx.Inputs)
	outputHash := tx.CalcOutputHash()

	combined := make([]byte, 128)
	copy(combined[:64], inputHash[:])
	copy(combined[64:], outputHash[:])
	return crypto.Hash512(combined)
}

// CalcOutputHash 计算输出哈希树根
func (tx *Transaction) CalcOutputHash() types.Hash512 {
	if len(tx.Outputs) == 0 {
		return types.Hash512{}
	}

	// 构建叶子节点
	leaves := make([]types.Hash512, len(tx.Outputs))
	for i, out := range tx.Outputs {
		leaves[i] = crypto.Hash512(out.Bytes())
	}

	// 计算哈希树根
	return calcMerkleRoot(leaves)
}

// calcMerkleRoot 计算默克尔树根
func calcMerkleRoot(leaves []types.Hash512) types.Hash512 {
	if len(leaves) == 0 {
		return types.Hash512{}
	}
	if len(leaves) == 1 {
		return leaves[0]
	}

	// 如果是奇数，复制最后一个
	if len(leaves)%2 != 0 {
		leaves = append(leaves, leaves[len(leaves)-1])
	}

	// 计算上一层
	parents := make([]types.Hash512, len(leaves)/2)
	for i := 0; i < len(leaves); i += 2 {
		combined := make([]byte, 128)
		copy(combined[:64], leaves[i][:])
		copy(combined[64:], leaves[i+1][:])
		parents[i/2] = crypto.Hash512(combined)
	}

	return calcMerkleRoot(parents)
}

// Finalize 完成交易构建，计算哈希
func (tx *Transaction) Finalize() error {
	if err := tx.validateBasic(); err != nil {
		return err
	}
	tx.Header.HashBody = tx.CalcBodyHash()
	return nil
}

// Validate 验证交易
func (tx *Transaction) Validate() error {
	if err := tx.validateBasic(); err != nil {
		return err
	}

	// 验证数据体哈希
	expectedHash := tx.CalcBodyHash()
	if tx.Header.HashBody != expectedHash {
		return ErrInvalidBodyHash
	}

	return tx.Header.Validate()
}

func (tx *Transaction) validateBasic() error {
	if len(tx.Inputs) == 0 {
		return ErrEmptyInputs
	}
	if len(tx.Outputs) == 0 {
		return ErrEmptyOutputs
	}
	return nil
}

// Size 计算交易大小（不含解锁数据）
func (tx *Transaction) Size() int {
	size := HeaderSize

	// 输入项大小
	for _, in := range tx.Inputs {
		size += len(in.Bytes())
	}

	// 输出项大小
	for _, out := range tx.Outputs {
		size += len(out.Bytes())
	}

	return size
}

// TotalInputAmount 计算输入总金额（需要 UTXO 集支持）
// 此方法需要外部提供 UTXO 查询接口
type UTXOGetter interface {
	GetUTXO(year uint16, txID types.Hash512, outIndex uint16) (Output, error)
}

func (tx *Transaction) TotalInputAmount(utxos UTXOGetter) (types.Amount, error) {
	var total types.Amount
	for _, in := range tx.Inputs {
		var utxo Output
		var err error
		if in.IsLeader {
			utxo, err = utxos.GetUTXO(in.Year, in.TxID, in.OutIndex)
		} else {
			// 需要从短ID解析完整ID
			return 0, errors.New("short ID lookup not implemented")
		}
		if err != nil {
			return 0, err
		}
		if coin, ok := utxo.(*CoinOutput); ok {
			total += coin.Amount
		}
	}
	return total, nil
}

// TotalOutputAmount 计算输出总金额
func (tx *Transaction) TotalOutputAmount() types.Amount {
	var total types.Amount
	for _, out := range tx.Outputs {
		if coin, ok := out.(*CoinOutput); ok {
			total += coin.Amount
		}
	}
	return total
}
```

### 5. coinbase.go - 铸币交易

```go
package tx

import (
	"encoding/binary"

	"evidcoin/pkg/types"
)

// CoinbaseData Coinbase 交易特有数据
type CoinbaseData struct {
	Height      uint64        // 区块高度
	ProofHash   types.Hash512 // 择优凭证哈希
	TotalReward types.Amount  // 收益总额
	FreeData    []byte        // 自由数据（<256字节）
}

// RewardDistribution 收益分成
type RewardDistribution struct {
	Minter    types.Amount // 铸造者（校验组）
	Proposer  types.Amount // 铸凭者（提案者）
	Findings  types.Amount // Findings 奖励
	Blockqs   types.Amount // Blockqs 奖励
	Archives  types.Amount // Archives 奖励
}

// CoinbaseTransaction Coinbase 交易
type CoinbaseTransaction struct {
	Header       *TxHeader
	Data         *CoinbaseData
	Distribution *RewardDistribution
	Outputs      []Output
}

// NewCoinbaseTransaction 创建 Coinbase 交易
func NewCoinbaseTransaction(height uint64, timestamp int64, proofHash types.Hash512) *CoinbaseTransaction {
	return &CoinbaseTransaction{
		Header: &TxHeader{
			Version:   1,
			Timestamp: timestamp,
		},
		Data: &CoinbaseData{
			Height:    height,
			ProofHash: proofHash,
		},
		Outputs: make([]Output, 0),
	}
}

// IsCoinbase 判断是否为 Coinbase 交易
func IsCoinbase(tx *Transaction) bool {
	// Coinbase 交易没有常规输入项
	return len(tx.Inputs) == 0
}

// CalcBlockReward 计算区块奖励
// 基础奖励随时间递减
func CalcBlockReward(height uint64) types.Amount {
	// 初始奖励：50 Bi
	baseReward := types.Amount(50 * types.Bi)
	
	// 每 87661 * 4 个区块（约4年）减半
	halvingInterval := uint64(87661 * 4)
	halvings := height / halvingInterval
	
	// 最多减半 10 次
	if halvings > 10 {
		halvings = 10
	}
	
	// 计算当前奖励
	reward := baseReward >> halvings
	return reward
}

// Bytes 序列化 Coinbase 数据
func (cb *CoinbaseData) Bytes() []byte {
	buf := make([]byte, 0, 128+len(cb.FreeData))
	
	// Height
	heightBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBuf, cb.Height)
	buf = append(buf, heightBuf...)
	
	// ProofHash
	buf = append(buf, cb.ProofHash[:]...)
	
	// TotalReward
	buf = append(buf, types.EncodeVarint(int64(cb.TotalReward))...)
	
	// FreeData
	buf = append(buf, byte(len(cb.FreeData)))
	buf = append(buf, cb.FreeData...)
	
	return buf
}
```

### 6. sigflag.go - 签名授权

```go
package tx

// SigFlag 签名授权标记
type SigFlag byte

const (
	// 独项
	SigInAll  SigFlag = 1 << 7 // 全部输入项
	SigInSelf SigFlag = 1 << 6 // 当前输入项

	// 主项
	SigOutAll  SigFlag = 1 << 5 // 全部输出项
	SigOutSelf SigFlag = 1 << 4 // 同序位输出项

	// 辅项
	SigReceiver SigFlag = 1 << 0 // 输出的接收者
	SigContent  SigFlag = 1 << 1 // 输出项的内容
	SigScript   SigFlag = 1 << 2 // 输出项的脚本
	SigOutput   SigFlag = 1 << 3 // 输出项完整条目
)

// DefaultSigFlag 默认签名标记：全部输入 + 全部输出完整条目
const DefaultSigFlag = SigInAll | SigOutAll | SigOutput

// BuildSignMessage 构建签名消息
func BuildSignMessage(tx *Transaction, inputIndex int, flag SigFlag) []byte {
	var message []byte

	// 处理输入部分
	if flag&SigInAll != 0 {
		// 全部输入项
		for _, in := range tx.Inputs {
			message = append(message, in.Bytes()...)
		}
	} else if flag&SigInSelf != 0 {
		// 仅当前输入项
		if inputIndex < len(tx.Inputs) {
			message = append(message, tx.Inputs[inputIndex].Bytes()...)
		}
	}

	// 处理输出部分
	outputs := tx.Outputs
	if flag&SigOutSelf != 0 {
		// 仅同序位输出
		if inputIndex < len(tx.Outputs) {
			outputs = []Output{tx.Outputs[inputIndex]}
		} else {
			outputs = nil
		}
	}

	for _, out := range outputs {
		if flag&SigOutput != 0 {
			// 完整条目
			message = append(message, out.Bytes()...)
		} else {
			// 部分条目
			if flag&SigReceiver != 0 {
				message = append(message, out.Receiver()[:]...)
			}
			if flag&SigContent != 0 {
				// 内容部分取决于输出类型
				// 这里简化处理
			}
			if flag&SigScript != 0 {
				message = append(message, out.LockScript()...)
			}
		}
	}

	return message
}
```

### 7. attachment.go - 附件ID

```go
package tx

import (
	"encoding/binary"
	"errors"
)

var (
	ErrInvalidAttachmentID = errors.New("invalid attachment ID")
)

// AttachmentType 附件类型
type AttachmentType struct {
	Major byte // 大类
	Minor byte // 小类
}

// AttachmentID 附件标识
type AttachmentID struct {
	Type        AttachmentType // 附件类型
	Fingerprint []byte         // 附件指纹（BLAKE3，16-64字节）
	ShardCount  uint16         // 分片数量
	ShardRoot   [48]byte       // 片组哈希（哈希树根）
	Size        int64          // 附件大小（字节）
}

// FingerprintStrength 指纹强度索引到字节长度
// 0=>16, 1=>20, 2=>24, ... 12=>64
func FingerprintStrength(strength byte) int {
	if strength > 12 {
		strength = 12
	}
	return 16 + int(strength)*4
}

// ParseAttachmentID 解析附件ID
func ParseAttachmentID(data []byte) (*AttachmentID, error) {
	if len(data) < 1 {
		return nil, ErrInvalidAttachmentID
	}

	totalLen := int(data[0])
	if len(data) < totalLen+1 {
		return nil, ErrInvalidAttachmentID
	}

	offset := 1
	aid := &AttachmentID{}

	// 类型（2字节）
	aid.Type.Major = data[offset]
	aid.Type.Minor = data[offset+1]
	offset += 2

	// 指纹强度（1字节）
	strength := data[offset]
	fpLen := FingerprintStrength(strength)
	offset++

	// 指纹
	aid.Fingerprint = make([]byte, fpLen)
	copy(aid.Fingerprint, data[offset:offset+fpLen])
	offset += fpLen

	// 分片数量（2字节）
	aid.ShardCount = binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	// 片组哈希（48字节）
	copy(aid.ShardRoot[:], data[offset:offset+48])
	offset += 48

	// 大小（变长整数）
	size, n := types.DecodeVarint(data[offset:])
	aid.Size = size
	offset += n

	return aid, nil
}

// Bytes 序列化附件ID
func (aid *AttachmentID) Bytes() []byte {
	// 计算指纹强度
	strength := byte((len(aid.Fingerprint) - 16) / 4)
	
	buf := make([]byte, 0, 128)
	
	// 占位符，后面填充总长度
	buf = append(buf, 0)
	
	// 类型
	buf = append(buf, aid.Type.Major, aid.Type.Minor)
	
	// 指纹强度
	buf = append(buf, strength)
	
	// 指纹
	buf = append(buf, aid.Fingerprint...)
	
	// 分片数量
	shardBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(shardBuf, aid.ShardCount)
	buf = append(buf, shardBuf...)
	
	// 片组哈希
	buf = append(buf, aid.ShardRoot[:]...)
	
	// 大小
	buf = append(buf, types.EncodeVarint(aid.Size)...)
	
	// 填充总长度
	buf[0] = byte(len(buf) - 1)
	
	return buf
}

// Validate 验证附件ID
func (aid *AttachmentID) Validate() error {
	if len(aid.Fingerprint) < 16 || len(aid.Fingerprint) > 64 {
		return errors.New("invalid fingerprint length")
	}
	if aid.ShardCount > 0 && aid.ShardCount == 1 {
		// 单分片时，ShardRoot 应为附件数据哈希
	}
	return nil
}
```

---

## 测试用例

### tx_test.go

```go
package tx

import (
	"testing"
	"time"

	"evidcoin/pkg/crypto"
	"evidcoin/pkg/types"
)

func TestTxHeader(t *testing.T) {
	header := &TxHeader{
		Version:   1,
		Timestamp: time.Now().UnixMilli(),
		HashBody:  types.Hash512{},
	}

	// 测试序列化
	data := header.Bytes()
	if len(data) != HeaderSize {
		t.Errorf("expected header size %d, got %d", HeaderSize, len(data))
	}

	// 测试解析
	parsed, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("failed to parse header: %v", err)
	}

	if parsed.Version != header.Version {
		t.Errorf("version mismatch: expected %d, got %d", header.Version, parsed.Version)
	}
	if parsed.Timestamp != header.Timestamp {
		t.Errorf("timestamp mismatch")
	}
}

func TestCoinOutput(t *testing.T) {
	addr := types.Address{}
	copy(addr[:], []byte("test_address_48_bytes_long_padded_here"))

	output := NewCoinOutput(0, 100*types.Bi, addr, []byte{0x01, 0x02})

	if output.Type() != OutputTypeCoin {
		t.Error("expected coin output type")
	}

	if output.Amount != 100*types.Bi {
		t.Errorf("amount mismatch: expected %d, got %d", 100*types.Bi, output.Amount)
	}

	data := output.Bytes()
	if len(data) == 0 {
		t.Error("serialization produced empty data")
	}
}

func TestTransaction(t *testing.T) {
	tx := NewTransaction(1, time.Now().UnixMilli())

	// 添加输入
	txID := crypto.Hash512([]byte("source_tx"))
	input := NewLeaderInput(2024, txID, 0)
	tx.AddInput(input)

	// 添加输出
	addr := types.Address{}
	output := NewCoinOutput(0, 50*types.Bi, addr, nil)
	tx.AddOutput(output)

	// 完成交易
	if err := tx.Finalize(); err != nil {
		t.Fatalf("failed to finalize tx: %v", err)
	}

	// 验证
	if err := tx.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// 测试 ID
	id := tx.ID()
	if id == (types.Hash512{}) {
		t.Error("tx ID should not be empty")
	}
}

func TestCalcInputHash(t *testing.T) {
	txID := crypto.Hash512([]byte("test"))
	
	inputs := []*Input{
		NewLeaderInput(2024, txID, 0),
		NewRestInput(2024, types.ShortHash{}, 1),
	}

	hash := CalcInputHash(inputs)
	if hash == (types.Hash512{}) {
		t.Error("input hash should not be empty")
	}
}

func TestBlockReward(t *testing.T) {
	// 初始奖励
	reward0 := CalcBlockReward(0)
	if reward0 != 50*types.Bi {
		t.Errorf("initial reward should be 50 Bi, got %d", reward0)
	}

	// 第一次减半后
	halvingHeight := uint64(87661 * 4)
	reward1 := CalcBlockReward(halvingHeight)
	if reward1 != 25*types.Bi {
		t.Errorf("reward after first halving should be 25 Bi, got %d", reward1)
	}
}

func TestSigFlag(t *testing.T) {
	flag := DefaultSigFlag
	
	if flag&SigInAll == 0 {
		t.Error("default flag should include SigInAll")
	}
	if flag&SigOutAll == 0 {
		t.Error("default flag should include SigOutAll")
	}
	if flag&SigOutput == 0 {
		t.Error("default flag should include SigOutput")
	}
}
```

---

## 实现步骤

### Step 1: 创建包结构

```bash
mkdir -p internal/tx
touch internal/tx/header.go
touch internal/tx/input.go
touch internal/tx/output.go
touch internal/tx/tx.go
touch internal/tx/coinbase.go
touch internal/tx/sigflag.go
touch internal/tx/attachment.go
touch internal/tx/serialize.go
touch internal/tx/tx_test.go
```

### Step 2: 按顺序实现

1. `header.go` - 交易头（依赖 crypto, types）
2. `input.go` - 输入项
3. `output.go` - 输出项（币金、凭信、存证）
4. `tx.go` - 完整交易结构
5. `coinbase.go` - Coinbase 交易
6. `sigflag.go` - 签名授权
7. `attachment.go` - 附件ID
8. `serialize.go` - 完整序列化/反序列化

### Step 3: 测试验证

```bash
go test -v ./internal/tx/...
```

---

## 注意事项

1. **交易大小限制**: 单笔交易不含解锁数据最大 8KB
2. **输入项引用**: 首领输入用 64 字节全 ID，其余用 20 字节短 ID
3. **输出项序位**: 用于哈希树构建和验证
4. **凭信合规性**: 转移时需检查可修改性规则
5. **存证不可转移**: 存证输出不能作为输入项
6. **签名消息构建**: 根据 SigFlag 选择性包含交易部分
