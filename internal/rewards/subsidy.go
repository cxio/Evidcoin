package rewards

import "github.com/cxio/evidcoin/pkg/types"

// 发行曲线常量（第 14 章 §1）。

const (
	// phase2StartYear 是第二阶段正式发行期的起始年份（1-indexed）。
	phase2StartYear = 4
	// phase2StartSubsidyChx 是第二阶段起始铸币量（40 币/块，单位 chx）。
	phase2StartSubsidyChx = uint64(40) * types.ChxPerBi
	// longTermSubsidyChx 是长期微通胀铸币量（3 币/块，单位 chx）。
	longTermSubsidyChx = uint64(3) * types.ChxPerBi
)

// phase1Subsidies 是第一阶段（预发布，第 1~3 年）每块铸币量（chx），下标 = 年份 - 1。
var phase1Subsidies = [3]uint64{
	10 * types.ChxPerBi,
	20 * types.ChxPerBi,
	30 * types.ChxPerBi,
}

// Issuance 根据区块高度返回该区块铸币量（单位 chx，整数除法）。
//
// 三阶段发行曲线（第 14 章 §1、proposal 14 §1）：
//   - 第一阶段（前三年，第 1~3 年）：10/20/30 币/块逐年递增。
//   - 第二阶段（正式发行期，第 4 年起）：自 40 币/块起，每 2 年递减 20%（整数除法）。
//   - 长期微通胀：递减到 < 3 币/块时，固定 300_000_000 chx/块（即 3 币/块）。
//
// 以 chx 整数除法为准，禁止浮点中间值。
func Issuance(blockHeight uint32) types.Amount {
	// 年份从 1 开始；区块 0 属第 1 年。
	year := int(blockHeight)/types.BlocksPerYear + 1

	if year <= 3 {
		return types.Amount(phase1Subsidies[year-1])
	}

	// 第二阶段：从第 4 年起，每 2 年递减 20%（一个 period = 2 年）。
	period := (year - phase2StartYear) / 2
	subsidy := phase2StartSubsidyChx
	for i := 0; i < period; i++ {
		subsidy = subsidy * 80 / 100
	}
	if subsidy < longTermSubsidyChx {
		return types.Amount(longTermSubsidyChx)
	}
	return types.Amount(subsidy)
}
