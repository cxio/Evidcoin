package types

import "time"

// BlockHeight 表示区块高度。其底层类型固定为 uint32，以匹配区块头 Height 字段
// （DEC-0001 定宽白名单）；任何参与区块头哈希/签名的高度值都必须通过
// AppendUint32BE 进行编码。
type BlockHeight uint32

// HeightYear 返回 h 的“按高度划分”的年份索引，即 h / BlocksPerYear。
// 这是年块使用的 87661 区块年边界；它不同于交易短引用使用的 UTC
// CalendarYear，二者不可混用。
func HeightYear(h BlockHeight) uint32 {
	return uint32(h) / BlocksPerYear
}

// IsYearBoundary 判断高度 h 是否落在按高度划分的年边界上
// （即 BlocksPerYear 的整数倍）。创世高度 0 也属于年边界。
func IsYearBoundary(h BlockHeight) bool {
	return uint32(h)%BlocksPerYear == 0
}

// BlockTime 返回高度 h 区块的规范时间戳，计算方式为
// genesis 时间加上 h 个出块间隔。
func BlockTime(genesis time.Time, h BlockHeight) time.Time {
	return genesis.Add(time.Duration(h) * BlockInterval)
}
