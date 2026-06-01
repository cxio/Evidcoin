package script

import (
	"encoding/binary"
	"fmt"

	"github.com/cxio/evidcoin/pkg/types"
)

// InstrFrame 是字节码解码后的指令帧，包含 opcode、附参字节列表和关联数据。
// 它不保存运行时实参——实参在执行阶段从实参区/数据栈取得。
type InstrFrame struct {
	// Op 指令码。
	Op Opcode
	// AttrParams 附参字节值序列（已按 AttrParamSizes 解析）。
	AttrParams [][]byte
	// AssocData 关联数据（原始字节）；如果指令无关联数据则为 nil。
	AssocData []byte
	// Offset 该指令在原始字节码中的起始偏移（调试/SOURCE 提取用）。
	Offset int
}

// DecodeScript 将字节码字节序列解码为指令帧列表。
//
// 严格遵循 DEC-0501：
//   - 拒绝截断附参（附参字节不足）。
//   - 拒绝截断关联数据（关联数据字节不足）。
//   - 拒绝尾随未被任何指令消费的残余字节。
//   - 拒绝未注册的 opcode（含系统保留 254/255）。
//
// maxLen 为允许的最大脚本字节数（0 表示不限制）。
func DecodeScript(code []byte, maxLen int) ([]InstrFrame, error) {
	if maxLen > 0 && len(code) > maxLen {
		return nil, fmt.Errorf("%w: len=%d, max=%d", ErrScriptTooLong, len(code), maxLen)
	}
	var frames []InstrFrame
	pos := 0
	for pos < len(code) {
		offset := pos
		op := Opcode(code[pos])
		pos++

		// 检查 opcode 是否注册
		meta := Lookup(op)
		if meta == nil {
			return nil, fmt.Errorf("%w: opcode %d at offset %d", ErrUnknownOpcode, op, offset)
		}

		// 解析附参
		attrParams, consumed, err := decodeAttrParams(code, pos, meta.AttrParamSizes)
		if err != nil {
			return nil, fmt.Errorf("opcode %d (%s) at offset %d: %w", op, meta.Mnemonic, offset, err)
		}
		pos += consumed

		// 解析关联数据
		var assocData []byte
		if meta.AssocDataParam >= 0 && meta.AssocDataParam < len(attrParams) {
			length, err := attrParamToInt(attrParams[meta.AssocDataParam])
			if err != nil {
				return nil, fmt.Errorf("opcode %d (%s) at offset %d: assoc data length invalid: %w", op, meta.Mnemonic, offset, err)
			}
			if pos+length > len(code) {
				return nil, fmt.Errorf("opcode %d (%s) at offset %d: %w (need %d bytes, have %d)", op, meta.Mnemonic, offset, ErrTruncatedAssocData, length, len(code)-pos)
			}
			assocData = make([]byte, length)
			copy(assocData, code[pos:pos+length])
			pos += length
		}

		frames = append(frames, InstrFrame{
			Op:         op,
			AttrParams: attrParams,
			AssocData:  assocData,
			Offset:     offset,
		})
	}

	// 检查尾随残余字节（循环结束时 pos 应恰好等于 len(code)）
	if pos != len(code) {
		return nil, fmt.Errorf("%w: %d bytes unconsumed", ErrTrailingBytes, len(code)-pos)
	}

	return frames, nil
}

// DecodeUnlockScript 解码解锁脚本字节码，额外检查 opcode 限制。
//
// 解锁脚本只允许 opcode [0-50] 以及 SYS_NULL(169)。
// SCRIPT(17)/VALUE(18) 虽在 [0-50] 范围内，但已被禁用，不得出现在主网有效解锁脚本中。
// 参考：docs/proposal/10.Script-System.md §11
func DecodeUnlockScript(code []byte) ([]InstrFrame, error) {
	if len(code) > types.MaxUnlockScript {
		return nil, fmt.Errorf("%w: len=%d, max=%d", ErrScriptTooLong, len(code), types.MaxUnlockScript)
	}
	frames, err := DecodeScript(code, 0)
	if err != nil {
		return nil, err
	}
	// 检查每条指令是否允许在解锁脚本中使用
	for _, f := range frames {
		if !f.Op.IsAllowedInUnlock() {
			return nil, fmt.Errorf("%w: opcode %d (%s) at offset %d",
				ErrInvalidUnlockOpcode, f.Op, Lookup(f.Op).Mnemonic, f.Offset)
		}
	}
	return frames, nil
}

// DecodeLockScript 解码锁定/识别脚本字节码（最大 MaxLockScript）。
func DecodeLockScript(code []byte) ([]InstrFrame, error) {
	return DecodeScript(code, types.MaxLockScript)
}

// ─── 内部辅助函数 ─────────────────────────────────────────────────────────────

// decodeAttrParams 从字节码 code[pos:] 中解析 sizes 描述的附参列表。
// sizes[i] = 0 表示变长 ULEB128；sizes[i] > 0 表示固定宽度大端。
// 返回：解析后的附参字节值列表和消费的总字节数。
func decodeAttrParams(code []byte, pos int, sizes []int) ([][]byte, int, error) {
	if len(sizes) == 0 {
		return nil, 0, nil
	}
	consumed := 0
	params := make([][]byte, len(sizes))
	for i, size := range sizes {
		if size == 0 {
			// 变长 ULEB128 解码
			val, n, err := decodeULEB128(code, pos+consumed)
			if err != nil {
				return nil, 0, fmt.Errorf("%w (varint param %d)", ErrTruncatedAttrParam, i)
			}
			// 存为 uint64 小端字节（便于后续 attrParamToInt 统一处理）
			buf := make([]byte, 8)
			binary.LittleEndian.PutUint64(buf, val)
			params[i] = buf
			consumed += n
		} else {
			// 固定宽度大端
			if pos+consumed+size > len(code) {
				return nil, 0, fmt.Errorf("%w (fixed param %d, need %d bytes)", ErrTruncatedAttrParam, i, size)
			}
			buf := make([]byte, size)
			copy(buf, code[pos+consumed:pos+consumed+size])
			params[i] = buf
			consumed += size
		}
	}
	return params, consumed, nil
}

// decodeULEB128 从 code[pos:] 解码一个 ULEB128 变长无符号整数。
// 返回值、消费的字节数。
// 符合 DEC-0001：拒绝非最短编码（前导零字节组）。
func decodeULEB128(code []byte, pos int) (uint64, int, error) {
	var val uint64
	var shift uint
	for i := 0; ; i++ {
		if pos+i >= len(code) {
			return 0, 0, ErrTruncatedAttrParam
		}
		b := code[pos+i]
		val |= uint64(b&0x7f) << shift
		shift += 7
		if b&0x80 == 0 {
			n := i + 1
			// 检查最短编码：若长度 > 1 且最后一个字节为 0，则非最短
			if n > 1 && b == 0 {
				return 0, 0, fmt.Errorf("script: non-shortest ULEB128 encoding at pos %d", pos)
			}
			return val, n, nil
		}
		if shift >= 64 {
			return 0, 0, fmt.Errorf("script: ULEB128 overflow at pos %d", pos)
		}
	}
}

// attrParamToInt 将附参字节值转换为整数长度（用于关联数据长度计算）。
// 对于变长参数（以小端 uint64 存储），直接读取；
// 对于固定宽度大端，按长度转换。
func attrParamToInt(param []byte) (int, error) {
	switch len(param) {
	case 1:
		return int(param[0]), nil
	case 2:
		return int(binary.BigEndian.Uint16(param)), nil
	case 4:
		return int(binary.BigEndian.Uint32(param)), nil
	case 8:
		// 变长参数存储为小端 uint64
		v := binary.LittleEndian.Uint64(param)
		if v > 1<<31 {
			return 0, fmt.Errorf("script: assoc data length too large: %d", v)
		}
		return int(v), nil
	default:
		return 0, fmt.Errorf("script: unexpected attr param size %d", len(param))
	}
}
