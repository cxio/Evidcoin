// Package consensus 承载历史证明（PoH）的铸造资格判定、铸凭哈希计算、
// 择优凭证与择优池、择优池同步、铸造者身份验证、创世与初段窗口规则（第 11 章）。
//
// 本包属 Layer 4，仅依赖 Layer 0-3 的稳定类型与接口，不被低层包反向 import。
// 本包不实现分叉选择与出块时序（属第 12 章）、不实现 P2P 线格式、
// 不直接 import 公共服务客户端；外部数据必须先验证再使用。
package consensus

import "errors"

// 铸凭与择优相关错误。程序输出文本使用英文。
var (
	// ErrMintProofTooShort 表示 MintProof 字节不足以解析固定字段。
	ErrMintProofTooShort = errors.New("consensus: mint proof too short")
	// ErrMintProofTrailing 表示 MintProof 解析后存在多余尾随字节。
	ErrMintProofTrailing = errors.New("consensus: mint proof has trailing bytes")
	// ErrMintIdentityMismatch 表示铸造者公钥哈希与铸凭身份不匹配。
	ErrMintIdentityMismatch = errors.New("consensus: mint identity mismatch")
	// ErrInputRootMismatch 表示 LeadPKHash 参与的输入根校验失败。
	ErrInputRootMismatch = errors.New("consensus: input root mismatch")
	// ErrCoinbaseMintPKHashMissing 表示 Coinbase 未显式设置 MintPKHash，不可参与竞争。
	ErrCoinbaseMintPKHashMissing = errors.New("consensus: coinbase mint pk hash missing")
	// ErrMintHashMismatch 表示重算铸凭哈希与 MintProof.MintHash 不一致。
	ErrMintHashMismatch = errors.New("consensus: recomputed mint hash mismatch")
	// ErrMintSignatureInvalid 表示铸造者对铸凭哈希的签名无效。
	ErrMintSignatureInvalid = errors.New("consensus: mint signature invalid")
	// ErrMintHeightOutOfWindow 表示铸凭交易高度不在允许窗口内。
	ErrMintHeightOutOfWindow = errors.New("consensus: mint tx height out of window")
	// ErrCheckRootMismatch 表示目标区块 CheckRoot 校验失败。
	ErrCheckRootMismatch = errors.New("consensus: check root mismatch")
	// ErrInclusionPathInvalid 表示交易 ID 到树根的验证路径无效。
	ErrInclusionPathInvalid = errors.New("consensus: inclusion path invalid")
	// ErrNotAuthorizedSyncer 表示发起同步的成员不在后 15 名授权集内。
	ErrNotAuthorizedSyncer = errors.New("consensus: syncer not authorized")
	// ErrSyncReplay 表示同一授权节点对同一目标池重复发起同步。
	ErrSyncReplay = errors.New("consensus: sync replay rejected")

	// ErrPublishStageInvalid 表示区块发布阶段迁移非法（回退或重复）。
	ErrPublishStageInvalid = errors.New("consensus: publish stage advance invalid")
	// ErrForkTooLong 表示新观察到的分叉超过接收上限 20。
	ErrForkTooLong = errors.New("consensus: fork segment exceeds accept limit")
	// ErrDecisionNoQuorum 表示长度 20 临界分叉裁决中前 5 名均未签名，默认否决。
	ErrDecisionNoQuorum = errors.New("consensus: critical fork decision rejected by no quorum")
	// ErrRandomXUnavailable 表示 RandomX 哈希器未注入，无法执行平局裁决。
	ErrRandomXUnavailable = errors.New("consensus: randomx hasher unavailable")
)
