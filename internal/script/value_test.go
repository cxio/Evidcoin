package script

import (
	"math"
	"testing"
)

// TestValueTypes 测试值类型构造和类型标签正确性。
func TestValueTypes(t *testing.T) {
	cases := []struct {
		name string
		v    Value
		typ  Type
	}{
		{"Nil", NilValue(), TypeNil},
		{"Bool true", BoolValue(true), TypeBool},
		{"Bool false", BoolValue(false), TypeBool},
		{"Byte", ByteValue(0xff), TypeByte},
		{"Rune", RuneValue('中'), TypeRune},
		{"Int", IntValue(-1), TypeInt},
		{"Float", FloatValue(3.14), TypeFloat},
		{"String", StringValue("hello"), TypeString},
		{"Bytes", BytesValue([]byte{1, 2, 3}), TypeBytes},
		{"Slice", SliceValue([]Value{NilValue()}), TypeSlice},
		{"Code", CodeValue([]byte{0x01}), TypeCode},
		{"Time", TimeValue(1000), TypeTime},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.v.Typ(); got != tc.typ {
				t.Errorf("Type() = %v, want %v", got, tc.typ)
			}
		})
	}
}

// TestValueIsNil 测试 IsNil。
func TestValueIsNil(t *testing.T) {
	if !NilValue().IsNil() {
		t.Error("NilValue().IsNil() should be true")
	}
	if BoolValue(true).IsNil() {
		t.Error("BoolValue(true).IsNil() should be false")
	}
}

// TestValueAccessors 测试类型访问函数正常路径。
func TestValueAccessors(t *testing.T) {
	t.Run("Bool", func(t *testing.T) {
		v := BoolValue(true)
		b, err := v.AsBool()
		if err != nil || b != true {
			t.Errorf("AsBool() = %v, %v", b, err)
		}
	})
	t.Run("Byte", func(t *testing.T) {
		v := ByteValue(42)
		b, err := v.AsByte()
		if err != nil || b != 42 {
			t.Errorf("AsByte() = %v, %v", b, err)
		}
	})
	t.Run("Rune", func(t *testing.T) {
		v := RuneValue('A')
		r, err := v.AsRune()
		if err != nil || r != 'A' {
			t.Errorf("AsRune() = %v, %v", r, err)
		}
	})
	t.Run("Int", func(t *testing.T) {
		v := IntValue(-99)
		n, err := v.AsInt()
		if err != nil || n != -99 {
			t.Errorf("AsInt() = %v, %v", n, err)
		}
	})
	t.Run("Float", func(t *testing.T) {
		v := FloatValue(2.718)
		f, err := v.AsFloat()
		if err != nil || f != 2.718 {
			t.Errorf("AsFloat() = %v, %v", f, err)
		}
	})
	t.Run("String", func(t *testing.T) {
		v := StringValue("world")
		s, err := v.AsString()
		if err != nil || s != "world" {
			t.Errorf("AsString() = %v, %v", s, err)
		}
	})
	t.Run("Bytes", func(t *testing.T) {
		orig := []byte{0xde, 0xad}
		v := BytesValue(orig)
		got, err := v.AsBytes()
		if err != nil || len(got) != 2 || got[0] != 0xde {
			t.Errorf("AsBytes() = %v, %v", got, err)
		}
	})
	t.Run("Time", func(t *testing.T) {
		v := TimeValue(9999)
		ms, err := v.AsTime()
		if err != nil || ms != 9999 {
			t.Errorf("AsTime() = %v, %v", ms, err)
		}
	})
}

// TestValueTypeMismatch 测试类型不符时返回 ErrTypeMismatch。
func TestValueTypeMismatch(t *testing.T) {
	v := IntValue(1)
	if _, err := v.AsBool(); err == nil {
		t.Error("AsBool on Int should return error")
	}
	if _, err := v.AsFloat(); err == nil {
		t.Error("AsFloat on Int should return error")
	}
	if _, err := v.AsString(); err == nil {
		t.Error("AsString on Int should return error")
	}
	if _, err := v.AsBytes(); err == nil {
		t.Error("AsBytes on Int should return error")
	}
	if _, err := NilValue().AsInt(); err == nil {
		t.Error("AsInt on Nil should return error")
	}
}

// TestFloatLiteralValue 测试字面量 Float 拒绝 NaN/Inf。
func TestFloatLiteralValue(t *testing.T) {
	cases := []struct {
		name    string
		f       float64
		wantErr bool
	}{
		{"normal", 1.5, false},
		{"zero", 0.0, false},
		{"negative", -3.14, false},
		{"NaN", math.NaN(), true},
		{"+Inf", math.Inf(1), true},
		{"-Inf", math.Inf(-1), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := FloatLiteralValue(tc.f)
			if tc.wantErr && err == nil {
				t.Errorf("FloatLiteralValue(%v) should return error", tc.f)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("FloatLiteralValue(%v) unexpected error: %v", tc.f, err)
			}
		})
	}
}

// TestFloatSpecial 测试异常浮点保留（运算路径）和 -0.0 == +0.0（DEC-0502）。
func TestFloatSpecial(t *testing.T) {
	// 运算产生的 NaN 允许保留（FloatValue 不拒绝）
	nan := FloatValue(math.NaN())
	if nan.Typ() != TypeFloat {
		t.Error("FloatValue(NaN) should succeed")
	}
	// -0.0 == +0.0（数值相等，DEC-0502）
	pos0 := FloatValue(0.0)
	neg0 := FloatValue(math.Copysign(0, -1))
	if !pos0.Equal(neg0) {
		t.Error("+0.0 and -0.0 should be equal (DEC-0502)")
	}
	// NaN != NaN（DEC-0502）
	nan1 := FloatValue(math.NaN())
	nan2 := FloatValue(math.NaN())
	if nan1.Equal(nan2) {
		t.Error("NaN == NaN should be false (DEC-0502)")
	}
}

// TestValueByteSize 测试 ByteSize 方法。
func TestValueByteSize(t *testing.T) {
	cases := []struct {
		name string
		v    Value
		want int
	}{
		{"Nil", NilValue(), 0},
		{"Bool", BoolValue(true), 1},
		{"Byte", ByteValue(0), 1},
		{"Rune", RuneValue('A'), 4},
		{"Int", IntValue(0), 8},
		{"Float", FloatValue(0), 8},
		{"Time", TimeValue(0), 8},
		{"String len=5", StringValue("hello"), 5},
		{"Bytes len=3", BytesValue([]byte{1, 2, 3}), 3},
		{"Code len=2", CodeValue([]byte{1, 2}), 2},
		{"Slice 2 ints", SliceValue([]Value{IntValue(1), IntValue(2)}), 16},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.v.ByteSize(); got != tc.want {
				t.Errorf("ByteSize() = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestValueEqual 测试 Equal 方法。
func TestValueEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b Value
		want bool
	}{
		{"Nil==Nil", NilValue(), NilValue(), true},
		{"Bool==Bool", BoolValue(true), BoolValue(true), true},
		{"Bool!=Bool", BoolValue(true), BoolValue(false), false},
		{"Int==Int", IntValue(42), IntValue(42), true},
		{"Int!=Int", IntValue(1), IntValue(2), false},
		{"String==", StringValue("ab"), StringValue("ab"), true},
		{"String!=", StringValue("a"), StringValue("b"), false},
		{"Bytes==", BytesValue([]byte{1}), BytesValue([]byte{1}), true},
		{"Type mismatch", IntValue(1), BoolValue(true), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.a.Equal(tc.b); got != tc.want {
				t.Errorf("Equal() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestValueBytesIsolation 测试 BytesValue 和 AsBytes 复制隔离（不共享底层切片）。
func TestValueBytesIsolation(t *testing.T) {
	orig := []byte{1, 2, 3}
	v := BytesValue(orig)
	orig[0] = 99 // 修改原始切片不影响 Value
	b, _ := v.AsBytes()
	if b[0] != 1 {
		t.Error("BytesValue should copy input; modification to original should not affect value")
	}
	b[0] = 77 // 修改返回的切片不影响 Value
	b2, _ := v.AsBytes()
	if b2[0] != 1 {
		t.Error("AsBytes should return a copy; modification should not affect stored value")
	}
}
