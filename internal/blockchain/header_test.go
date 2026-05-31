package blockchain

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// mustHash48 在测试中由原始字节构造 Hash48，长度错误即失败。
func mustHash48(t *testing.T, b []byte) types.Hash48 {
	t.Helper()
	h, err := types.NewHash48(b)
	if err != nil {
		t.Fatalf("NewHash48: %v", err)
	}
	return h
}

// mustCheckRoot 在测试中由原始字节构造 CheckRoot。
func mustCheckRoot(t *testing.T, b []byte) types.CheckRoot {
	t.Helper()
	r, err := types.NewCheckRoot(b)
	if err != nil {
		t.Fatalf("NewCheckRoot: %v", err)
	}
	return r
}

// TestBlockHeaderCanonicalBytesLayout 校验常规（非年块）区块头的字段顺序与尺寸。
func TestBlockHeaderCanonicalBytesLayout(t *testing.T) {
	h := &BlockHeader{
		Version:   1,
		Height:    100, // 非年度边界
		PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0x11}, 48)),
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x22}, 48)),
		Stakes:    0xDEADBEEFCAFE,
	}

	var want []byte
	want = binary.BigEndian.AppendUint32(want, 1)
	want = binary.BigEndian.AppendUint32(want, 100)
	want = append(want, bytes.Repeat([]byte{0x11}, 48)...)
	want = append(want, bytes.Repeat([]byte{0x22}, 48)...)
	want = binary.BigEndian.AppendUint64(want, 0xDEADBEEFCAFE)

	got := h.CanonicalBytes()
	if len(got) != 112 {
		t.Fatalf("常规区块头尺寸 = %d, 期望 112", len(got))
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("CanonicalBytes 不匹配手工拼接向量\n got=%x\nwant=%x", got, want)
	}
}

// TestBlockHeaderCanonicalBytesYearBlock 校验年块多出 48 字节 YearBlock 字段。
func TestBlockHeaderCanonicalBytesYearBlock(t *testing.T) {
	yb := bytes.Repeat([]byte{0x33}, 48)
	h := &BlockHeader{
		Version:   1,
		Height:    types.BlocksPerYear, // 年度边界
		PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0x11}, 48)),
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x22}, 48)),
		Stakes:    7,
		YearBlock: mustHash48(t, yb),
	}

	got := h.CanonicalBytes()
	if len(got) != 160 {
		t.Fatalf("年块区块头尺寸 = %d, 期望 160", len(got))
	}
	if !bytes.Equal(got[112:], yb) {
		t.Fatalf("YearBlock 字段不匹配\n got=%x\nwant=%x", got[112:], yb)
	}
}

// TestBlockHeaderGenesisIsYearBlock 校验创世（高度 0）为年块：YearBlock 存在但全零，尺寸 160。
func TestBlockHeaderGenesisIsYearBlock(t *testing.T) {
	h := &BlockHeader{
		Version:   1,
		Height:    0,
		PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0x00}, 48)),
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x22}, 48)),
		Stakes:    0,
		// YearBlock 留零值
	}
	if !h.IsYearBlock() {
		t.Fatal("高度 0 应为年块")
	}
	got := h.CanonicalBytes()
	if len(got) != 160 {
		t.Fatalf("创世区块头尺寸 = %d, 期望 160", len(got))
	}
	if !bytes.Equal(got[112:], make([]byte, 48)) {
		t.Fatalf("创世 YearBlock 应全零, got=%x", got[112:])
	}
}

// TestBlockHeaderCanonicalBytesDeterministic 校验相同字段得到相同字节。
func TestBlockHeaderCanonicalBytesDeterministic(t *testing.T) {
	build := func() *BlockHeader {
		return &BlockHeader{
			Version:   1,
			Height:    100,
			PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0x11}, 48)),
			CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x22}, 48)),
			Stakes:    42,
		}
	}
	if !bytes.Equal(build().CanonicalBytes(), build().CanonicalBytes()) {
		t.Fatal("相同字段的 CanonicalBytes 不一致")
	}
}
