package tx

import "errors"

// 签名与见证层错误定义（第 08 章，DEC-0102/0103/0104）。

var (
	// ErrOutputAuxConflict 表示 OUTPUT 与 SCRIPT/CONTENT/RECEIVER 任一辅项并存
	// （DEC-0102 §4 互斥规则），签名消息构造失败、交易必须拒绝。
	ErrOutputAuxConflict = errors.New("tx: auth_flag OUTPUT conflicts with SCRIPT/CONTENT/RECEIVER")
	// ErrMainWithoutAux 表示主项（SIGOUT_ALL/SIGOUT_SELF）存在但无辅项配合（DEC-0102 §4）。
	ErrMainWithoutAux = errors.New("tx: auth_flag main output bit requires an auxiliary bit")
	// ErrAuxWithoutMain 表示辅项存在但无主项配合（DEC-0102 §4）。
	ErrAuxWithoutMain = errors.New("tx: auth_flag auxiliary bit requires a main output bit")
	// ErrSigOutSelfRange 表示 SIGOUT_SELF 的 input_index 越界（>= len(outputs)），
	// 签名验证必须失败（DEC-0102 §3）。
	ErrSigOutSelfRange = errors.New("tx: SIGOUT_SELF input_index out of output range")
	// ErrInputIndexRange 表示签名消息引用的 input_index 越界于输入集（SIGIN_SELF）。
	ErrInputIndexRange = errors.New("tx: input_index out of input range")

	// ErrMultisigConfig 表示多签 m/n 配比非法（m 或 n 为 0，或 m > n）。
	ErrMultisigConfig = errors.New("tx: invalid multisig m/n configuration")
	// ErrMultisigSetMismatch 表示签名集与公钥集数量不一致，或集合规模与 m/n 不符。
	ErrMultisigSetMismatch = errors.New("tx: multisig signature/public-key set size mismatch")
	// ErrCompletionNotSorted 表示补全公钥哈希集未按字典序升序排列（DEC-0103）。
	ErrCompletionNotSorted = errors.New("tx: completion base-hash set not in ascending order")
	// ErrReceiverMismatch 表示由见证派生的复合/单签公钥哈希与接收者不一致。
	ErrReceiverMismatch = errors.New("tx: derived public-key hash does not match receiver")
	// ErrSignatureInvalid 表示某个签名验证失败。
	ErrSignatureInvalid = errors.New("tx: signature verification failed")
)
