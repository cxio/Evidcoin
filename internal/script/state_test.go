package script

import "testing"

// TestExecStateIsDone 测试 IsDone 正确区分运行中与终止态。
func TestExecStateIsDone(t *testing.T) {
	cases := []struct {
		state ExecState
		done  bool
	}{
		{StateRunning, false},
		{StatePassStop, true},
		{StateVerifyFail, true},
		{StateScriptError, true},
		{StateCostFail, true},
		{StatePrivateStop, true},
	}
	for _, tc := range cases {
		t.Run(tc.state.String(), func(t *testing.T) {
			if got := tc.state.IsDone(); got != tc.done {
				t.Errorf("IsDone() = %v, want %v", got, tc.done)
			}
		})
	}
}

// TestExecStatePassed 测试 Passed 仅在 PassStop+passState=true 时为真。
func TestExecStatePassed(t *testing.T) {
	cases := []struct {
		state     ExecState
		passState bool
		want      bool
	}{
		{StatePassStop, true, true},
		{StatePassStop, false, false},
		{StateVerifyFail, true, false},
		{StateScriptError, true, false},
		{StateCostFail, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.state.String(), func(t *testing.T) {
			if got := tc.state.Passed(tc.passState); got != tc.want {
				t.Errorf("Passed(%v) = %v, want %v", tc.passState, got, tc.want)
			}
		})
	}
}

// TestExecStateString 测试状态名称输出。
func TestExecStateString(t *testing.T) {
	if s := StateRunning.String(); s != "Running" {
		t.Errorf("StateRunning.String() = %q", s)
	}
	if s := StatePassStop.String(); s != "PassStop" {
		t.Errorf("StatePassStop.String() = %q", s)
	}
}
