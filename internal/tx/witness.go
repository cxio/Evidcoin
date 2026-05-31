package tx

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// 见证容器与剪枝（DEC-0103 §6-§7）。每个输入拥有一个见证容器，见证不计入 TxID，
// 普通交易输入见证可整体剪枝；解锁脚本参与输入根（计入交易体），不属于可剪枝见证。

// 见证容器错误。
var (
	// ErrWitnessItemType 表示见证 item 类型字节不在标准 6 类（0x01-0x06）之内。
	ErrWitnessItemType = errors.New("tx: unknown witness item type")
	// ErrWitnessMalformed 表示见证容器字节截断或存在多余尾随字节。
	ErrWitnessMalformed = errors.New("tx: malformed witness container")
)

// WitnessItemType 是见证项类型字节（DEC-0103 §6）。
type WitnessItemType byte

const (
	// WitCategory 是验证类别（0x01）。
	WitCategory WitnessItemType = 0x01
	// WitAuthFlag 是授权标记（0x02）。
	WitAuthFlag WitnessItemType = 0x02
	// WitSignature 是签名（0x03）。
	WitSignature WitnessItemType = 0x03
	// WitPublicKey 是公钥（0x04）。
	WitPublicKey WitnessItemType = 0x04
	// WitBaseHash 是补全公钥哈希（0x05）。
	WitBaseHash WitnessItemType = 0x05
	// WitExternal 是解锁脚本外部数据（0x06）。
	WitExternal WitnessItemType = 0x06
)

// validWitnessItemType 报告 t 是否为标准 6 类之一。
func validWitnessItemType(t WitnessItemType) bool {
	return t >= WitCategory && t <= WitExternal
}

// WitnessItem 是单个见证项（DEC-0103 §6）：type(byte) || data。
// data 以 varint(len)||bytes 编码以保证容器自界定（DEC-0001 长度前缀约定）。
type WitnessItem struct {
	// Type 是见证项类型。
	Type WitnessItemType
	// Data 是见证项数据字节。
	Data []byte
}

// Witness 是单个输入的见证容器（DEC-0103 §6）。见证容器不进入 TxID 计算，
// 可整体剪枝；缺失见证是否导致验证失败由脚本执行 SYS_CHKPASS 时判定（第 06 章）。
type Witness struct {
	// Items 是见证项序列，按构造者给定顺序排列，容器不重排（多签签名/公钥配对顺序见 DEC-0103 §9）。
	Items []WitnessItem
}

// Encode 返回见证容器的规范编码：varint(item_count) || (type || varint(len)||data)*。
// 任一 item 类型非法时返回 ErrWitnessItemType。空容器编码为单字节 0x00。
func (w Witness) Encode() ([]byte, error) {
	dst := types.AppendVarUint(nil, uint64(len(w.Items)))
	for _, it := range w.Items {
		if !validWitnessItemType(it.Type) {
			return nil, ErrWitnessItemType
		}
		dst = append(dst, byte(it.Type))
		dst = types.AppendBytes(dst, it.Data)
	}
	return dst, nil
}

// DecodeWitness 从 src 前缀解析一个见证容器，返回解析结果与已消费字节数。
// 类型非法、字节截断或计数不符时返回相应错误。
func DecodeWitness(src []byte) (Witness, int, error) {
	count, n, err := types.ReadVarUint(src)
	if err != nil {
		return Witness{}, 0, err
	}
	off := n
	items := make([]WitnessItem, 0, count)
	for i := uint64(0); i < count; i++ {
		if off >= len(src) {
			return Witness{}, 0, ErrWitnessMalformed
		}
		t := WitnessItemType(src[off])
		if !validWitnessItemType(t) {
			return Witness{}, 0, ErrWitnessItemType
		}
		off++
		data, dn, derr := types.ReadBytes(src[off:])
		if derr != nil {
			return Witness{}, 0, derr
		}
		off += dn
		items = append(items, WitnessItem{Type: t, Data: data})
	}
	return Witness{Items: items}, off, nil
}

// Prune 返回该输入见证的剪枝结果（DEC-0103 §7）。普通交易输入见证整体可剪枝，
// 剪枝后返回空见证；解锁脚本不在见证容器内（已计入交易体、参与输入根），故不受影响。
//
// 注意：以下签名不属于可剪枝见证，必须由各自承载结构保留，不经本函数处理：
//   - 择优凭证中对 mintHash 的签名（随 Coinbase Minter 数据保留，见第 11 章）；
//   - 创世区块铸造者对 CheckRoot 的签名（链根锚定，见 coinbase_sig.go）；
//   - Coinbase 普通交易签名采用分层保存，长期共识最小验证不依赖（DEC-0103）。
func (w Witness) Prune() Witness {
	return Witness{}
}
