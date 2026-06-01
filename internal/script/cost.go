package script

// Budget 封装脚本执行的成本预算（DEC-0504）。
// 成本具体数值由 C-6 裁决，当前实现以占位等级计算；
// limit=0 表示不限制（用于测试）。
type Budget struct {
	current int64
	limit   int64
}

// NewBudget 创建成本预算，limit=0 表示不限制。
func NewBudget(limit int64) *Budget { return &Budget{limit: limit} }

// Consume 消费指定成本点数。若超出上限（limit > 0）返回 ErrCostExceeded。
func (b *Budget) Consume(cost int64) error {
	b.current += cost
	if b.limit > 0 && b.current > b.limit {
		return ErrCostExceeded
	}
	return nil
}

// Current 返回当前已消费的成本。
func (b *Budget) Current() int64 { return b.current }

// Limit 返回成本上限（0=不限制）。
func (b *Budget) Limit() int64 { return b.limit }

// costForTier 将成本等级转换为成本点数（C-6 占位实现）。
// C-6 裁决前所有非免费等级以 1 占位；裁决后替换此函数。
func costForTier(tier CostTier) int64 {
	// C-6 待决：成本数值未定，以占位值实现（不得当作协议事实固化）
	if tier == CostTierFree {
		return 0
	}
	return 1 // 占位
}
