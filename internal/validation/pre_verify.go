package validation

import (
	"bytes"

	"github.com/cxio/evidcoin/internal/blockchain"
	"github.com/cxio/evidcoin/internal/consensus"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// PoolMemberChecker 查询公钥是否属于当前最优择优池成员（第 13 章 §5 步骤 2）。
// 调用方负责将共识层 BestPool 包装为此接口。
type PoolMemberChecker interface {
	// IsBestPoolMember 报告给定公钥是否为当前择优池的有效成员。
	IsBestPoolMember(mintPubKey []byte) bool
}

// CheckRootSigVerifier 验证铸造者对 CheckRoot 的签名（DEC-0601 步骤 9，DEC-0102 §5）。
// 调用方负责提供匹配铸造者签名算法的验证实现（如 ML-DSA）。
type CheckRootSigVerifier interface {
	// VerifyCheckRootSig 验证公钥 pubKey 对 checkRoot 的签名 sig 是否有效。
	// 消息构造遵循 DEC-0102 §5：message = checkRoot.Bytes()。
	VerifyCheckRootSig(pubKey, sig []byte, checkRoot types.CheckRoot) (bool, error)
}

// LocalState 是节点当前本地链末端状态（第 13 章 §5 步骤 1, 3）。
// 仅包含快速预验证所需的最小字段。
type LocalState struct {
	// TipBlockID 是本地链末端区块 ID（用于验证候选区块的 PrevBlock 衔接）。
	TipBlockID types.BlockID
	// UTXORoot 是当前已完成区块后的 UTXO 状态指纹。
	UTXORoot types.TreeHash
	// UTCORoot 是当前已完成区块后的 UTCO 状态指纹。
	UTCORoot types.TreeHash
}

// PreVerify 对区块证明包执行 9 步快速预验证（DEC-0601，第 13 章 §5）。
// 验证顺序从廉价到昂贵：先做结构/衔接检查，最后做签名验证。
//
// 所有步骤通过（无错误）表示证明包可进入候选转播池；
// 不通过（返回非 nil）则丢弃此证明包，不对来源施以协议惩罚（转播误差容忍）。
//
// 注意：PreVerify 不证明 UTXO/UTCO 状态的真实性——步骤 3 仅将包中声明值与本地状态比对；
// 完整状态验证需要在打包完成后独立执行。
func PreVerify(pp ProofPackage, local LocalState, pool PoolMemberChecker, sv CheckRootSigVerifier) error {
	// 步骤 1：PrevBlock 衔接本地末端区块 ID。
	if pp.BlockHeader.PrevBlock != local.TipBlockID {
		return ErrPrevBlockMismatch
	}

	// 步骤 2：Minter 字段存在，且铸造者公钥属于当前择优池。
	if len(pp.CoinbaseTx.Minter) == 0 {
		return ErrNoMinterField
	}
	mintProof, _, err := consensus.ReadMintProof(pp.CoinbaseTx.Minter)
	if err != nil {
		// Minter 字节无法解析为合法 MintProof
		return ErrNoMinterField
	}
	if !pool.IsBestPoolMember(mintProof.MintPubKey) {
		return ErrMinterNotInPool
	}

	// 步骤 3：UTXORoot 与 UTCORoot 与本地当前状态吻合。
	if pp.UTXORoot != local.UTXORoot || pp.UTCORoot != local.UTCORoot {
		return ErrStateRootMismatch
	}

	// 步骤 4：CoinbaseTxIndex 必须为 0。
	if pp.CoinbaseTxIndex != 0 {
		return ErrCoinbaseTxIndexNot0
	}

	// 步骤 5：计算 Coinbase TxID。
	coinbaseTxID, err := pp.CoinbaseTx.TxID()
	if err != nil {
		return ErrCoinbaseTxIDFailed
	}

	// 步骤 6：验证 Coinbase 的 Merkle 路径。
	// 6a：叶哈希应为 LeafHash(OrderedLeaf3(CoinbaseTxIndex, coinbaseTxID.Bytes()))
	expectedLeaf := hashtree.LeafHash(hashtree.OrderedLeaf3(pp.CoinbaseTxIndex, coinbaseTxID.Bytes()))
	if !bytes.Equal(pp.CoinbaseMerklePath.LeafHash, expectedLeaf) {
		return ErrTreeRootMismatch
	}
	// 6b：路径自身有效（从叶哈希推算出 Root 与 CoinbaseMerklePath.Root 一致）。
	if !hashtree.Verify(pp.CoinbaseMerklePath) {
		return ErrTreeRootMismatch
	}
	// 6c：路径 Root 与 ProofPackage.TreeRoot 一致。
	if !bytes.Equal(pp.CoinbaseMerklePath.Root, pp.TreeRoot) {
		return ErrTreeRootMismatch
	}

	// 步骤 7：重算 CheckRoot，对照 BlockHeader.CheckRoot。
	gotCheckRoot := blockchain.ComputeCheckRoot(pp.TreeRoot, pp.UTXORoot, pp.UTCORoot)
	if gotCheckRoot != pp.BlockHeader.CheckRoot {
		return ErrCheckRootMismatch
	}

	// 步骤 8（略过）：MintHash 签名在铸造者加入择优池时已完成验证，无需重复校验。

	// 步骤 9：验证铸造者对 CheckRoot 的签名。
	ok, err := sv.VerifyCheckRootSig(mintProof.MintPubKey, pp.MinterCheckRootSignature, pp.BlockHeader.CheckRoot)
	if err != nil || !ok {
		return ErrMinterSigInvalid
	}

	return nil
}
