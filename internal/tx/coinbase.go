package tx

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// maxFreeData 是 Coinbase FreeData 自由数据的最大长度（<256 字节，第 06 章 §2）。
const maxFreeData = 255

// awardSlotsLen 是公共服务兑奖槽字段的固定长度（第 06 章 §2、DEC-0401）。
const awardSlotsLen = 18

// CoinbaseHeader 是 Coinbase 交易头结构（第 06 章 §2、DEC-0003/DEC-0401）。
// 与普通交易头解析 profile 不共用：Coinbase 无 HashInputs 字段，MintPKHash 定长 32B。
// 本层只固定结构编码、字段边界与位置规则；奖励、销毁与兑奖槽的结算语义见第 10/14 章。
type CoinbaseHeader struct {
	// Version 是交易版本号（创世 Coinbase 为 1）。
	Version uint16
	// HashOutputs 是输出项根哈希（BLAKE3-256，见 output_hash.go）。
	HashOutputs types.Hash32
	// Timestamp 是交易时间戳（Unix 毫秒），也即区块时间戳。
	Timestamp int64
	// MintPKHash 是铸凭公钥哈希，Coinbase 必填且定长 32 字节（不使用变长封装）。
	MintPKHash [32]byte
	// BlockHeight 是交易所在区块高度，创世为 0。
	BlockHeight uint32
	// Minter 是择优凭证（MintProof）的已编码字节，由共识层（第 11 章 PoH）产出后注入；
	// 本层视为不透明序列。当且仅当 BlockHeight==0（创世）时省略（须为空），
	// 其余高度必须存在（非空），无额外 presence 标识。
	Minter []byte
	// FreeData 是自由数据（<256 字节）。
	FreeData []byte
	// BurnCoin 是销毁交易费量（单位 chx，语义见第 14 章；销毁单点化于此）。
	BurnCoin int64
	// AwardSlots 是公共服务兑奖槽（固定 18 字节）；对所有 Coinbase（含创世）始终存在，
	// 创世与百日前 Coinbase 因无公共服务激励其值恒为全零（bit 语义见第 14 章）。
	AwardSlots [awardSlotsLen]byte
}

// CanonicalBytes 返回 Coinbase 交易头的规范编码（第 06 章 §2、DEC-0401）：
//
//	Version(uint16 BE) || HashOutputs[32] || Timestamp(int64 BE) || MintPKHash[32]
//	  || BlockHeight(uint32 BE) || [Minter if BlockHeight>0]
//	  || FreeData(varint(len)||bytes) || BurnCoin(int64 BE) || AwardSlots[18]
//
// 无 HashInputs 字段。当创世携带 Minter、非创世缺少 Minter，或 FreeData 超过 255 字节时
// 返回相应错误。Minter 作为不透明字节原样追加（其内部自界定编码由共识层保证）。
func (h *CoinbaseHeader) CanonicalBytes() ([]byte, error) {
	genesis := h.BlockHeight == 0
	if genesis && len(h.Minter) != 0 {
		return nil, ErrGenesisMinterPresent
	}
	if !genesis && len(h.Minter) == 0 {
		return nil, ErrMinterMissing
	}
	if len(h.FreeData) > maxFreeData {
		return nil, ErrFreeDataTooLong
	}
	dst := make([]byte, 0, 128)
	dst = types.AppendUint16BE(dst, h.Version)
	dst = append(dst, h.HashOutputs.Bytes()...)
	dst = types.AppendInt64BE(dst, h.Timestamp)
	dst = append(dst, h.MintPKHash[:]...)
	dst = types.AppendUint32BE(dst, h.BlockHeight)
	if !genesis {
		dst = append(dst, h.Minter...)
	}
	dst = types.AppendBytes(dst, h.FreeData)
	dst = types.AppendInt64BE(dst, h.BurnCoin)
	dst = append(dst, h.AwardSlots[:]...)
	return dst, nil
}

// TxID 计算 Coinbase 交易头的交易标识（第 06 章 §1.1）：
//
//	TxID = SHA3-384( DomainTag("tx.header") || CanonicalBytes() )
//
// 与普通头共用 tx.header 域标签，但前像为 Coinbase 字段集（解析 profile 不共用）。
func (h *CoinbaseHeader) TxID() (types.TxID, error) {
	pre, err := h.CanonicalBytes()
	if err != nil {
		return types.TxID{}, err
	}
	return crypto.HashTxHeader(pre), nil
}

// ValidateCoinbaseOutputs 校验 Coinbase 输出集仅含币金输出（DEC-0401）。
// 出现凭信/存证/自定义（含介管脚本）输出时返回 ErrCoinbaseOutputNotCoin。
func ValidateCoinbaseOutputs(outputs []Output) error {
	for i := range outputs {
		o := outputs[i]
		if o.IsCustom || o.Type != TypeCoin {
			return ErrCoinbaseOutputNotCoin
		}
	}
	return nil
}

// ValidateCoinbasePosition 校验 Coinbase 位于区块交易序列首位（下标 0，第 06 章 §2）。
// 非 0 位置返回 ErrCoinbasePosition。
func ValidateCoinbasePosition(index int) error {
	if index != 0 {
		return ErrCoinbasePosition
	}
	return nil
}
