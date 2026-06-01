package script

import (
	"errors"
	"testing"
)

// TestInstrConversion 测试转换指令 [67-79]。
func TestInstrConversion(t *testing.T) {
	cases := []struct {
		name      string
		op        Opcode
		input     Value
		attrParam []byte // 可选附参
		want      Value
		wantErr   bool
	}{
		// BOOL
		{"bool_nil", BOOL, NilValue(), nil, BoolValue(false), false},
		{"bool_false", BOOL, BoolValue(false), nil, BoolValue(false), false},
		{"bool_true", BOOL, BoolValue(true), nil, BoolValue(true), false},
		{"bool_int_0", BOOL, IntValue(0), nil, BoolValue(false), false},
		{"bool_int_1", BOOL, IntValue(1), nil, BoolValue(true), false},
		{"bool_empty_str", BOOL, StringValue(""), nil, BoolValue(false), false},
		{"bool_str", BOOL, StringValue("x"), nil, BoolValue(true), false},
		// BYTE_CONV
		{"byte_nil", BYTE_CONV, NilValue(), nil, ByteValue(0), false},
		{"byte_bool_true", BYTE_CONV, BoolValue(true), nil, ByteValue(1), false},
		{"byte_int_ok", BYTE_CONV, IntValue(42), nil, ByteValue(42), false},
		{"byte_int_overflow", BYTE_CONV, IntValue(256), nil, NilValue(), true},
		// RUNE_CONV
		{"rune_nil", RUNE_CONV, NilValue(), nil, RuneValue(0), false},
		{"rune_byte", RUNE_CONV, ByteValue(65), nil, RuneValue(65), false},
		{"rune_int_ok", RUNE_CONV, IntValue(0x1F600), nil, RuneValue(0x1F600), false},
		{"rune_int_overflow", RUNE_CONV, IntValue(0x200000), nil, NilValue(), true},
		// INT_CONV
		{"int_nil", INT_CONV, NilValue(), nil, IntValue(0), false},
		{"int_bool_true", INT_CONV, BoolValue(true), nil, IntValue(1), false},
		{"int_bool_false", INT_CONV, BoolValue(false), nil, IntValue(0), false},
		{"int_byte", INT_CONV, ByteValue(10), nil, IntValue(10), false},
		{"int_float", INT_CONV, FloatValue(3.9), nil, IntValue(3), false},
		{"int_string", INT_CONV, StringValue("42"), nil, IntValue(42), false},
		{"int_string_hex", INT_CONV, StringValue("0xff"), nil, IntValue(255), false},
		// FLOAT_CONV
		{"float_nil", FLOAT_CONV, NilValue(), nil, FloatValue(0.0), false},
		{"float_int", FLOAT_CONV, IntValue(42), nil, FloatValue(42.0), false},
		{"float_string_ok", FLOAT_CONV, StringValue("3.14"), nil, FloatValue(3.14), false},
		{"float_string_bad", FLOAT_CONV, StringValue("abc"), nil, NilValue(), true},
		// BYTES_CONV
		{"bytes_nil", BYTES_CONV, NilValue(), nil, BytesValue(nil), false},
		{"bytes_byte", BYTES_CONV, ByteValue(0xAB), nil, BytesValue([]byte{0xAB}), false},
		{"bytes_int", BYTES_CONV, IntValue(1), nil, BytesValue([]byte{0, 0, 0, 0, 0, 0, 0, 1}), false},
		{"bytes_string", BYTES_CONV, StringValue("hi"), nil, BytesValue([]byte("hi")), false},
		{"bytes_code_fail", BYTES_CONV, CodeValue([]byte{1, 2}), nil, NilValue(), true},
		// RUNES_CONV
		{"runes_nil", RUNES_CONV, NilValue(), nil, SliceValue([]Value{}), false},
		{"runes_rune", RUNES_CONV, RuneValue('A'), nil, SliceValue([]Value{RuneValue('A')}), false},
		{"runes_string", RUNES_CONV, StringValue("ab"), nil, SliceValue([]Value{RuneValue('a'), RuneValue('b')}), false},
		// TIME_CONV
		{"time_nil", TIME_CONV, NilValue(), nil, TimeValue(0), false},
		{"time_int", TIME_CONV, IntValue(1000), nil, TimeValue(1000), false},
		// STRING_CONV
		{"string_int", STRING_CONV, IntValue(255), nil, StringValue("255"), false},
		{"string_int_hex", STRING_CONV, IntValue(255), []byte{16}, StringValue("ff"), false},
		{"string_bool", STRING_CONV, BoolValue(true), nil, StringValue("true"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVM()
			vm.stack.Push(tc.input)
			var frame InstrFrame
			frame.Op = tc.op
			if tc.attrParam != nil {
				frame.AttrParams = [][]byte{tc.attrParam}
			}
			state := vm.Run([]InstrFrame{frame})

			if tc.wantErr {
				if state == StatePassStop {
					t.Fatalf("expected error state, got PassStop")
				}
				return
			}
			if state != StatePassStop {
				t.Fatalf("expected PassStop, got %v", state)
			}
			top, err := vm.stack.Pop()
			if err != nil {
				t.Fatalf("pop error: %v", err)
			}
			if !top.Equal(tc.want) {
				// 对 Slice 类型做特殊检查（Equal 对 Slice 未实现深度比较）
				if top.Typ() == TypeSlice && tc.want.Typ() == TypeSlice {
					gotSl, _ := top.AsSlice()
					wantSl, _ := tc.want.AsSlice()
					if len(gotSl) != len(wantSl) {
						t.Fatalf("slice len: want %d, got %d", len(wantSl), len(gotSl))
					}
					for i := range gotSl {
						if !gotSl[i].Equal(wantSl[i]) {
							t.Fatalf("slice[%d]: want %v, got %v", i, wantSl[i], gotSl[i])
						}
					}
					return
				}
				t.Fatalf("want %v, got %v", tc.want, top)
			}
		})
	}

	t.Run("regexp_conv_ok", func(t *testing.T) {
		vm := NewVM()
		vm.stack.Push(StringValue(`^\d+$`))
		vm.Run([]InstrFrame{{Op: REGEXP_CONV}})
		top, err := vm.stack.Pop()
		if err != nil {
			t.Fatalf("pop error: %v", err)
		}
		if top.Typ() != TypeRegExp {
			t.Fatalf("want TypeRegExp, got %s", top.Typ())
		}
	})

	t.Run("regexp_conv_fail", func(t *testing.T) {
		vm := NewVM()
		vm.stack.Push(StringValue(`[invalid`))
		state := vm.Run([]InstrFrame{{Op: REGEXP_CONV}})
		if state == StatePassStop {
			t.Fatal("expected error state for invalid regex")
		}
	})

	t.Run("bigint_conv_placeholder", func(t *testing.T) {
		vm := NewVM()
		vm.stack.Push(IntValue(42))
		vm.Run([]InstrFrame{{Op: BIGINT_CONV}})
		top, err := vm.stack.Pop()
		if err != nil {
			t.Fatalf("pop error: %v", err)
		}
		if top.Typ() != TypeNil {
			t.Fatalf("want Nil (placeholder), got %s", top.Typ())
		}
	})

	t.Run("anys_mode0", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{IntValue(1), IntValue(2)})
		vm.stack.Push(sl)
		vm.Run([]InstrFrame{{Op: ANYS, AttrParams: [][]byte{{0}}}})
		top, _ := vm.stack.Pop()
		got, _ := top.AsSlice()
		if len(got) != 2 {
			t.Fatalf("want 2, got %d", len(got))
		}
	})

	t.Run("anys_mode1_bool", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{NilValue(), IntValue(1)})
		vm.stack.Push(sl)
		vm.Run([]InstrFrame{{Op: ANYS, AttrParams: [][]byte{{1}}}})
		top, _ := vm.stack.Pop()
		got, _ := top.AsSlice()
		if len(got) != 2 {
			t.Fatalf("want 2, got %d", len(got))
		}
		b0, _ := got[0].AsBool()
		b1, _ := got[1].AsBool()
		if b0 != false || b1 != true {
			t.Fatalf("want [false,true], got [%v,%v]", b0, b1)
		}
	})

	t.Run("float_nan_rejected", func(t *testing.T) {
		// FLOAT_CONV: 输入字符串 "NaN" 应被拒绝（返回 ErrInvalidFloat）
		vm := NewVM()
		vm.stack.Push(StringValue("NaN"))
		state := vm.Run([]InstrFrame{{Op: FLOAT_CONV}})
		if state == StatePassStop {
			t.Fatal("expected error state for NaN float")
		}
	})

	t.Run("time_conv_string", func(t *testing.T) {
		vm := NewVM()
		vm.stack.Push(StringValue("2024-01-01T00:00:00Z"))
		state := vm.Run([]InstrFrame{{Op: TIME_CONV}})
		if state != StatePassStop {
			t.Fatalf("expected PassStop, got %v", state)
		}
		top, _ := vm.stack.Pop()
		if top.Typ() != TypeTime {
			t.Fatalf("want TypeTime, got %s", top.Typ())
		}
	})

	_ = errors.New // 避免未使用 import
}
