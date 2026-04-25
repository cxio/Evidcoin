package blockchain_test

import (
	"testing"

	"github.com/cxio/evidcoin/internal/blockchain"
	"github.com/cxio/evidcoin/pkg/types"
)

// makeGenesis 创建测试用创世区块头。
func makeGenesis() *blockchain.BlockHeader {
	return &blockchain.BlockHeader{
		Version:   1,
		Height:    0,
		PrevBlock: types.Hash384{}, // 创世块前一哈希为零
		CheckRoot: types.Hash384{1},
		Stakes:    0,
	}
}

// makeNext 在 parent 后创建合法的下一区块头。
func makeNext(parent *blockchain.BlockHeader) *blockchain.BlockHeader {
	return &blockchain.BlockHeader{
		Version:   1,
		Height:    parent.Height + 1,
		PrevBlock: parent.Hash(),
		CheckRoot: types.Hash384{byte(parent.Height + 2)},
		Stakes:    100,
	}
}

// TestSubmitBlockSuccess 测试正常提交区块。
func TestSubmitBlockSuccess(t *testing.T) {
	genesis := makeGenesis()
	bc, err := blockchain.New(blockchain.Config{Genesis: genesis})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	block1 := makeNext(genesis)
	if err := bc.SubmitBlock(block1); err != nil {
		t.Errorf("SubmitBlock() error = %v", err)
	}
}

// TestSubmitBlockConflict 测试同高度提交返回 ErrConflict。
func TestSubmitBlockConflict(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	block1 := makeNext(genesis)
	bc.SubmitBlock(block1)

	// 再次提交同一区块
	err := bc.SubmitBlock(block1)
	if err == nil {
		t.Error("expected ErrConflict, got nil")
	}
}

// TestSubmitBlockInvalidPrevBlock 测试 PrevBlock 不匹配被拒绝。
func TestSubmitBlockInvalidPrevBlock(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	bad := &blockchain.BlockHeader{
		Version:   1,
		Height:    1,
		PrevBlock: types.Hash384{0xff}, // 错误的前一哈希
		CheckRoot: types.Hash384{1},
	}
	err := bc.SubmitBlock(bad)
	if err == nil {
		t.Error("expected error for invalid PrevBlock, got nil")
	}
}

// TestSubmitBlockInvalidHeight 测试非连续高度被拒绝。
func TestSubmitBlockInvalidHeight(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	bad := &blockchain.BlockHeader{
		Version:   1,
		Height:    5, // 应为 1
		PrevBlock: genesis.Hash(),
		CheckRoot: types.Hash384{1},
	}
	err := bc.SubmitBlock(bad)
	if err == nil {
		t.Error("expected error for invalid height, got nil")
	}
}

// TestChainHeightAfterSubmit 测试提交后链高度正确更新。
func TestChainHeightAfterSubmit(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	if h := bc.ChainHeight(); h != 0 {
		t.Errorf("initial height = %d, want 0", h)
	}
	bc.SubmitBlock(makeNext(genesis))
	if h := bc.ChainHeight(); h != 1 {
		t.Errorf("height after submit = %d, want 1", h)
	}
}

// TestHeaderByHeight 测试按高度查询区块头。
func TestHeaderByHeight(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})
	h, err := bc.HeaderByHeight(0)
	if err != nil {
		t.Fatalf("HeaderByHeight(0) error = %v", err)
	}
	if h.Height != 0 {
		t.Errorf("Height = %d, want 0", h.Height)
	}
}

// TestHeaderByHash 测试按哈希查询区块头。
func TestHeaderByHash(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})
	hash := genesis.Hash()
	h, err := bc.HeaderByHash(hash)
	if err != nil {
		t.Fatalf("HeaderByHash() error = %v", err)
	}
	if h.Hash() != hash {
		t.Error("HeaderByHash returned wrong header")
	}
}

// TestBlockTimeDeterminism 测试区块时间戳计算确定性。
func TestBlockTimeDeterminism(t *testing.T) {
	genesis := int64(1700000000000)
	t1 := blockchain.BlockTime(genesis, 0)
	t2 := blockchain.BlockTime(genesis, 0)
	if t1 != t2 {
		t.Error("BlockTime is not deterministic")
	}
	t3 := blockchain.BlockTime(genesis, 1)
	expected := genesis + int64(types.BlockInterval.Milliseconds())
	if t3 != expected {
		t.Errorf("BlockTime(1) = %d, want %d", t3, expected)
	}
}

// TestIsYearBlock 测试年块判断逻辑。
func TestIsYearBlock(t *testing.T) {
	cases := []struct {
		height int32
		want   bool
	}{
		{0, false},
		{1, false},
		{int32(types.BlocksPerYear), true},
		{int32(types.BlocksPerYear * 2), true},
		{int32(types.BlocksPerYear) + 1, false},
	}
	for _, tc := range cases {
		h := &blockchain.BlockHeader{Height: tc.height}
		if got := h.IsYearBlock(); got != tc.want {
			t.Errorf("Height %d IsYearBlock() = %v, want %v", tc.height, got, tc.want)
		}
	}
}

// TestSubscribeNewBlock 测试新区块订阅通知。
func TestSubscribeNewBlock(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	ch := make(chan *blockchain.BlockHeader, 1)
	bc.Subscribe(ch)

	block1 := makeNext(genesis)
	bc.SubmitBlock(block1)

	select {
	case received := <-ch:
		if received.Height != 1 {
			t.Errorf("received block height = %d, want 1", received.Height)
		}
	default:
		t.Error("expected notification on channel, got none")
	}
}

// TestSwitchChainValidates 测试 SwitchChain 验证区块头连续性。
func TestSwitchChainValidates(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	// 构造一个不连续的替代链
	bad := []*blockchain.BlockHeader{
		{Version: 1, Height: 1, PrevBlock: genesis.Hash(), CheckRoot: types.Hash384{9}},
		{Version: 1, Height: 3, PrevBlock: types.Hash384{0xab}, CheckRoot: types.Hash384{10}}, // 高度不连续
	}
	err := bc.SwitchChain(0, bad)
	if err == nil {
		t.Error("expected error for non-contiguous fork chain, got nil")
	}
}

// TestSwitchChainSuccess 测试 SwitchChain 正常切换。
func TestSwitchChainSuccess(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	h1 := &blockchain.BlockHeader{Version: 1, Height: 1, PrevBlock: genesis.Hash(), CheckRoot: types.Hash384{9}}
	h2 := &blockchain.BlockHeader{Version: 1, Height: 2, PrevBlock: h1.Hash(), CheckRoot: types.Hash384{10}}
	err := bc.SwitchChain(0, []*blockchain.BlockHeader{h1, h2})
	if err != nil {
		t.Errorf("SwitchChain() error = %v", err)
	}
}

// TestChainTip 测试链顶查询。
func TestChainTip(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})
	tip, err := bc.ChainTip()
	if err != nil {
		t.Fatalf("ChainTip() error = %v", err)
	}
	if tip.Height != 0 {
		t.Errorf("ChainTip().Height = %d, want 0", tip.Height)
	}
}

// TestIdentity 测试链标识获取。
func TestIdentity(t *testing.T) {
	identity := &blockchain.ChainIdentity{
		ProtocolID: "Evidcoin@V1",
		ChainID:    "mainnet",
	}
	bc, _ := blockchain.New(blockchain.Config{
		Genesis:  makeGenesis(),
		Identity: identity,
	})
	id := bc.Identity()
	if id == nil || id.ChainID != "mainnet" {
		t.Error("Identity() returned wrong value")
	}
}

// TestReplaceBlock 测试链顶区块替换。
func TestReplaceBlock(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})
	block1 := makeNext(genesis)
	bc.SubmitBlock(block1)

	// 用另一个区块替换 height=1
	replacement := &blockchain.BlockHeader{
		Version: 1, Height: 1, PrevBlock: genesis.Hash(), CheckRoot: types.Hash384{0xcc},
	}
	err := bc.ReplaceBlock(1, replacement)
	if err != nil {
		t.Errorf("ReplaceBlock() error = %v", err)
	}

	// 替换高度不是链顶的区块应该失败
	err = bc.ReplaceBlock(0, genesis)
	if err == nil {
		t.Error("expected error replacing non-tip block, got nil")
	}
}

// TestSyncHeaders 测试 SyncHeaders（noop 客户端返回错误）。
func TestSyncHeaders(t *testing.T) {
	bc, _ := blockchain.New(blockchain.Config{Genesis: makeGenesis()})
	err := bc.SyncHeaders(1, 10)
	if err == nil {
		t.Error("expected ErrBlockqsUnavailable from noop client, got nil")
	}
}

// TestDetectFork 测试 DetectFork stub 返回 nil。
func TestDetectFork(t *testing.T) {
	bc, _ := blockchain.New(blockchain.Config{Genesis: makeGenesis()})
	info, err := bc.DetectFork([]string{"peer1"})
	if err != nil || info != nil {
		t.Errorf("DetectFork() = %v, %v; want nil, nil", info, err)
	}
}

// TestBootstrapVerify 测试 BootstrapVerify stub。
func TestBootstrapVerify(t *testing.T) {
	bc, _ := blockchain.New(blockchain.Config{Genesis: makeGenesis()})
	result, err := bc.BootstrapVerify(nil)
	if err != nil {
		t.Fatalf("BootstrapVerify() error = %v", err)
	}
	if result.Height != 0 {
		t.Errorf("BootstrapVerify().Height = %d, want 0", result.Height)
	}
	if result.Confidence != 1.0 {
		t.Errorf("Confidence = %v, want 1.0", result.Confidence)
	}
}

// TestGenesisID 测试创世区块 ID 获取。
func TestGenesisID(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})
	id, err := bc.GenesisID()
	if err != nil {
		t.Fatalf("GenesisID() error = %v", err)
	}
	if id != genesis.Hash() {
		t.Error("GenesisID() returned wrong hash")
	}
}

// TestBoundIDFromHeader 测试 BoundID 计算。
func TestBoundIDFromHeader(t *testing.T) {
	genesis := makeGenesis()
	bound := blockchain.BoundIDFromHeader(genesis)
	if len(bound) != 20 {
		t.Errorf("BoundID length = %d, want 20", len(bound))
	}
}

// TestUpdateBoundID 测试更新链标识 BoundID。
func TestUpdateBoundID(t *testing.T) {
	identity := &blockchain.ChainIdentity{ProtocolID: "Evidcoin@V1", ChainID: "testnet"}
	bc, _ := blockchain.New(blockchain.Config{
		Genesis:  makeGenesis(),
		Identity: identity,
	})
	err := bc.UpdateBoundID(0)
	if err != nil {
		t.Fatalf("UpdateBoundID() error = %v", err)
	}
	if len(identity.BoundID) != 20 {
		t.Errorf("BoundID length = %d, want 20", len(identity.BoundID))
	}
}

// TestChainIdentityBytes 测试链标识序列化。
func TestChainIdentityBytes(t *testing.T) {
	ci := &blockchain.ChainIdentity{
		ProtocolID: "Evidcoin@V1",
		ChainID:    "mainnet",
		GenesisID:  types.Hash384{1, 2, 3},
		BoundID:    []byte{0xaa, 0xbb},
	}
	b := ci.Bytes()
	if len(b) == 0 {
		t.Error("ChainIdentity.Bytes() returned empty slice")
	}
}

// TestHeaderBytesYearBlock 测试年块的序列化长度。
func TestHeaderBytesYearBlock(t *testing.T) {
	h := &blockchain.BlockHeader{
		Version:   1,
		Height:    int32(types.BlocksPerYear),
		CheckRoot: types.Hash384{1},
	}
	b := h.Bytes()
	if len(b) != 160 {
		t.Errorf("year block Bytes() len = %d, want 160", len(b))
	}
}

// TestSubmitBlockInvalidVersion 测试无效版本号被拒绝。
func TestSubmitBlockInvalidVersion(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	bad := &blockchain.BlockHeader{
		Version:   99,
		Height:    1,
		PrevBlock: genesis.Hash(),
		CheckRoot: types.Hash384{1},
	}
	err := bc.SubmitBlock(bad)
	if err == nil {
		t.Error("expected error for invalid version, got nil")
	}
}

// TestSubmitBlockInvalidCheckRoot 测试零 CheckRoot 被拒绝。
func TestSubmitBlockInvalidCheckRoot(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	bad := &blockchain.BlockHeader{
		Version:   1,
		Height:    1,
		PrevBlock: genesis.Hash(),
		CheckRoot: types.Hash384{}, // 零值
	}
	err := bc.SubmitBlock(bad)
	if err == nil {
		t.Error("expected ErrInvalidCheckRoot, got nil")
	}
}

// TestHeaderByHeightFallback 测试本地缺失时回退到 Blockqs（noop 返回错误）。
func TestHeaderByHeightFallback(t *testing.T) {
	bc, _ := blockchain.New(blockchain.Config{Genesis: makeGenesis()})
	_, err := bc.HeaderByHeight(999)
	if err == nil {
		t.Error("expected error for missing height from blockqs, got nil")
	}
}

// TestHeaderByHashFallback 测试按哈希本地缺失时回退到 Blockqs。
func TestHeaderByHashFallback(t *testing.T) {
	bc, _ := blockchain.New(blockchain.Config{Genesis: makeGenesis()})
	_, err := bc.HeaderByHash(types.Hash384{0xde, 0xad})
	if err == nil {
		t.Error("expected error for missing hash from blockqs, got nil")
	}
}
