package consensus

import "github.com/cxio/evidcoin/pkg/types"

// 创世工件结构与初段窗口规则（第 11 章 §7，DEC-0302）。
//
// 待决 C-9：mainnet 的创世 BlockID（Genesis-ID）与创世时间戳尚未冻结。
// 本文件只固定创世工件的**结构**与初段窗口**规则**；genesis.bin 的具体字节取值
// （创世时间戳、mainnet Genesis-ID）依赖 C-9 数值裁决，裁决前以占位标注阻塞，
// 不虚构具体值（与 internal/blockchain/genesis.go、identity.go 的 C-9 处理一致）。

const (
	// initialEvalRefOffset 是初段评参区块偏移：currentHeight >= 8 时取 -8（DEC-0302）。
	initialEvalRefOffset = 8
	// initialMintRelaxLimit 是初段铸凭高度放宽的上界：
	// currentHeight < 480 时铸凭高度判定放宽（DEC-0302）。
	initialMintRelaxLimit = 480
)

// InInitialMintWindow 判定是否处于初段铸凭放宽期：currentHeight < 480（DEC-0302）。
func InInitialMintWindow(currentHeight uint32) bool {
	return currentHeight < initialMintRelaxLimit
}

// InitialEvalRefHeight 返回初段评参区块高度（DEC-0302）：
// currentHeight < 8 取 0（创世块）；currentHeight >= 8 取 currentHeight - 8。
func InitialEvalRefHeight(currentHeight uint32) uint32 {
	if currentHeight < initialEvalRefOffset {
		return 0
	}
	return currentHeight - initialEvalRefOffset
}

// MintTxEligibleInitial 判定初段铸凭交易高度资格（DEC-0302）：
//   - currentHeight < 480：放宽为 txHeight < currentHeight（仍须引用已确认交易，
//     当前待铸区块内交易不得自引用为铸凭交易）；
//   - currentHeight >= 480：切回正常窗口 h > 239 && h <= 80000。
//
// #1/#2 的特殊处理由调用方在上层隔离，不泄漏进本高度逻辑。
func MintTxEligibleInitial(currentHeight, txHeight uint32) bool {
	if currentHeight < initialMintRelaxLimit {
		// 放宽期：引用任何更早的已确认交易即可；自引用（txHeight >= currentHeight）拒绝。
		return txHeight < currentHeight
	}
	return MintTxEligibleNormal(currentHeight, txHeight)
}

// GenesisMintHash 返回创世块的铸凭哈希定义值：32 字节全零（DEC-0302）。
// 该全零值仅用于 #1~#7 评参区块铸凭哈希引用语义，不表示有效择优凭证。
func GenesisMintHash() types.MintHash {
	return types.MintHash{}
}

// GenesisArtifact 是创世工件结构（客户端硬编码、发布后不可变，DEC-0302）。
// 双形式发布：genesis.bin（canonical 二进制，权威共识形式）为本结构的规范序列化；
// genesis.json（人工审阅形式）仅供交叉校验，不参与共识。
//
// 五项组成（DEC-0302）：
//   - HeaderBytes：创世区块头完整编码（见 internal/blockchain.NewGenesisHeader）；
//   - CoinbaseBytes：创世 Coinbase 完整交易体（Minter 省略，见 internal/tx.CoinbaseHeader）；
//   - CoinbaseSignature：创世铸造者对 Coinbase 的签名；
//   - CheckRootSignature：创世铸造者对区块 CheckRoot 的签名（链根锚定，第 08 章）；
//   - FreeData：创世声明自由数据。
type GenesisArtifact struct {
	HeaderBytes        []byte
	CoinbaseBytes      []byte
	CoinbaseSignature  []byte
	CheckRootSignature []byte
	FreeData           []byte
}

// —— C-9 占位（创世具体参数未冻结，阻塞至作者裁决，不虚构 mainnet 值）——

// GenesisTimestampPlaceholder 是创世时间戳的占位常量。具体数值属 C-9 待决，
// 裁决前不得用真实 mainnet 时间戳填充；此占位仅表达「尚未冻结」语义。
const GenesisTimestampPlaceholder int64 = 0

// GenesisTimestamp 返回创世时间戳。C-9 裁决前返回占位值（未冻结）。
func GenesisTimestamp() int64 {
	return GenesisTimestampPlaceholder
}

// MainnetGenesisID 返回 mainnet 创世 BlockID（Genesis-ID）。
// C-9 裁决前返回零值（未冻结），不虚构具体值。
func MainnetGenesisID() types.BlockID {
	return types.BlockID{}
}

// GenesisParamsUnfrozen 报告创世具体参数（时间戳、mainnet Genesis-ID）是否仍未冻结。
// 在 C-9 裁决并回填真实值之前恒为 true；用于上层在装配 genesis.bin 时显式阻塞。
func GenesisParamsUnfrozen() bool {
	return GenesisTimestamp() == GenesisTimestampPlaceholder &&
		MainnetGenesisID() == (types.BlockID{})
}
