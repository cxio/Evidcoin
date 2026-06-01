package script

import (
	"fmt"
	"math"
)

// Type 表示 VM 值的类型标签。
// 参考：docs/proposal/10.Script-System.md §3，DEC-0502。
type Type uint8

const (
	TypeNil    Type = 0  // nil
	TypeBool   Type = 1  // bool
	TypeByte   Type = 2  // byte（uint8）
	TypeRune   Type = 3  // rune（int32，Unicode 码点）
	TypeInt    Type = 4  // int64
	TypeBigInt Type = 5  // 大整数（slen||magnitude，DEC-0001）
	TypeFloat  Type = 6  // float64（DEC-0502）
	TypeString Type = 7  // UTF-8 字符串
	TypeBytes  Type = 8  // 字节序列
	TypeRegExp Type = 9  // 正则表达式（RE2），占位类型
	TypeTime   Type = 10 // 时间（UNIX 毫秒有符号整数）
	TypeDict   Type = 11 // 有序字典，占位类型
	TypeSlice  Type = 12 // 值切片
	TypeCode   Type = 13 // 编译后指令序列字面值（字节序列）
	TypeObject Type = 14 // 对象/模块，占位类型
)

// String 返回类型名称。
func (t Type) String() string {
	switch t {
	case TypeNil:
		return "Nil"
	case TypeBool:
		return "Bool"
	case TypeByte:
		return "Byte"
	case TypeRune:
		return "Rune"
	case TypeInt:
		return "Int"
	case TypeBigInt:
		return "BigInt"
	case TypeFloat:
		return "Float"
	case TypeString:
		return "String"
	case TypeBytes:
		return "Bytes"
	case TypeRegExp:
		return "RegExp"
	case TypeTime:
		return "Time"
	case TypeDict:
		return "Dict"
	case TypeSlice:
		return "Slice"
	case TypeCode:
		return "Code"
	case TypeObject:
		return "Object"
	default:
		return fmt.Sprintf("Type(%d)", t)
	}
}

// Value 是 VM 栈和实参区中的类型值容器。
// 内部以 data any 承载 Go 原生值，类型标签由 typ 区分。
// 参考：docs/proposal/10.Script-System.md §3
type Value struct {
	typ  Type
	data any
}

// Typ 返回值的类型标签。
func (v Value) Typ() Type { return v.typ }

// IsNil 返回值是否为 Nil 类型。
func (v Value) IsNil() bool { return v.typ == TypeNil }

// ─── 构造函数 ─────────────────────────────────────────────────────────────────

// NilValue 构造 Nil 值。
func NilValue() Value { return Value{typ: TypeNil} }

// BoolValue 构造 Bool 值。
func BoolValue(b bool) Value { return Value{typ: TypeBool, data: b} }

// ByteValue 构造 Byte 值。
func ByteValue(b byte) Value { return Value{typ: TypeByte, data: b} }

// RuneValue 构造 Rune 值（Unicode 码点）。
func RuneValue(r rune) Value { return Value{typ: TypeRune, data: r} }

// IntValue 构造 Int（int64）值。
func IntValue(n int64) Value { return Value{typ: TypeInt, data: n} }

// FloatValue 构造 Float（float64）值。
// 允许携带运算产生的 NaN/Inf（异常浮点保留，DEC-0502）。
// 来自字面量的浮点请使用 FloatLiteralValue。
func FloatValue(f float64) Value { return Value{typ: TypeFloat, data: f} }

// FloatLiteralValue 构造来自字面量的 Float 值，拒绝 NaN/Inf（DEC-0502）。
// 脚本字节码中 FLOAT_LIT 指令必须使用此函数确保字面量合法性。
func FloatLiteralValue(f float64) (Value, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Value{}, ErrInvalidFloat
	}
	return FloatValue(f), nil
}

// StringValue 构造 String 值（UTF-8）。
func StringValue(s string) Value { return Value{typ: TypeString, data: s} }

// BytesValue 构造 Bytes 值（字节序列），复制输入切片。
func BytesValue(b []byte) Value {
	buf := make([]byte, len(b))
	copy(buf, b)
	return Value{typ: TypeBytes, data: buf}
}

// SliceValue 构造 Slice 值（值切片），复制输入切片。
func SliceValue(vs []Value) Value {
	cp := make([]Value, len(vs))
	copy(cp, vs)
	return Value{typ: TypeSlice, data: cp}
}

// CodeValue 构造 Code 值（编译后指令序列），复制输入字节切片。
func CodeValue(b []byte) Value {
	buf := make([]byte, len(b))
	copy(buf, b)
	return Value{typ: TypeCode, data: buf}
}

// TimeValue 构造 Time 值（UNIX 毫秒有符号整数）。
func TimeValue(ms int64) Value { return Value{typ: TypeTime, data: ms} }

// ─── 访问函数 ─────────────────────────────────────────────────────────────────

// AsBool 返回 Bool 值，类型不符时返回 ErrTypeMismatch。
func (v Value) AsBool() (bool, error) {
	if v.typ != TypeBool {
		return false, fmt.Errorf("%w: expected Bool, got %s", ErrTypeMismatch, v.typ)
	}
	return v.data.(bool), nil
}

// AsByte 返回 Byte 值，类型不符时返回 ErrTypeMismatch。
func (v Value) AsByte() (byte, error) {
	if v.typ != TypeByte {
		return 0, fmt.Errorf("%w: expected Byte, got %s", ErrTypeMismatch, v.typ)
	}
	return v.data.(byte), nil
}

// AsRune 返回 Rune 值，类型不符时返回 ErrTypeMismatch。
func (v Value) AsRune() (rune, error) {
	if v.typ != TypeRune {
		return 0, fmt.Errorf("%w: expected Rune, got %s", ErrTypeMismatch, v.typ)
	}
	return v.data.(rune), nil
}

// AsInt 返回 Int（int64）值，类型不符时返回 ErrTypeMismatch。
func (v Value) AsInt() (int64, error) {
	if v.typ != TypeInt {
		return 0, fmt.Errorf("%w: expected Int, got %s", ErrTypeMismatch, v.typ)
	}
	return v.data.(int64), nil
}

// AsFloat 返回 Float（float64）值，类型不符时返回 ErrTypeMismatch。
func (v Value) AsFloat() (float64, error) {
	if v.typ != TypeFloat {
		return 0, fmt.Errorf("%w: expected Float, got %s", ErrTypeMismatch, v.typ)
	}
	return v.data.(float64), nil
}

// AsString 返回 String 值，类型不符时返回 ErrTypeMismatch。
func (v Value) AsString() (string, error) {
	if v.typ != TypeString {
		return "", fmt.Errorf("%w: expected String, got %s", ErrTypeMismatch, v.typ)
	}
	return v.data.(string), nil
}

// AsBytes 返回 Bytes 值的副本，类型不符时返回 ErrTypeMismatch。
func (v Value) AsBytes() ([]byte, error) {
	if v.typ != TypeBytes {
		return nil, fmt.Errorf("%w: expected Bytes, got %s", ErrTypeMismatch, v.typ)
	}
	b := v.data.([]byte)
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp, nil
}

// AsSlice 返回 Slice 值的副本，类型不符时返回 ErrTypeMismatch。
func (v Value) AsSlice() ([]Value, error) {
	if v.typ != TypeSlice {
		return nil, fmt.Errorf("%w: expected Slice, got %s", ErrTypeMismatch, v.typ)
	}
	s := v.data.([]Value)
	cp := make([]Value, len(s))
	copy(cp, s)
	return cp, nil
}

// AsTime 返回 Time（UNIX 毫秒）值，类型不符时返回 ErrTypeMismatch。
func (v Value) AsTime() (int64, error) {
	if v.typ != TypeTime {
		return 0, fmt.Errorf("%w: expected Time, got %s", ErrTypeMismatch, v.typ)
	}
	return v.data.(int64), nil
}

// AsCode 返回 Code 值的副本，类型不符时返回 ErrTypeMismatch。
func (v Value) AsCode() ([]byte, error) {
	if v.typ != TypeCode {
		return nil, fmt.Errorf("%w: expected Code, got %s", ErrTypeMismatch, v.typ)
	}
	b := v.data.([]byte)
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp, nil
}

// ─── 大小与工具 ───────────────────────────────────────────────────────────────

// ByteSize 返回值的字节大小，用于 MaxStackItem 上限检查。
// 固定类型返回其固定大小；变长类型返回数据字节数；复合类型递归累计。
func (v Value) ByteSize() int {
	switch v.typ {
	case TypeNil:
		return 0
	case TypeBool, TypeByte:
		return 1
	case TypeRune:
		return 4
	case TypeInt, TypeFloat, TypeTime:
		return 8
	case TypeString:
		if s, ok := v.data.(string); ok {
			return len(s)
		}
		return 0
	case TypeBytes, TypeCode:
		if b, ok := v.data.([]byte); ok {
			return len(b)
		}
		return 0
	case TypeSlice:
		if s, ok := v.data.([]Value); ok {
			total := 0
			for _, item := range s {
				total += item.ByteSize()
			}
			return total
		}
		return 0
	default:
		// BigInt, RegExp, Dict, Object：保守估计，实际大小由子系统决定
		return 8
	}
}

// Equal 判断两个值是否相等（类型+值均相同）。
// 浮点遵循 DEC-0502：NaN != NaN；+0.0 == -0.0（数值相等）。
func (v Value) Equal(other Value) bool {
	if v.typ != other.typ {
		return false
	}
	switch v.typ {
	case TypeNil:
		return true
	case TypeBool:
		return v.data.(bool) == other.data.(bool)
	case TypeByte:
		return v.data.(byte) == other.data.(byte)
	case TypeRune:
		return v.data.(rune) == other.data.(rune)
	case TypeInt:
		return v.data.(int64) == other.data.(int64)
	case TypeFloat:
		a, b := v.data.(float64), other.data.(float64)
		// NaN != NaN（DEC-0502）
		if math.IsNaN(a) || math.IsNaN(b) {
			return false
		}
		// +0.0 == -0.0（数值相等）
		return a == b
	case TypeString:
		return v.data.(string) == other.data.(string)
	case TypeBytes, TypeCode:
		a, b := v.data.([]byte), other.data.([]byte)
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	case TypeTime:
		return v.data.(int64) == other.data.(int64)
	default:
		// BigInt, RegExp, Dict, Slice, Object：暂不支持深度相等
		return false
	}
}
