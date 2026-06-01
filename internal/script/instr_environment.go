package script

import (
	"encoding/binary"
	"fmt"
)

// instr_environment.go 实现环境指令 [128-137] 的执行函数。
// 参考：docs/proposal/Instruction/13.Environment-Instructions.md，DEC-0503/0505。

func init() {
	registerExec(ENV, execENV)
	registerExec(IN, execIN)
	registerExec(OUT, execOUT)
	registerExec(INOUT, execINOUT)
	// opcode 132 保留，不注册
	registerExec(XFROM, execXFROM)
	registerExec(SIGNED, execSIGNED)
	registerExec(VAR, execVAR)
	registerExec(SETVAR, execSETVAR)
	registerExec(SOURCE, execSOURCE)
}

// envAttrNames 将 ENV/IN/XFROM 附参值映射为字段名。
var envAttrNames = map[uint64]string{
	0: "BlockTime",
	1: "Height",
	2: "TipHeight",
	3: "TxID",
	4: "TxTime",
	5: "Passed",
	6: "InGoto",
	7: "InEmbed",
}

// readULEB128Param 从小端 uint64 编码的附参字节切片中读取值。
// bytecode.go 的 decodeAttrParams 将 ULEB128 变长参数存储为小端 uint64（8 字节）。
func readULEB128Param(b []byte) uint64 {
	if len(b) == 8 {
		// 变长参数已被 decodeAttrParams 解码并以小端 uint64 存储
		return binary.LittleEndian.Uint64(b)
	}
	// 回退：直接解码原始 ULEB128 字节（兼容测试构造的原始帧）
	var val uint64
	var shift uint
	for _, byt := range b {
		val |= uint64(byt&0x7f) << shift
		shift += 7
		if byt&0x80 == 0 {
			break
		}
	}
	return val
}

// attrName 将附参值转换为字段名字符串。
func attrName(v uint64) string {
	if name, ok := envAttrNames[v]; ok {
		return name
	}
	return fmt.Sprintf("env.%d", v)
}

// execENV 从运行时环境取变量（ENV，opcode 128）。
// 附参=目标标识（ULEB128），映射为预定义字段名后调用 env.Lookup。
func execENV(vm *VM, f *InstrFrame) error {
	if len(f.AttrParams) == 0 {
		return fmt.Errorf("%w: ENV requires attr param", ErrTypeMismatch)
	}
	attrVal := readULEB128Param(f.AttrParams[0])
	name := attrName(attrVal)
	if vm.env == nil {
		return vm.stack.Push(NilValue())
	}
	v, err := vm.env.Lookup(name)
	if err != nil {
		return err
	}
	return vm.stack.Push(v)
}

// execIN 取输入项数据（IN，opcode 129）。
// 附参=目标标识，查询 "in.<fieldName>"。
func execIN(vm *VM, f *InstrFrame) error {
	if len(f.AttrParams) == 0 {
		return fmt.Errorf("%w: IN requires attr param", ErrTypeMismatch)
	}
	attrVal := readULEB128Param(f.AttrParams[0])
	name := "in." + attrName(attrVal)
	if vm.env == nil {
		return vm.stack.Push(NilValue())
	}
	v, err := vm.env.Lookup(name)
	if err != nil {
		return err
	}
	return vm.stack.Push(v)
}

// execOUT 取输出项数据（OUT，opcode 130）。
// 附参 1=输出序位，附参 2=目标标识，查询 "out.<seqIdx>.<fieldName>"。
func execOUT(vm *VM, f *InstrFrame) error {
	if len(f.AttrParams) < 2 {
		return fmt.Errorf("%w: OUT requires 2 attr params", ErrTypeMismatch)
	}
	seqIdx := readULEB128Param(f.AttrParams[0])
	fieldID := readULEB128Param(f.AttrParams[1])
	fieldName := attrName(fieldID)
	key := fmt.Sprintf("out.%d.%s", seqIdx, fieldName)
	if vm.env == nil {
		return vm.stack.Push(NilValue())
	}
	v, err := vm.env.Lookup(key)
	if err != nil {
		return err
	}
	return vm.stack.Push(v)
}

// execINOUT 取输入项来源输出集兄弟条目（INOUT，opcode 131）。
// 前期禁用（DEC-0505），任何路径执行都返回 ScriptError。
func execINOUT(vm *VM, _ *InstrFrame) error {
	return ErrDisabledInPublic
}

// execXFROM 获取源交易信息（XFROM，opcode 133）。
// 仅在 GOTO/EMBED 目标脚本中有意义，附参=目标标识。
func execXFROM(vm *VM, f *InstrFrame) error {
	if len(f.AttrParams) == 0 {
		return fmt.Errorf("%w: XFROM requires attr param", ErrTypeMismatch)
	}
	attrVal := readULEB128Param(f.AttrParams[0])
	name := "xfrom." + attrName(attrVal)
	if vm.env == nil {
		return vm.stack.Push(NilValue())
	}
	v, err := vm.env.Lookup(name)
	if err != nil {
		return err
	}
	return vm.stack.Push(v)
}

// execSIGNED 查询签名序位是否已通过验证（SIGNED，opcode 134）。
// 附参=目标序位（ULEB128），压入 BoolValue。
func execSIGNED(vm *VM, f *InstrFrame) error {
	if len(f.AttrParams) == 0 {
		return fmt.Errorf("%w: SIGNED requires attr param", ErrTypeMismatch)
	}
	idx := int(readULEB128Param(f.AttrParams[0]))
	return vm.stack.Push(BoolValue(vm.GetSigned(idx)))
}

// execVAR 读取全局变量并压栈（VAR，opcode 135）。
// 附参=变量位置 [0-255]（ULEB128）。
func execVAR(vm *VM, f *InstrFrame) error {
	if len(f.AttrParams) == 0 {
		return fmt.Errorf("%w: VAR requires attr param", ErrTypeMismatch)
	}
	idx := int(readULEB128Param(f.AttrParams[0]))
	return vm.stack.Push(vm.GetGlobalVar(idx))
}

// execSETVAR 从实参区/栈取 1 值写入全局变量（SETVAR，opcode 136）。
// 附参=变量位置 [0-255]（ULEB128）。
func execSETVAR(vm *VM, f *InstrFrame) error {
	if len(f.AttrParams) == 0 {
		return fmt.Errorf("%w: SETVAR requires attr param", ErrTypeMismatch)
	}
	idx := int(readULEB128Param(f.AttrParams[0]))
	val, err := vm.getOneArg()
	if err != nil {
		return err
	}
	vm.SetGlobalVar(idx, val)
	return nil
}

// execSOURCE 脚本源码提取（SOURCE，opcode 137）。
// 附参=标识值，简化实现：从 env 取 "source" 或压入空字节。
func execSOURCE(vm *VM, _ *InstrFrame) error {
	if vm.env == nil {
		return vm.stack.Push(BytesValue(nil))
	}
	v, err := vm.env.Lookup("source")
	if err != nil {
		return vm.stack.Push(BytesValue(nil))
	}
	return vm.stack.Push(v)
}
