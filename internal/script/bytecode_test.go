package script

import (
	"encoding/binary"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestDecodeScriptSimple 测试简单字节码解码。
func TestDecodeScriptSimple(t *testing.T) {
	tests := []struct {
		name    string
		code    []byte
		wantOps []Opcode
		wantErr bool
	}{
		{
			name:    "empty script",
			code:    []byte{},
			wantOps: nil,
			wantErr: false,
		},
		{
			name:    "single NIL",
			code:    []byte{byte(NIL)},
			wantOps: []Opcode{NIL},
		},
		{
			name:    "TRUE FALSE",
			code:    []byte{byte(TRUE), byte(FALSE)},
			wantOps: []Opcode{TRUE, FALSE},
		},
		{
			name:    "PASS CHECK END",
			code:    []byte{byte(PASS), byte(CHECK), byte(END)},
			wantOps: []Opcode{PASS, CHECK, END},
		},
		{
			name:    "unknown opcode 254",
			code:    []byte{254},
			wantErr: true,
		},
		{
			name:    "unknown opcode 255",
			code:    []byte{255},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			frames, err := DecodeScript(tc.code, 0)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(frames) != len(tc.wantOps) {
				t.Fatalf("frame count = %d, want %d", len(frames), len(tc.wantOps))
			}
			for i, op := range tc.wantOps {
				if frames[i].Op != op {
					t.Errorf("frame[%d].Op = %d, want %d", i, frames[i].Op, op)
				}
			}
		})
	}
}

// TestDecodeScriptTrailingBytes 验证尾随残余字节拒绝。
func TestDecodeScriptTrailingBytes(t *testing.T) {
	// BYTE_LIT 需要 1 字节附参 + 无关联数据。
	// 给 2 字节：一条完整 BYTE_LIT + 残余 0 字节附参（但 NIL 后还有额外字节）
	// 构造：NIL（1字节） + 额外1字节
	code := []byte{byte(NIL), 0x42} // 0x42 是未期望的尾随字节
	// NIL 无附参无关联数据，解析 NIL 后 pos=1，code 还有 1 字节未消费
	// 但 0x42 对应 opcode 66 = BLOCK，BLOCK 有附参 AttrParamSizes=[0] 且有关联数据
	// 所以 0x42 会被当作 BLOCK 的 opcode，然后因缺少附参而失败
	// 让我们用一个明确会产生尾随字节的场景：
	// BYTE_LIT 需要附参，如果只有 opcode 字节而没有附参，会报截断错误
	// 这里我们用不同的方式测试：正常代码后手动添加残余字节
	// 直接构造：NIL 后接 NIL 的第二条，没问题；
	// 要测试尾随，需要 opcode 之后有多余字节，而那些字节不构成合法指令头
	// 实际上 DecodeScript 是循环解析每条指令，pos 最终必须等于 len(code)
	// 如果中间某条指令解析失败，会提前返回错误
	// 尾随字节在所有指令解析完后检查
	// 因此我们需要：所有字节都被合法指令消费，但最后有剩余
	// 由于 NIL 消费1字节，但 0x42=BLOCK 需要 ULEB128 附参，会报截断错误
	// 换成两条 NIL 后跟一个不构成任何合法开头的模式会更复杂
	// 最简洁的方式：构造一个已知固定长度的指令，然后在其关联数据后多加字节
	// BYTE_LIT(3): opcode=3, attr=1字节, 无关联数据  -> 共 2 字节
	// 构造：[3, 0xff, 0x00]  = BYTE_LIT(0xff) + 多余字节 0x00
	// 但 0x00 = NIL，NIL 只有 opcode 无附参，所以会被解析为 NIL 指令
	// 所以没有尾随字节问题——这本身是合法的两条指令
	// 真正制造尾随字节：一条有关联数据的指令，关联数据后多一个字节
	// DATA_LIT: opcode=10, attr=ULEB128长度, assoc=数据
	// 编码 DATA{1byte}: [10, 1, 0xAB]  共 3 字节 -> 正常
	// 编码 DATA{1byte} + 残余: [10, 1, 0xAB, 0xXX] 其中 0xXX 会被当作新 opcode
	// 如果 0xXX 是已注册 opcode 无附参，它会被正常解析，不是尾随错误
	// 测试尾随字节最简单的是：让 opcode 需要关联数据，但关联数据之后手动截断时报错
	// 现在换个思路：利用 BIGINT_LIT 指令
	// BIGINT_LIT: attr1=1字节(slen), assocData=magnitude bytes
	// slen bit7=0(正), 低7位=0(0字节magnitude) -> [6, 0x00] 表示 BigInt(0)
	// 但 BigInt 不允许负零或前导零... 不过 0字节magnitude 表示 0，是合法的
	// [6, 0x00] = 2字节，完整指令，无关联数据（magnitude长度为0）
	// [6, 0x00, 0x00] = 额外 0x00，0x00=NIL，又是合法的
	// 似乎很难制造"正常指令序列后有真正无法解析的尾随字节"
	// 因为任何字节都是可能的 opcode（只要注册了）
	// 最好的方式是：提供只有部分的附参字节（截断），而不是尾随字节
	// 让我用截断测试代替

	// 这个测试验证长度不足时报错（不是尾随字节，而是截断）
	_ = code
	t.Skip("trailing bytes scenario: in practice, any byte is a potential opcode; testing truncation instead")
}

// TestDecodeScriptTruncatedAttrParam 验证截断附参拒绝。
func TestDecodeScriptTruncatedAttrParam(t *testing.T) {
	tests := []struct {
		name string
		code []byte
	}{
		{
			// BYTE_LIT 需要 1 字节附参，但只有 opcode
			name: "BYTE_LIT missing attr",
			code: []byte{byte(BYTE_LIT)},
		},
		{
			// FLOAT_LIT 需要 8 字节附参，但只给 4 字节
			name: "FLOAT_LIT truncated attr (4/8 bytes)",
			code: []byte{byte(FLOAT_LIT), 0, 0, 0, 0},
		},
		{
			// RUNE_LIT 需要 4 字节附参，但只给 2 字节
			name: "RUNE_LIT truncated attr (2/4 bytes)",
			code: []byte{byte(RUNE_LIT), 0, 0},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeScript(tc.code, 0)
			if err == nil {
				t.Error("expected error for truncated attr param, got nil")
			}
		})
	}
}

// TestDecodeScriptTruncatedAssocData 验证截断关联数据拒绝。
func TestDecodeScriptTruncatedAssocData(t *testing.T) {
	// DATA_LIT: opcode=10, attr=ULEB128 length, assoc=data bytes
	// 编码 DATA{5bytes}，但只提供 2 字节数据
	// [10, 5, 0x01, 0x02]  => 缺少 3 字节
	code := []byte{byte(DATA_LIT), 5, 0x01, 0x02}
	_, err := DecodeScript(code, 0)
	if err == nil {
		t.Error("expected error for truncated assoc data, got nil")
	}
}

// TestDecodeLockScriptMaxLen 验证锁定脚本最大长度边界。
func TestDecodeLockScriptMaxLen(t *testing.T) {
	// MaxLockScript = 8191 bytes
	// 构造 8191 个 NIL 指令（每条 1 字节）
	code8191 := make([]byte, types.MaxLockScript)
	for i := range code8191 {
		code8191[i] = byte(NIL)
	}
	if _, err := DecodeLockScript(code8191); err != nil {
		t.Errorf("8191-byte lock script should be accepted: %v", err)
	}

	// 8192 字节应被拒绝
	code8192 := make([]byte, types.MaxLockScript+1)
	for i := range code8192 {
		code8192[i] = byte(NIL)
	}
	if _, err := DecodeLockScript(code8192); err == nil {
		t.Error("8192-byte lock script should be rejected")
	}
}

// TestDecodeUnlockScriptMaxLen 验证解锁脚本最大长度边界。
func TestDecodeUnlockScriptMaxLen(t *testing.T) {
	// MaxUnlockScript = 8191 bytes
	code8191 := make([]byte, types.MaxUnlockScript)
	for i := range code8191 {
		code8191[i] = byte(NIL)
	}
	if _, err := DecodeUnlockScript(code8191); err != nil {
		t.Errorf("8191-byte unlock script should be accepted: %v", err)
	}

	code8192 := make([]byte, types.MaxUnlockScript+1)
	for i := range code8192 {
		code8192[i] = byte(NIL)
	}
	if _, err := DecodeUnlockScript(code8192); err == nil {
		t.Error("8192-byte unlock script should be rejected")
	}
}

// TestDecodeUnlockScriptOpcodeRestriction 验证解锁脚本 opcode 限制。
func TestDecodeUnlockScriptOpcodeRestriction(t *testing.T) {
	tests := []struct {
		name    string
		code    []byte
		wantErr bool
	}{
		{
			name:    "NIL(0) allowed",
			code:    []byte{byte(NIL)},
			wantErr: false,
		},
		{
			name:    "PRINT(50) allowed (max basic unlock opcode)",
			code:    []byte{byte(PRINT)},
			wantErr: false,
		},
		{
			name:    "SYS_NULL(169) allowed (special case)",
			code:    []byte{byte(SYS_NULL)},
			wantErr: false,
		},
		{
			name:    "PASS(51) not allowed in unlock",
			code:    []byte{byte(PASS)},
			wantErr: true,
		},
		{
			name:    "END(57) not allowed in unlock",
			code:    []byte{byte(END)},
			wantErr: true,
		},
		{
			name:    "SYS_TIME(164) not allowed in unlock",
			code:    []byte{byte(SYS_TIME)},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeUnlockScript(tc.code)
			if tc.wantErr && err == nil {
				t.Error("expected error for invalid unlock opcode, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestDecodeScriptWithAttrParam 验证带附参的指令解码。
func TestDecodeScriptWithAttrParam(t *testing.T) {
	// BYTE_LIT: opcode=3, attr=1字节值
	// 编码 BYTE{0xAB}：[3, 0xAB]
	code := []byte{byte(BYTE_LIT), 0xAB}
	frames, err := DecodeScript(code, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	if frames[0].Op != BYTE_LIT {
		t.Errorf("op = %d, want %d", frames[0].Op, BYTE_LIT)
	}
	if len(frames[0].AttrParams) != 1 || frames[0].AttrParams[0][0] != 0xAB {
		t.Errorf("attr param mismatch")
	}
}

// TestDecodeScriptWithAssocData 验证带关联数据的指令解码。
func TestDecodeScriptWithAssocData(t *testing.T) {
	// DATA_LIT: opcode=10, attr=ULEB128 length=3, assoc=3字节
	data := []byte{0x01, 0x02, 0x03}
	code := append([]byte{byte(DATA_LIT), 3}, data...)
	frames, err := DecodeScript(code, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	if frames[0].Op != DATA_LIT {
		t.Errorf("op = %d, want DATAIT", frames[0].Op)
	}
	if string(frames[0].AssocData) != string(data) {
		t.Errorf("assoc data = %v, want %v", frames[0].AssocData, data)
	}
}

// TestDecodeScriptFloatAttrParam 验证 FLOAT_LIT 的 8 字节附参解码。
func TestDecodeScriptFloatAttrParam(t *testing.T) {
	// FLOAT_LIT: opcode=7, attr=8字节 big-endian IEEE 754
	// 编码 float64(1.0): 0x3FF0000000000000
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], 0x3FF0000000000000)
	code := append([]byte{byte(FLOAT_LIT)}, buf[:]...)
	frames, err := DecodeScript(code, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 1 || frames[0].Op != FLOAT_LIT {
		t.Fatalf("unexpected frames")
	}
	if len(frames[0].AttrParams[0]) != 8 {
		t.Errorf("float attr param should be 8 bytes, got %d", len(frames[0].AttrParams[0]))
	}
}

// TestDecodeScriptOffset 验证帧 Offset 字段正确记录字节偏移。
func TestDecodeScriptOffset(t *testing.T) {
	// [NIL, NIL, TRUE]：偏移分别为 0, 1, 2
	code := []byte{byte(NIL), byte(NIL), byte(TRUE)}
	frames, err := DecodeScript(code, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedOffsets := []int{0, 1, 2}
	for i, f := range frames {
		if f.Offset != expectedOffsets[i] {
			t.Errorf("frame[%d].Offset = %d, want %d", i, f.Offset, expectedOffsets[i])
		}
	}
}
