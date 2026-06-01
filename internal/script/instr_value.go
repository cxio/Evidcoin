package script

import (
	"encoding/binary"
	"math"
)

// instr_value.go 实现值指令 [0-18] 的执行函数。
// 参考：docs/proposal/Instruction/1.Value-Instructions.md，DEC-0501/0502。

func init() {
	registerExec(NIL, execNIL)
	registerExec(TRUE, execTRUE)
	registerExec(FALSE, execFALSE)
	registerExec(BYTE_LIT, execBYTE_LIT)
	registerExec(RUNE_LIT, execRUNE_LIT)
	registerExec(INT_LIT, execINT_LIT)
	registerExec(BIGINT_LIT, execBIGINT_LIT)
	registerExec(FLOAT_LIT, execFLOAT_LIT)
	registerExec(STRING_LIT, execSTRING_LIT)
	registerExec(VALUES_LIT, execVALUES_LIT)
	registerExec(DATA_LIT, execDATA_LIT)
	registerExec(REGEXP_LIT, execREGEXP_LIT)
	registerExec(DATE_LIT, execDATE_LIT)
	registerExec(DICT_LIT, execDICT_LIT)
	registerExec(CODE_LIT, execCODE_LIT)
	// SCRIPT(17) 和 VALUE(18) 为禁用指令：静态存在不拒绝，
	// 执行时由 checkPublicSafety 返回 ErrDisabledInPublic（公共路径）；
	// 私有路径若执行则直接返回 ScriptError（无执行函数，executor 处理）。
	// 因此不注册执行函数，执行时由 executor 以"未注册"处理为 ScriptError。
}

// execNIL 压入 nil 值。
func execNIL(vm *VM, _ *InstrFrame) error {
	return vm.stack.Push(NilValue())
}

// execTRUE 压入 bool true。
func execTRUE(vm *VM, _ *InstrFrame) error {
	return vm.stack.Push(BoolValue(true))
}

// execFALSE 压入 bool false。
func execFALSE(vm *VM, _ *InstrFrame) error {
	return vm.stack.Push(BoolValue(false))
}

// execBYTE_LIT 压入单字节字面量（1 字节固定附参）。
func execBYTE_LIT(vm *VM, f *InstrFrame) error {
	return vm.stack.Push(ByteValue(f.AttrParams[0][0]))
}

// execRUNE_LIT 压入 Unicode 码点字面量（4 字节大端固定附参）。
func execRUNE_LIT(vm *VM, f *InstrFrame) error {
	r := rune(binary.BigEndian.Uint32(f.AttrParams[0]))
	return vm.stack.Push(RuneValue(r))
}

// execINT_LIT 压入 int64 字面量（ULEB128 附参，按 uint64 位模式解释为 int64）。
func execINT_LIT(vm *VM, f *InstrFrame) error {
	// 附参已由 decodeULEB128 解码存为 8 字节小端 uint64
	u := binary.LittleEndian.Uint64(f.AttrParams[0])
	return vm.stack.Push(IntValue(int64(u)))
}

// execBIGINT_LIT 压入大整数字面量（DEC-0001 slen||magnitude）。
// 附参 1 字节：bit7=符号（1=负），低 7 位=magnitude 字节数；关联数据=magnitude。
// 当前以 TypeBigInt+BytesValue 占位存储。
func execBIGINT_LIT(vm *VM, f *InstrFrame) error {
	slen := f.AttrParams[0][0]
	mag := f.AssocData
	// DEC-0001：拒绝前导零（magnitude 首字节为 0 且长度 > 0）
	if len(mag) > 0 && mag[0] == 0 {
		return ErrTypeMismatch // 前导零
	}
	// DEC-0001：拒绝负零（符号位为 1 且 magnitude 长度为 0）
	if slen&0x80 != 0 && len(mag) == 0 {
		return ErrTypeMismatch // 负零
	}
	// 以 (slen<<(8*len)) || magnitude 形式存储为 Bytes 占位
	raw := make([]byte, 1+len(mag))
	raw[0] = slen
	copy(raw[1:], mag)
	v := Value{typ: TypeBigInt, data: raw}
	return vm.stack.Push(v)
}

// execFLOAT_LIT 压入 float64 字面量（8 字节大端 IEEE 754 bit pattern，DEC-0502）。
// 字面量不允许 NaN/Inf。
func execFLOAT_LIT(vm *VM, f *InstrFrame) error {
	bits := binary.BigEndian.Uint64(f.AttrParams[0])
	f64 := math.Float64frombits(bits)
	v, err := FloatLiteralValue(f64)
	if err != nil {
		return err
	}
	return vm.stack.Push(v)
}

// execSTRING_LIT 压入字符串字面量（ULEB128 长度附参 + 关联数据 UTF-8 字节）。
func execSTRING_LIT(vm *VM, f *InstrFrame) error {
	return vm.stack.Push(StringValue(string(f.AssocData)))
}

// execVALUES_LIT 压入值切片字面量（关联数据为成员序列字节，当前以 Bytes 占位）。
// 完整实现需要在关联数据中再解码子值序列；此版本以原始字节存储。
func execVALUES_LIT(vm *VM, f *InstrFrame) error {
	// 占位：以 TypeSlice 存储空切片，关联数据原字节附加为 Code 值
	// TODO：Task 7+ 中替换为子值序列解码
	_ = f.AssocData
	return vm.stack.Push(SliceValue(nil))
}

// execDATA_LIT 压入字节序列字面量（ULEB128 长度附参 + 关联数据字节）。
func execDATA_LIT(vm *VM, f *InstrFrame) error {
	return vm.stack.Push(BytesValue(f.AssocData))
}

// execREGEXP_LIT 压入正则表达式字面量（1 字节长度附参 + 关联数据字节，RE2）。
// 当前以 TypeRegExp+Bytes 占位，完整实现在 Task 7（regexp.MustCompile）。
func execREGEXP_LIT(vm *VM, f *InstrFrame) error {
	v := Value{typ: TypeRegExp, data: append([]byte(nil), f.AssocData...)}
	return vm.stack.Push(v)
}

// execDATE_LIT 压入时间字面量（关联数据为 UNIX 毫秒有符号变长整数，DEC-0001）。
// 当前以 ULEB128 解码 uint64 后强转 int64 占位（负时间戳后续需 signed LEB128）。
func execDATE_LIT(vm *VM, f *InstrFrame) error {
	// 附参 ULEB128 已解码为 8 字节小端 uint64
	u := binary.LittleEndian.Uint64(f.AttrParams[0])
	return vm.stack.Push(TimeValue(int64(u)))
}

// execDICT_LIT 从实参区取两个切片（键序列+值序列）构造有序字典。
// arg1=[]String 键序列，arg2=[]Value 值序列。当前以 TypeDict+Bytes 占位。
func execDICT_LIT(vm *VM, f *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	// 占位：以两个 Value 打包为 Slice 存储
	v := Value{typ: TypeDict, data: []Value{args[0], args[1]}}
	return vm.stack.Push(v)
}

// execCODE_LIT 压入编译后指令序列字面值（ULEB128 长度附参 + 关联数据字节）。
func execCODE_LIT(vm *VM, f *InstrFrame) error {
	return vm.stack.Push(CodeValue(f.AssocData))
}
