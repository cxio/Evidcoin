package script

import "testing"

// TestExecModeIsPublic 测试模式判断。
func TestExecModeIsPublic(t *testing.T) {
	if !ModePublic.IsPublic() {
		t.Error("ModePublic.IsPublic() should be true")
	}
	if ModePrivate.IsPublic() {
		t.Error("ModePrivate.IsPublic() should be false")
	}
}

// TestCheckPublicSafety 测试公共路径安全检查。
func TestCheckPublicSafety(t *testing.T) {
	cases := []struct {
		name    string
		mode    ExecMode
		op      Opcode
		wantErr error
	}{
		// 私有路径：任何 opcode 均通过
		{"private SCRIPT", ModePrivate, SCRIPT, nil},
		{"private SYS_TIME", ModePrivate, SYS_TIME, nil},
		{"private EXT_PRIV", ModePrivate, EXT_PRIV, nil},
		// 公共路径：合法指令通过
		{"public NIL", ModePublic, NIL, nil},
		{"public END", ModePublic, END, nil},
		{"public SYS_CHKPASS", ModePublic, SYS_CHKPASS, nil},
		// 公共路径：禁用指令触达 ScriptError
		{"public SCRIPT", ModePublic, SCRIPT, ErrDisabledInPublic},
		{"public VALUE", ModePublic, VALUE, ErrDisabledInPublic},
		{"public EVAL", ModePublic, EVAL, ErrDisabledInPublic},
		{"public INOUT", ModePublic, INOUT, ErrDisabledInPublic},
		// 公共路径：SYS_TIME 触达 ScriptError
		{"public SYS_TIME", ModePublic, SYS_TIME, ErrSysTimeInPublic},
		// 公共路径：EXT_PRIV 触达 ScriptError
		{"public EXT_PRIV", ModePublic, EXT_PRIV, ErrExtPrivInPublic},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkPublicSafety(tc.mode, tc.op)
			if tc.wantErr == nil && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tc.wantErr != nil && err != tc.wantErr {
				t.Errorf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}
