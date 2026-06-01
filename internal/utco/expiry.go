package utco

// ExpireAt 在区块结算时清理过期凭信（第 09 章 §6、第 07 章 §5）：
//
//  1. 删除所有过期（未转出但 age > CreditMaxAge）的有效凭信，其状态位失效。
//  2. 对因此再无任何有效（未转出且未过期）凭信的 TxID 分组，删除其残留叶——
//     即清除该 TxID 下仍留存的已转出输出，使该 TxID 不再产生状态指纹叶。
//
// 同一 TxID 仍存在其它未转出且未过期凭信时，仅删除过期输出并保留叶（Count 递减）。
//
// 返回因过期而被删除的有效凭信数量（步骤一计数；残留叶清理不计入）。
func (s *Store) ExpireAt(currentHeight uint32) int {
	removed := 0
	// 步骤一：删除过期的有效凭信。
	for op, e := range s.entries {
		if !e.Spent && e.Expired(currentHeight) {
			delete(s.entries, op)
			removed++
		}
	}
	// 步骤二：清理再无有效凭信的 TxID 残留叶。
	hasValid := make(map[OutPoint]bool) // 以 Year+TxID 标识分组（OutIndex 置 0）
	for _, e := range s.entries {
		if !e.Spent {
			hasValid[groupKey(e)] = true
		}
	}
	for op, e := range s.entries {
		if !hasValid[groupKey(e)] {
			delete(s.entries, op)
		}
	}
	return removed
}

// groupKey 返回 entry 所属 TxID 分组的标识键（Year + TxID，OutIndex 归零）。
func groupKey(e Entry) OutPoint {
	return OutPoint{Year: e.Year, TxID: e.TxID}
}
