package script

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"
)

// instr_conversion.go 实现转换指令 [67-79] 的执行函数。
// opcode 78 保留，不注册。
// 参考：docs/proposal/Instruction/8.Conversion-Instructions.md

func init() {
	registerExec(BOOL, execBOOL)
	registerExec(BYTE_CONV, execBYTE_CONV)
	registerExec(RUNE_CONV, execRUNE_CONV)
	registerExec(INT_CONV, execINT_CONV)
	registerExec(BIGINT_CONV, execBIGINT_CONV)
	registerExec(FLOAT_CONV, execFLOAT_CONV)
	registerExec(STRING_CONV, execSTRING_CONV)
	registerExec(BYTES_CONV, execBYTES_CONV)
	registerExec(RUNES_CONV, execRUNES_CONV)
	registerExec(TIME_CONV, execTIME_CONV)
	registerExec(REGEXP_CONV, execREGEXP_CONV)
	// opcode 78 保留，不注册
	registerExec(ANYS, execANYS)
}

// execBOOL 转换为布尔类型（BOOL，opcode 67）。
// nil/0/0.0/""/空切片/false → false；其余 → true。
func execBOOL(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	var result bool
	switch v.Typ() {
	case TypeNil:
		result = false
	case TypeBool:
		result, _ = v.AsBool()
	case TypeByte:
		b, _ := v.AsByte()
		result = b != 0
	case TypeRune:
		r, _ := v.AsRune()
		result = r != 0
	case TypeInt:
		n, _ := v.AsInt()
		result = n != 0
	case TypeFloat:
		f, _ := v.AsFloat()
		result = f != 0.0 && !math.IsNaN(f)
	case TypeString:
		s, _ := v.AsString()
		result = s != ""
	case TypeBytes:
		b, _ := v.AsBytes()
		result = len(b) != 0
	case TypeSlice:
		sl, _ := v.AsSlice()
		result = len(sl) != 0
	default:
		// 其他类型（RegExp/Dict/Object 等）视为 true
		result = true
	}
	return vm.stack.Push(BoolValue(result))
}

// execBYTE_CONV 转换为字节类型（BYTE_CONV，opcode 68）。
func execBYTE_CONV(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	switch v.Typ() {
	case TypeNil:
		return vm.stack.Push(ByteValue(0))
	case TypeBool:
		b, _ := v.AsBool()
		if b {
			return vm.stack.Push(ByteValue(1))
		}
		return vm.stack.Push(ByteValue(0))
	case TypeByte:
		return vm.stack.Push(v)
	case TypeRune:
		r, _ := v.AsRune()
		if r > 255 {
			return fmt.Errorf("%w: rune 0x%X exceeds byte range", ErrIndexOutOfRange, r)
		}
		return vm.stack.Push(ByteValue(byte(r)))
	case TypeInt:
		n, _ := v.AsInt()
		if n < 0 || n > 255 {
			return fmt.Errorf("%w: int %d exceeds byte range", ErrIndexOutOfRange, n)
		}
		return vm.stack.Push(ByteValue(byte(n)))
	case TypeFloat:
		f, _ := v.AsFloat()
		n := int64(math.Trunc(f))
		if n < 0 || n > 255 {
			return fmt.Errorf("%w: float truncated to %d exceeds byte range", ErrIndexOutOfRange, n)
		}
		return vm.stack.Push(ByteValue(byte(n)))
	default:
		return fmt.Errorf("%w: BYTE_CONV unsupported type %s", ErrTypeMismatch, v.Typ())
	}
}

// execRUNE_CONV 转换为 Unicode 码点类型（RUNE_CONV，opcode 69）。
func execRUNE_CONV(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	const maxRune = 0x10FFFF
	switch v.Typ() {
	case TypeNil:
		return vm.stack.Push(RuneValue(0))
	case TypeBool:
		b, _ := v.AsBool()
		if b {
			return vm.stack.Push(RuneValue(1))
		}
		return vm.stack.Push(RuneValue(0))
	case TypeByte:
		b, _ := v.AsByte()
		return vm.stack.Push(RuneValue(rune(b)))
	case TypeRune:
		return vm.stack.Push(v)
	case TypeInt:
		n, _ := v.AsInt()
		if n < 0 || n > maxRune {
			return fmt.Errorf("%w: int %d exceeds rune range", ErrIndexOutOfRange, n)
		}
		return vm.stack.Push(RuneValue(rune(n)))
	case TypeFloat:
		f, _ := v.AsFloat()
		n := int64(math.Trunc(f))
		if n < 0 || n > maxRune {
			return fmt.Errorf("%w: float truncated to %d exceeds rune range", ErrIndexOutOfRange, n)
		}
		return vm.stack.Push(RuneValue(rune(n)))
	default:
		return fmt.Errorf("%w: RUNE_CONV unsupported type %s", ErrTypeMismatch, v.Typ())
	}
}

// execINT_CONV 转换为 int64 类型（INT_CONV，opcode 70）。
func execINT_CONV(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	switch v.Typ() {
	case TypeNil:
		return vm.stack.Push(IntValue(0))
	case TypeBool:
		b, _ := v.AsBool()
		if b {
			return vm.stack.Push(IntValue(1))
		}
		return vm.stack.Push(IntValue(0))
	case TypeByte:
		b, _ := v.AsByte()
		return vm.stack.Push(IntValue(int64(b)))
	case TypeRune:
		r, _ := v.AsRune()
		return vm.stack.Push(IntValue(int64(r)))
	case TypeInt:
		return vm.stack.Push(v)
	case TypeBigInt:
		// BigInt 占位：无法转换，返回错误
		return fmt.Errorf("%w: INT_CONV from BigInt not yet implemented", ErrIndexOutOfRange)
	case TypeFloat:
		f, _ := v.AsFloat()
		return vm.stack.Push(IntValue(int64(math.Trunc(f))))
	case TypeTime:
		ms, _ := v.AsTime()
		return vm.stack.Push(IntValue(ms))
	case TypeString:
		s, _ := v.AsString()
		n, err := strconv.ParseInt(s, 0, 64)
		if err != nil {
			return fmt.Errorf("%w: INT_CONV parse error: %v", ErrTypeMismatch, err)
		}
		return vm.stack.Push(IntValue(n))
	default:
		return fmt.Errorf("%w: INT_CONV unsupported type %s", ErrTypeMismatch, v.Typ())
	}
}

// execBIGINT_CONV 转换为大整数类型（BIGINT_CONV，opcode 71）。
// BigInt 未完整实现，占位返回 NilValue()。
func execBIGINT_CONV(vm *VM, _ *InstrFrame) error {
	_, err := vm.getOneArg()
	if err != nil {
		return err
	}
	// BigInt 占位
	return vm.stack.Push(NilValue())
}

// execFLOAT_CONV 转换为 float64 类型（FLOAT_CONV，opcode 72）。
// 结果不得为 NaN/Inf（DEC-0502），否则 ErrInvalidFloat。
func execFLOAT_CONV(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	var f float64
	switch v.Typ() {
	case TypeNil:
		f = 0.0
	case TypeBool:
		b, _ := v.AsBool()
		if b {
			f = 1.0
		}
	case TypeByte:
		b, _ := v.AsByte()
		f = float64(b)
	case TypeRune:
		r, _ := v.AsRune()
		f = float64(r)
	case TypeInt:
		n, _ := v.AsInt()
		f = float64(n)
	case TypeFloat:
		return vm.stack.Push(v)
	case TypeString:
		s, _ := v.AsString()
		parsed, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("%w: FLOAT_CONV parse error: %v", ErrTypeMismatch, err)
		}
		f = parsed
	default:
		return fmt.Errorf("%w: FLOAT_CONV unsupported type %s", ErrTypeMismatch, v.Typ())
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return ErrInvalidFloat
	}
	return vm.stack.Push(FloatValue(f))
}

// execSTRING_CONV 转换为字符串（STRING_CONV，opcode 73）。
// 附参=进制或格式标识（ULEB128）；0 或无附参=默认 fmt 格式。
func execSTRING_CONV(vm *VM, f *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	var base uint64
	if len(f.AttrParams) > 0 {
		base = readULEB128Param(f.AttrParams[0])
	}
	var s string
	switch v.Typ() {
	case TypeNil:
		s = "nil"
	case TypeBool:
		b, _ := v.AsBool()
		s = strconv.FormatBool(b)
	case TypeByte:
		b, _ := v.AsByte()
		if base >= 2 && base <= 36 {
			s = strconv.FormatUint(uint64(b), int(base))
		} else {
			s = strconv.FormatUint(uint64(b), 10)
		}
	case TypeRune:
		r, _ := v.AsRune()
		if base >= 2 && base <= 36 {
			s = strconv.FormatInt(int64(r), int(base))
		} else {
			s = string(r)
		}
	case TypeInt:
		n, _ := v.AsInt()
		if base >= 2 && base <= 36 {
			s = strconv.FormatInt(n, int(base))
		} else {
			s = strconv.FormatInt(n, 10)
		}
	case TypeFloat:
		fl, _ := v.AsFloat()
		s = strconv.FormatFloat(fl, 'g', -1, 64)
	case TypeString:
		s, _ = v.AsString()
	case TypeBytes:
		b, _ := v.AsBytes()
		s = string(b)
	case TypeTime:
		ms, _ := v.AsTime()
		t := time.UnixMilli(ms).UTC()
		s = t.Format(time.RFC3339)
	default:
		s = fmt.Sprintf("%v", v)
	}
	return vm.stack.Push(StringValue(s))
}

// execBYTES_CONV 转换为字节序列（BYTES_CONV，opcode 74）。
// 不支持从 Code 转换（安全）。
func execBYTES_CONV(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	switch v.Typ() {
	case TypeNil:
		return vm.stack.Push(BytesValue(nil))
	case TypeByte:
		b, _ := v.AsByte()
		return vm.stack.Push(BytesValue([]byte{b}))
	case TypeRune:
		r, _ := v.AsRune()
		buf := make([]byte, utf8.RuneLen(r))
		utf8.EncodeRune(buf, r)
		return vm.stack.Push(BytesValue(buf))
	case TypeInt:
		n, _ := v.AsInt()
		var buf [8]byte
		for i := 7; i >= 0; i-- {
			buf[i] = byte(n)
			n >>= 8
		}
		return vm.stack.Push(BytesValue(buf[:]))
	case TypeBigInt:
		// BigInt 占位：返回空 Bytes
		return vm.stack.Push(BytesValue(nil))
	case TypeString:
		s, _ := v.AsString()
		return vm.stack.Push(BytesValue([]byte(s)))
	case TypeBytes:
		b, _ := v.AsBytes()
		return vm.stack.Push(BytesValue(b))
	case TypeCode:
		// 安全限制：不允许从 Code 转换
		return fmt.Errorf("%w: BYTES_CONV from Code is not allowed", ErrTypeMismatch)
	default:
		return fmt.Errorf("%w: BYTES_CONV unsupported type %s", ErrTypeMismatch, v.Typ())
	}
}

// execRUNES_CONV 转换为符文序列（RUNES_CONV，opcode 75）。
func execRUNES_CONV(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	switch v.Typ() {
	case TypeNil:
		return vm.stack.Push(SliceValue([]Value{}))
	case TypeRune:
		r, _ := v.AsRune()
		return vm.stack.Push(SliceValue([]Value{RuneValue(r)}))
	case TypeString:
		s, _ := v.AsString()
		runes := []rune(s)
		vals := make([]Value, len(runes))
		for i, r := range runes {
			vals[i] = RuneValue(r)
		}
		return vm.stack.Push(SliceValue(vals))
	case TypeBytes:
		b, _ := v.AsBytes()
		var vals []Value
		for len(b) > 0 {
			r, size := utf8.DecodeRune(b)
			vals = append(vals, RuneValue(r))
			b = b[size:]
		}
		if vals == nil {
			vals = []Value{}
		}
		return vm.stack.Push(SliceValue(vals))
	default:
		return fmt.Errorf("%w: RUNES_CONV unsupported type %s", ErrTypeMismatch, v.Typ())
	}
}

// execTIME_CONV 转换为时间类型（TIME_CONV，opcode 76）。
// Int→TimeValue(unix 毫秒)；String→RFC3339 解析；nil→TimeValue(0)。
func execTIME_CONV(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	switch v.Typ() {
	case TypeNil:
		return vm.stack.Push(TimeValue(0))
	case TypeInt:
		ms, _ := v.AsInt()
		return vm.stack.Push(TimeValue(ms))
	case TypeTime:
		return vm.stack.Push(v)
	case TypeString:
		s, _ := v.AsString()
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return fmt.Errorf("%w: TIME_CONV parse error: %v", ErrTypeMismatch, err)
		}
		return vm.stack.Push(TimeValue(t.UnixMilli()))
	default:
		return fmt.Errorf("%w: TIME_CONV unsupported type %s", ErrTypeMismatch, v.Typ())
	}
}

// execREGEXP_CONV 从字符串构造正则（REGEXP_CONV，opcode 77）。
// String→编译 RE2 正则，成功则压入 TypeRegExp 值。
func execREGEXP_CONV(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	s, err := v.AsString()
	if err != nil {
		return fmt.Errorf("%w: REGEXP_CONV requires String", ErrTypeMismatch)
	}
	re, err := regexp.Compile(s)
	if err != nil {
		return fmt.Errorf("%w: REGEXP_CONV invalid pattern: %v", ErrTypeMismatch, err)
	}
	return vm.stack.Push(Value{typ: TypeRegExp, data: re})
}

// execANYS 切片类型转换（ANYS，opcode 79）。
// 附参=转换标识（0=原样；1/2=转为 Bool）；占位实现。
func execANYS(vm *VM, f *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	sl, err := v.AsSlice()
	if err != nil {
		return err
	}
	var mode uint64
	if len(f.AttrParams) > 0 {
		mode = readULEB128Param(f.AttrParams[0])
	}
	switch mode {
	case 0:
		// 原样返回
		return vm.stack.Push(SliceValue(sl))
	case 1, 2:
		// 每个成员转为 Bool
		result := make([]Value, len(sl))
		for i, item := range sl {
			var b bool
			switch item.Typ() {
			case TypeNil:
				b = false
			case TypeBool:
				b, _ = item.AsBool()
			default:
				b = true
			}
			result[i] = BoolValue(b)
		}
		return vm.stack.Push(SliceValue(result))
	default:
		return fmt.Errorf("%w: ANYS unsupported mode %d", ErrTypeMismatch, mode)
	}
}
