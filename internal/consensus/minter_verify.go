package consensus

import (
	"errors"

	"github.com/cxio/evidcoin/internal/consensus/equix"
	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// 铸造者三段验证（第 11 章 §6）。
//
// 前提：客户端持有区块头链作为验证终点。竞争者除 MintProof 外提供：
// TxID 到 TreeRoot 的验证路径（叶含 3 字节前置序号，由数据源提供）；
// 若铸凭交易未设 MintPKHash，还需提供 ListHash 以计算输入根。
//
// 本层不直接 import 公共服务客户端：外部检索数据经 MintDataSource 注入，
// 必须先验证再使用，不得作为可信输入。

// RetrievedMintTx 是数据源检索到的铸凭交易关键字段，用于重算交易头哈希、
// 判定身份并计算铸凭哈希。字段语义与第 06 章普通交易头一致。
type RetrievedMintTx struct {
	// Version/HashInputs/HashOutputs/Timestamp 用于重算普通交易头哈希。
	Version     uint16
	HashInputs  types.Hash32
	HashOutputs types.Hash32
	Timestamp   int64
	// MintPKHash 是检索到的铸凭公钥哈希；长度 0 表示未设置（走 LeadPKHash 路径）。
	MintPKHash []byte
	// ListHash 是输入项列表哈希，仅在 MintPKHash 为空（LeadPKHash 路径）时需要。
	ListHash types.Hash48
	// Stakes 是 -32 号区块头币权销毁值（聪时），用于重算铸凭哈希。
	Stakes uint64
	// RefMintHash 是评参区块 Coinbase 记录的铸凭哈希；创世/初段无 Minter 时全零。
	RefMintHash types.MintHash
}

// MintDataSource 提供铸造者验证所需的外部检索数据。实现者（如经验证的公共服务
// 适配层）负责检索；本层只消费，不信任未经验证的数据。
type MintDataSource interface {
	// MintTransactionHeader 按 TxID 检索铸凭交易头字段。
	MintTransactionHeader(id types.TxID) (RetrievedMintTx, error)
	// CheckRootAt 按区块高度返回该区块头的 CheckRoot。
	CheckRootAt(height uint32) (types.CheckRoot, error)
	// InclusionPath 返回 TxID 到目标区块交易树根的验证路径。
	InclusionPath(id types.TxID) (hashtree.Proof, error)
}

// MintSignatureVerifier 校验铸造者对铸凭哈希的签名。具体后量子算法（ML-DSA-65）
// 实现尚未就绪，由上层注入封装 crypto.Verifier 的实现，避免本层固化算法依赖。
type MintSignatureVerifier interface {
	// VerifyMintSignature 校验 pubKey 对 message 的签名 sig 是否有效。
	VerifyMintSignature(pubKey, message, sig []byte) (bool, error)
}

// MinterVerifyConfig 是三段验证的上下文参数。
type MinterVerifyConfig struct {
	// CurrentHeight 是当前待铸区块高度（用于铸凭交易资格窗口判定与铸凭哈希前像计算）。
	CurrentHeight uint32
	// StateUTXORoot/StateUTCORoot 是目标区块的 UTXO/UTCO 状态指纹，
	// 与验证路径推出的交易树根一起组合 CheckRoot 进行比对（第 05 章 §2）。
	StateUTXORoot types.TreeHash
	StateUTCORoot types.TreeHash
	// SigVerifier 校验铸造者对铸凭哈希的签名。
	SigVerifier MintSignatureVerifier
	// EquiXSolver 提供 Equi-X 解验证（DEC-0301 第二阶段）。
	// 为 nil 时返回 ErrEquiXUnavailable。
	EquiXSolver equix.Solver
}

// VerifyMinter 执行铸造者三段验证（第 11 章 §6）：
//  1. 交易 ID 合法：高度在窗口内；按检索的 MintPKHash 区分身份并重算交易头哈希，
//     验证与 MintProof.TxID 匹配；
//  2. 属于目标区块：用验证路径推出的交易树根与状态指纹组合 CheckRoot，
//     与数据源返回的目标区块 CheckRoot 比对；
//  3. 铸造者身份：重算铸凭哈希与 MintProof.MintHash 比对，并校验对铸凭哈希的签名。
func VerifyMinter(ds MintDataSource, mp MintProof, cfg MinterVerifyConfig) error {
	// —— 第一段：交易 ID 合法 ——
	if !MintTxEligibleNormal(cfg.CurrentHeight, mp.TxHeight) {
		return ErrMintHeightOutOfWindow
	}

	retrieved, err := ds.MintTransactionHeader(mp.TxID)
	if err != nil {
		return err
	}

	// 身份两路：按检索的 MintPKHash 是否为空区分。
	if len(retrieved.MintPKHash) == 32 {
		if err := VerifyMintIdentityWithPKHash(mp.MintPubKey, retrieved.MintPKHash); err != nil {
			return err
		}
	} else {
		// LeadPKHash 路径：公钥哈希作 LeadPKHash 参与输入根，须与重算交易头输入根一致。
		hdrInputRoot := buildInputRootFromRetrieved(retrieved)
		if err := VerifyMintIdentityWithLeadInput(mp.MintPubKey, retrieved.ListHash, hdrInputRoot); err != nil {
			return err
		}
	}

	// 重算交易头哈希，验证与 MintProof.TxID 匹配。
	hdr := &tx.TxHeader{
		Version:     retrieved.Version,
		HashInputs:  retrieved.HashInputs,
		HashOutputs: retrieved.HashOutputs,
		Timestamp:   retrieved.Timestamp,
		MintPKHash:  retrieved.MintPKHash,
	}
	gotTxID, err := hdr.TxID()
	if err != nil {
		return err
	}
	if gotTxID != mp.TxID {
		return ErrMintIdentityMismatch
	}

	// —— 第二段：属于目标区块 ——
	proof, err := ds.InclusionPath(mp.TxID)
	if err != nil {
		return err
	}
	if !hashtree.Verify(proof) {
		return ErrInclusionPathInvalid
	}
	wantCheckRoot, err := ds.CheckRootAt(mp.TxHeight)
	if err != nil {
		return err
	}
	gotCheckRoot := combineCheckRoot(proof.Root, cfg.StateUTXORoot, cfg.StateUTCORoot)
	if gotCheckRoot != wantCheckRoot {
		return ErrCheckRootMismatch
	}

	// —— 第三段：铸造者身份 ——
	pre := MintHashPreimage{
		MintPubKey:  mp.MintPubKey,
		MintTxID:    mp.TxID,
		Stakes:      retrieved.Stakes,
		RefMintHash: retrieved.RefMintHash,
		BlockHeight: cfg.CurrentHeight,
	}
	if cfg.EquiXSolver == nil {
		return ErrEquiXUnavailable
	}
	challengeSeed := ComputeChallengeSeed(pre)
	// 基本规则检查（第 11 章 §4）：优先排除不合法项，再调用工作量验证。
	if mp.Nonce < uint64(cfg.CurrentHeight) {
		return ErrNonceTooSmall
	}
	if !isStrictAscendingUniqueSolution(mp.Solution) {
		return ErrEquiXSolutionInvalid
	}
	hashList, valid, equixErr := cfg.EquiXSolver.Verify(challengeSeed, mp.Nonce, mp.Solution)
	if equixErr != nil {
		if errors.Is(equixErr, equix.ErrUnavailable) {
			return ErrEquiXUnavailable
		}
		return equixErr
	}
	if !valid {
		return ErrEquiXSolutionInvalid
	}
	if ComputeMintHash(hashList) != mp.MintHash {
		return ErrMintHashMismatch
	}
	ok, err := cfg.SigVerifier.VerifyMintSignature(mp.MintPubKey, mp.MintHash.Bytes(), mp.Signature)
	if err != nil {
		return err
	}
	if !ok {
		return ErrMintSignatureInvalid
	}
	return nil
}

// buildInputRootFromRetrieved 在 LeadPKHash 路径下，从检索到的交易头还原其声明的输入根。
// 这里直接返回检索头中的 HashInputs；身份验证负责确认 LeadPKHash 推出的根与之一致。
func buildInputRootFromRetrieved(r RetrievedMintTx) types.Hash32 {
	return r.HashInputs
}

// combineCheckRoot 复用第 05 章 §2 的 CheckRoot 前像规则：
// SHA3-384( DomainTag("checkroot") || TreeRoot || UTXORoot || UTCORoot )，顺序不可换。
// 本层不反向 import blockchain，按相同规则就地组合以比对。
func combineCheckRoot(treeRoot []byte, utxoRoot, utcoRoot types.TreeHash) types.CheckRoot {
	pre := make([]byte, 0, len(treeRoot)+32+32)
	pre = append(pre, treeRoot...)
	pre = append(pre, utxoRoot.Bytes()...)
	pre = append(pre, utcoRoot.Bytes()...)
	return crypto.HashCheckRoot(pre)
}

// YearSearchHeights 返回铸造者信息检索时应尝试的年度代表高度集合（第 11 章 §6 年初回退）。
// 当 height 处于某年度起始 1 天（ExpiryWindow=240 块）内且非首年时，除当年外追加上一年度
// 代表高度（年初回退检索一次）；否则仅返回当年。区块不收录未来交易，故无需向后回退。
func YearSearchHeights(height uint32) []uint32 {
	year := height / types.BlocksPerYear
	withinYearStart := height%types.BlocksPerYear < ExpiryWindowBlocks
	if withinYearStart && year > 0 {
		return []uint32{height, (year - 1) * types.BlocksPerYear}
	}
	return []uint32{height}
}

// isStrictAscendingUniqueSolution 校验 solution 字节序列严格升序且无重复。
// 该前置检查用于在调用 Equi-X 验证前快速拒绝明显不合法的解。
func isStrictAscendingUniqueSolution(solution []byte) bool {
	for i := 1; i < len(solution); i++ {
		if solution[i] <= solution[i-1] {
			return false
		}
	}
	return true
}

// ExpiryWindowBlocks 是「年初 1 天」回退窗口的区块数（与第 03 章 ExpiryWindow 一致）。
const ExpiryWindowBlocks = types.ExpiryWindow
