package rewards

// ServiceIndex 是公共服务槽索引（DEC-0401）。
type ServiceIndex int

const (
	// ServiceBlockqs 是区块交易查询服务（第 15 章），槽位字节 0~5。
	ServiceBlockqs ServiceIndex = 0
	// ServiceDepots 是交易附件与完整区块存储服务（第 15 章），槽位字节 6~11。
	ServiceDepots ServiceIndex = 1
	// ServiceSTUN 是 NAT 探测与打洞协助服务，槽位字节 12~17。
	ServiceSTUN ServiceIndex = 2
)

// slotServiceSize 是每个服务占用的字节数（6 字节 = 48 bit，覆盖前 48 块）。
const slotServiceSize = 6

// AwardSlots 是 Coinbase 头中的公共服务兑奖槽（固定 18 字节，DEC-0401）。
//
// 三个服务各占 6 字节（48 bit），顺序：Blockqs(0~5)、Depots(6~11)、STUN(12~17)。
// bit 编号从 0 起，采用字节内 LSB 优先：bit0 对应 H-1，bit47 对应 H-48。
// 创世交易与百日前 Coinbase 因无公共服务激励，其值恒为全零，但字段本身始终存在。
type AwardSlots [18]byte

// SetBit 将服务 svc 在前向偏移 offset（1~48）对应的 bit 置 1。
//
// offset=1 表示当前块的前 1 个块（H-1），offset=48 表示 H-48。
// offset 超出 [1,48] 范围或 svc 超出 [0,2] 时忽略。
func (a *AwardSlots) SetBit(svc ServiceIndex, offset int) {
	if offset < 1 || offset > 48 || svc < 0 || svc > 2 {
		return
	}
	n := offset - 1                    // 0-indexed bit 编号
	base := int(svc) * slotServiceSize // 服务的起始字节偏移
	a[base+n/8] |= 1 << (n % 8)
}

// IsSet 检查服务 svc 在前向偏移 offset 对应的 bit 是否已置 1。
//
// offset 超出 [1,48] 范围或 svc 超出 [0,2] 时返回 false。
func (a *AwardSlots) IsSet(svc ServiceIndex, offset int) bool {
	if offset < 1 || offset > 48 || svc < 0 || svc > 2 {
		return false
	}
	n := offset - 1
	base := int(svc) * slotServiceSize
	return (a[base+n/8]>>(n%8))&1 == 1
}

// CountForService 统计兑奖槽中服务 svc 被置 1 的 bit 总数（最大 48）。
// svc 超出 [0,2] 时返回 0。
func (a *AwardSlots) CountForService(svc ServiceIndex) int {
	if svc < 0 || svc > 2 {
		return 0
	}
	count := 0
	for offset := 1; offset <= 48; offset++ {
		if a.IsSet(svc, offset) {
			count++
		}
	}
	return count
}

// IsZero 报告兑奖槽是否为全零（创世与百日前 Coinbase 的约束值）。
func (a *AwardSlots) IsZero() bool {
	for _, b := range a {
		if b != 0 {
			return false
		}
	}
	return true
}
