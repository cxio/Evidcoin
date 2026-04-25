package tx

import "github.com/cxio/evidcoin/pkg/types"

// ---- 输出配置字节标志 ----

const (
	// OutFlagCustomClass 自定义类标志（Bit 7），余下 7 位为类 ID 长度
	OutFlagCustomClass = byte(1 << 7)
	// OutFlagHasAttach 包含附件标志（Bit 6）
	OutFlagHasAttach = byte(1 << 6)
	// OutFlagDestroy 销毁标志（Bit 5）
	OutFlagDestroy = byte(1 << 5)
	// OutTypeMask 低 4 位类型掩码
	OutTypeMask = byte(0x0f)
)

// CoinbaseOutputTarget Coinbase 输出目标类型（低 4 位）
type CoinbaseOutputTarget byte

const (
	// CoinbaseMinter 铸凭者（10%）
	CoinbaseMinter CoinbaseOutputTarget = 1
	// CoinbaseCheckTeam 校验组（40%）
	CoinbaseCheckTeam CoinbaseOutputTarget = 2
	// CoinbaseBlockqs 区块查询服务（20%）
	CoinbaseBlockqs CoinbaseOutputTarget = 3
	// CoinbaseDepots 数据驿站服务（20%）
	CoinbaseDepots CoinbaseOutputTarget = 4
	// CoinbaseSTUN NAT 穿透服务（10%）
	CoinbaseSTUN CoinbaseOutputTarget = 5
)

// ---- 输出结构 ----

// Output 交易输出项，承载 Coin / Credit / Proof / Mediator 之一。
type Output struct {
	// Serial 输出序位（从 0 开始）
	Serial int
	// Config 配置字节（类型 + 标志位）
	Config byte
	// Amount 币金数量（最小单位 chx/聪，仅 Coin 有效）
	Amount int64
	// Address 接收地址（公钥哈希，32 字节；Destroy 时可为零值）
	Address types.PubKeyHash
	// LockScript 锁定脚本（最大 1024 字节；Proof 无此字段，改用 IdentScript）
	LockScript []byte
	// ---- 以下字段按信元类型选择性存在 ----
	// Memo 币金附言（最大 255 字节）
	Memo []byte
	// Creator 创建者标识（凭信/存证，< 256 字节）
	Creator []byte
	// Title 标题（凭信/存证，最大 255 字节）
	Title []byte
	// Desc 描述（凭信，最大 1023 字节）
	Desc []byte
	// Content 内容（存证，最大 4095 字节）
	Content []byte
	// AttachID 附件 ID（可选）
	AttachID []byte
	// CredConfig 凭信配置（2 字节）
	CredConfig uint16
	// ProofContentLen 存证内容长度字段（2 字节，低 12 位为内容长度）
	ProofContentLen uint16
	// IdentScript 识别脚本（仅存证使用）
	IdentScript []byte
}

// Type 返回输出类型（低 4 位）。
func (o *Output) Type() byte {
	return o.Config & OutTypeMask
}

// IsDestroy 返回销毁标志是否置位。
func (o *Output) IsDestroy() bool {
	return o.Config&OutFlagDestroy != 0
}

// HasAttachment 返回是否包含附件。
func (o *Output) HasAttachment() bool {
	return o.Config&OutFlagHasAttach != 0
}

// IsCustomClass 返回是否为自定义类输出。
func (o *Output) IsCustomClass() bool {
	return o.Config&OutFlagCustomClass != 0
}

// CanBeUTXO 返回该输出是否可以进入 UTXO 集。
func (o *Output) CanBeUTXO() bool {
	if o.IsDestroy() || o.IsCustomClass() {
		return false
	}
	return o.Type() == types.OutTypeCoin
}

// CanBeUTCO 返回该输出是否可以进入 UTCO 集。
func (o *Output) CanBeUTCO() bool {
	if o.IsDestroy() || o.IsCustomClass() {
		return false
	}
	return o.Type() == types.OutTypeCredit
}

// ---- 输入结构 ----

// LeadInput 首领输入（使用完整 48 字节 TxID）。
type LeadInput struct {
	// Year 被引用交易所在年度
	Year int
	// TxID 完整交易 ID（48 字节）
	TxID types.Hash384
	// OutIndex 被引用输出序位
	OutIndex int
}

// RestInput 非首领输入（使用截断的 20 字节 TxID 引用）。
type RestInput struct {
	// Year 被引用交易所在年度
	Year int
	// TxIDPart 交易 ID 前 20 字节
	TxIDPart [20]byte
	// OutIndex 被引用输出序位
	OutIndex int
	// TransferIndex 凭信转出序位（可选，-1 表示不存在）
	TransferIndex int
}

// ---- 交易头 ----

// TxHeader 交易头结构。
// TxID = SHA3-384( TxHeader )，签名不参与计算。
type TxHeader struct {
	// Version 版本号
	Version int
	// Timestamp 交易时间戳（Unix 毫秒）
	Timestamp int64
	// HashInputs 输入项根哈希（BLAKE3-256，32 字节）
	HashInputs types.Hash256
	// HashOutputs 输出项根哈希（BLAKE3-256，32 字节）
	HashOutputs types.Hash256
}

// SigFlag 签名授权标志（1 字节）。
type SigFlag byte

const (
	// SIGIN_ALL 全部输入项（独项，Bit 7）
	SIGIN_ALL SigFlag = 1 << 7
	// SIGIN_SELF 仅当前输入项（独项，Bit 6）
	SIGIN_SELF SigFlag = 1 << 6
	// SIGOUT_ALL 全部输出项（主项，Bit 5）
	SIGOUT_ALL SigFlag = 1 << 5
	// SIGOUT_SELF 与当前输入同序位的输出项（主项，Bit 4）
	SIGOUT_SELF SigFlag = 1 << 4
	// SIGOUTPUT 完整输出条目（辅项，Bit 3）
	SIGOUTPUT SigFlag = 1 << 3
	// SIGSCRIPT 输出的锁定脚本（辅项，Bit 2）
	SIGSCRIPT SigFlag = 1 << 2
	// SIGCONTENT 输出内容（辅项，Bit 1）
	SIGCONTENT SigFlag = 1 << 1
	// SIGRECEIVER 输出的接收者（辅项，Bit 0）
	SIGRECEIVER SigFlag = 1 << 0
)

// Outpoint 引用交易输出的唯一标识。
type Outpoint struct {
	// TxID 所在交易 ID
	TxID types.Hash384
	// Index 输出序位
	Index int
}
