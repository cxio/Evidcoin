package tx

import "errors"

// 交易核心层错误定义（第 06、07 章）。

var (
	// ErrMintPKHashLength 表示普通交易头 MintPKHash 长度不属于 {0, 32}（DEC-0003）。
	ErrMintPKHashLength = errors.New("tx: MintPKHash length must be 0 or 32")
	// ErrTxIDPartTooShort 表示输入短引用 TxIDPart 短于 MinTxIDPartLen（16 字节）。
	ErrTxIDPartTooShort = errors.New("tx: input TxIDPart shorter than minimum")
	// ErrUnlockScriptTooLong 表示解锁脚本超过 MaxUnlockScript。
	ErrUnlockScriptTooLong = errors.New("tx: unlock script exceeds maximum length")
	// ErrInputKindInvalid 表示其余输入来源类别非法（仅允许币金/凭信，存证不可作输入源）。
	ErrInputKindInvalid = errors.New("tx: input kind must be coin or credit")
	// ErrCustomIDTooLong 表示自定义类私有标识 ID 超过 127 字节。
	ErrCustomIDTooLong = errors.New("tx: custom class id exceeds 127 bytes")
	// ErrOutputType 表示输出类型值为预留(0)或未知（非法位置）。
	ErrOutputType = errors.New("tx: invalid output type value")
	// ErrLockScriptTooLong 表示锁定脚本超过 MaxLockScript。
	ErrLockScriptTooLong = errors.New("tx: lock script exceeds maximum length")
	// ErrReceiverTooLong 表示 Receiver 字段超过 255 字节。
	ErrReceiverTooLong = errors.New("tx: receiver exceeds 255 bytes")
	// ErrMemoTooLong 表示 Memo 字段超过 255 字节。
	ErrMemoTooLong = errors.New("tx: memo exceeds 255 bytes")
	// ErrCreatorTooLong 表示 Creator 字段超过 255 字节。
	ErrCreatorTooLong = errors.New("tx: creator exceeds 255 bytes")
	// ErrTitleTooLong 表示 Title 字段超过 255 字节。
	ErrTitleTooLong = errors.New("tx: title exceeds 255 bytes")
	// ErrDescriptionTooLong 表示 Description/Content 字段超过 2KB。
	ErrDescriptionTooLong = errors.New("tx: description exceeds 2KB")
	// ErrAttachmentIDTooLong 表示附件 ID 结构超过 255 字节。
	ErrAttachmentIDTooLong = errors.New("tx: attachment id exceeds 255 bytes")
	// ErrTooManyCreditOutputs 表示单笔交易凭信输出超过 MaxCreditOutputsPerTx。
	ErrTooManyCreditOutputs = errors.New("tx: credit outputs exceed per-transaction limit")
	// ErrNoOutputs 表示普通交易输出集为空（币金输出数量须 >0）。
	ErrNoOutputs = errors.New("tx: transaction must have at least one output")
	// ErrOutputSerialMismatch 表示输出 Serial 与其在输出集中的位置下标不一致。
	ErrOutputSerialMismatch = errors.New("tx: output serial does not match its position")
	// ErrTxTooLarge 表示交易体（不含见证）尺寸超过 MaxTxSize。
	ErrTxTooLarge = errors.New("tx: transaction size exceeds maximum")
	// ErrCoinAmountDecode 表示币金输出 Payload 的前导 Amount varint 解码失败。
	ErrCoinAmountDecode = errors.New("tx: failed to decode coin output amount")
	// ErrCoinNotConserved 表示输出币金总额大于输入币金总额（违反币金守恒）。
	ErrCoinNotConserved = errors.New("tx: coin outputs exceed coin inputs")
	// ErrGenesisMinterPresent 表示创世 Coinbase（BlockHeight==0）非法携带 Minter。
	ErrGenesisMinterPresent = errors.New("tx: genesis coinbase must omit minter")
	// ErrMinterMissing 表示非创世 Coinbase（BlockHeight>0）缺少必填 Minter。
	ErrMinterMissing = errors.New("tx: non-genesis coinbase requires minter")
	// ErrFreeDataTooLong 表示 Coinbase FreeData 超过 255 字节。
	ErrFreeDataTooLong = errors.New("tx: coinbase free data exceeds 255 bytes")
	// ErrCoinbaseOutputNotCoin 表示 Coinbase 输出集包含非币金输出。
	ErrCoinbaseOutputNotCoin = errors.New("tx: coinbase output must be coin")
	// ErrCoinbasePosition 表示 Coinbase 不在区块交易序列首位（下标 0）。
	ErrCoinbasePosition = errors.New("tx: coinbase must be first transaction")
)
