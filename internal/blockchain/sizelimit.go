package blockchain

import "github.com/cxio/evidcoin/pkg/types"

// 区块尺寸限额曲线（第 05 章 §8 / conception #区块限额）。
// 限额仅约束区块数据量（尺寸）：数据包含解锁脚本，但不含见证信息
// （定制验证放入解锁脚本的签名数据计入）。本层只给出限额数值，不在此处统计实际区块尺寸。

const (
	// mebiByte 为限额单位 1MB，取 1 MiB = 1<<20 字节。
	mebiByte = 1 << 20

	// blocksPerMonth 为月块数（87661/12 ≈ 7305）。前 11 个月各 7305 块，
	// 末月容纳尾数 7306 块（7305×11 + 7306 = 87661），以补足完整一年。
	blocksPerMonth = 7305

	// firstYearCapMB 为第一年末（第 12 月）的限额上限，单位 MB。
	firstYearCapMB = 10
)

// BlockSizeLimit 返回给定区块高度的尺寸限额（字节），尺寸口径含解锁脚本、不含见证。
//
// 曲线（第 05 章 §8）：
//   - 第 1~3 月（块 0..21914）：固定 1MB。
//   - 第 4~12 月：逐月递增 1MB，至第 12 月达 10MB。
//   - 第 2 年起：按恒星年（BlocksPerYear=87661）逐年递增 1MB。
func BlockSizeLimit(height uint32) int {
	year := height / types.BlocksPerYear
	if year == 0 {
		return firstYearLimitMB(height) * mebiByte
	}
	// 第 2 年（year==1）在第一年封顶 10MB 基础上 +1MB，其后逐年递增 1MB。
	return (firstYearCapMB + int(year)) * mebiByte
}

// firstYearLimitMB 计算第一年内（height < BlocksPerYear）的限额，单位 MB。
// monthIdx 取 0..11：前 3 个月（monthIdx 0..2）固定 1MB；
// 自第 4 月（monthIdx 3）起逐月递增，第 12 月（monthIdx 11）达 10MB。
//
// 末月（第 12 月）含尾数块 7306，区间为 [80355, 87660]，其最后一块
// 87660 == 7305×12 恰好整除，使 height/blocksPerMonth 溢出为 12，故需
// 将 monthIdx 钳制到 11（第 12 月），避免第一年末块被误算为更高限额。
func firstYearLimitMB(height uint32) int {
	monthIdx := height / blocksPerMonth
	if monthIdx > 11 {
		monthIdx = 11
	}
	if monthIdx <= 2 {
		return 1
	}
	return int(monthIdx) - 1
}
