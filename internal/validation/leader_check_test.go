package validation

import (
	"testing"
	"time"

	"github.com/cxio/evidcoin/internal/tx"
)

// makeOutPoint 构造一个测试用 tx.OutPoint（TxIDPart 填充 16 字节）。
func makeOutPoint(year uint64, tag byte, outIndex uint64) tx.OutPoint {
	part := make([]byte, 16)
	part[0] = tag
	return tx.OutPoint{Year: year, TxIDPart: part, OutIndex: outIndex}
}

// mapStakeLookup 是基于 map 的 CoinStakeLookup 桩实现。
type mapStakeLookup struct {
	stakes map[string]uint64
}

func newMapStakeLookup() *mapStakeLookup {
	return &mapStakeLookup{stakes: make(map[string]uint64)}
}

func (m *mapStakeLookup) put(ref tx.OutPoint, stake uint64) {
	m.stakes[outPointKey(ref)] = stake
}

func (m *mapStakeLookup) CoinStake(ref tx.OutPoint) (uint64, bool) {
	s, ok := m.stakes[outPointKey(ref)]
	return s, ok
}

// TestCheckLeadInput_Valid 首领输入是最大币权，无黑名单，应通过。
func TestCheckLeadInput_Valid(t *testing.T) {
	lead := tx.LeadInput{Ref: makeOutPoint(2024, 0x01, 0)}
	rest := []tx.RestInput{
		{Kind: tx.InputCoin, Ref: makeOutPoint(2024, 0x02, 0)},
	}
	lookup := newMapStakeLookup()
	lookup.put(lead.Ref, 100)
	lookup.put(rest[0].Ref, 50)

	if err := CheckLeadInput(lead, rest, lookup, nil); err != nil {
		t.Fatalf("CheckLeadInput valid: unexpected error %v", err)
	}
}

// TestCheckLeadInput_EqualStake 等于首领币权的输入也合法（须严格大于才报错）。
func TestCheckLeadInput_EqualStake(t *testing.T) {
	lead := tx.LeadInput{Ref: makeOutPoint(2024, 0x01, 0)}
	rest := []tx.RestInput{
		{Kind: tx.InputCoin, Ref: makeOutPoint(2024, 0x02, 0)},
	}
	lookup := newMapStakeLookup()
	lookup.put(lead.Ref, 100)
	lookup.put(rest[0].Ref, 100) // 相等，不违规

	if err := CheckLeadInput(lead, rest, lookup, nil); err != nil {
		t.Fatalf("CheckLeadInput equal stake should be valid, got %v", err)
	}
}

// TestCheckLeadInput_NotMaxStake 其余输入币权大于首领时报错。
func TestCheckLeadInput_NotMaxStake(t *testing.T) {
	lead := tx.LeadInput{Ref: makeOutPoint(2024, 0x01, 0)}
	rest := []tx.RestInput{
		{Kind: tx.InputCoin, Ref: makeOutPoint(2024, 0x02, 0)},
	}
	lookup := newMapStakeLookup()
	lookup.put(lead.Ref, 50)
	lookup.put(rest[0].Ref, 100) // 大于首领

	err := CheckLeadInput(lead, rest, lookup, nil)
	if err != ErrLeadInputNotMaxStake {
		t.Fatalf("CheckLeadInput: want ErrLeadInputNotMaxStake, got %v", err)
	}
}

// TestCheckLeadInput_NotFound 首领输入 UTXO 不存在。
func TestCheckLeadInput_NotFound(t *testing.T) {
	lead := tx.LeadInput{Ref: makeOutPoint(2024, 0xFF, 0)}
	lookup := newMapStakeLookup() // 空表

	err := CheckLeadInput(lead, nil, lookup, nil)
	if err != ErrLeadInputNotFound {
		t.Fatalf("CheckLeadInput: want ErrLeadInputNotFound, got %v", err)
	}
}

// TestCheckLeadInput_Blacklisted 首领输入在黑名单冻结期内。
func TestCheckLeadInput_Blacklisted(t *testing.T) {
	lead := tx.LeadInput{Ref: makeOutPoint(2024, 0xAA, 0)}
	lookup := newMapStakeLookup()
	lookup.put(lead.Ref, 100)

	bl := NewBlacklist(0)
	bl.Add(lead.Ref, time.Now())

	err := CheckLeadInput(lead, nil, lookup, bl)
	if err != ErrBlacklisted {
		t.Fatalf("CheckLeadInput: want ErrBlacklisted, got %v", err)
	}
}

// TestCheckLeadInput_BlacklistExpired 黑名单已过期时不阻拦。
func TestCheckLeadInput_BlacklistExpired(t *testing.T) {
	lead := tx.LeadInput{Ref: makeOutPoint(2024, 0xBB, 0)}
	lookup := newMapStakeLookup()
	lookup.put(lead.Ref, 100)

	bl := NewBlacklist(time.Millisecond) // 极短冻结窗口
	past := time.Now().Add(-time.Second) // 确保已过期
	bl.Add(lead.Ref, past)

	if err := CheckLeadInput(lead, nil, lookup, bl); err != nil {
		t.Fatalf("CheckLeadInput with expired blacklist: unexpected error %v", err)
	}
}

// TestCheckLeadInput_NonCoinRestIgnored 非 InputCoin 的 RestInput 不参与币权比较。
func TestCheckLeadInput_NonCoinRestIgnored(t *testing.T) {
	lead := tx.LeadInput{Ref: makeOutPoint(2024, 0x01, 0)}
	// InputCredit 来源不参与首领校验币权比较
	rest := []tx.RestInput{
		{Kind: tx.InputCredit, Ref: makeOutPoint(2024, 0x02, 0)},
	}
	lookup := newMapStakeLookup()
	lookup.put(lead.Ref, 1)
	lookup.put(rest[0].Ref, 9999) // 极高，但非 InputCoin 应被忽略

	if err := CheckLeadInput(lead, rest, lookup, nil); err != nil {
		t.Fatalf("CheckLeadInput with InputCredit rest: unexpected error %v", err)
	}
}

// TestBlacklist_AddContainsPurge 检验黑名单的增删逻辑。
func TestBlacklist_AddContainsPurge(t *testing.T) {
	bl := NewBlacklist(time.Hour)
	ref := makeOutPoint(2024, 0x01, 0)

	if bl.Contains(ref) {
		t.Fatal("empty blacklist should not contain ref")
	}

	now := time.Now()
	bl.Add(ref, now)

	if !bl.Contains(ref) {
		t.Fatal("blacklist should contain ref after Add")
	}
	if bl.Len() != 1 {
		t.Fatalf("blacklist Len: got %d, want 1", bl.Len())
	}

	// Purge 不清理未过期记录
	bl.Purge(now.Add(time.Minute))
	if bl.Len() != 1 {
		t.Fatalf("after Purge(1min), Len: got %d, want 1", bl.Len())
	}

	// Purge 清理已过期记录
	bl.Purge(now.Add(2 * time.Hour))
	if bl.Len() != 0 {
		t.Fatalf("after Purge(2h), Len: got %d, want 0", bl.Len())
	}
}

// TestBlacklist_DefaultWindow 零窗口使用默认时长 LeaderCheckWindow。
func TestBlacklist_DefaultWindow(t *testing.T) {
	bl := NewBlacklist(0)
	if bl.window != LeaderCheckWindow {
		t.Fatalf("default window: got %v, want %v", bl.window, LeaderCheckWindow)
	}
}
